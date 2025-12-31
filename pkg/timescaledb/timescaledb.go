package timescaledb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Config TimescaleDB配置
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string // disable, require, verify-ca, verify-full
}

// Client TimescaleDB客户端
type Client struct {
	db     *sql.DB
	config Config
}

// NewClient 创建TimescaleDB客户端
func NewClient(config Config) *Client {
	// 设置默认值
	if config.Host == "" {
		config.Host = "localhost"
	}
	if config.Port == 0 {
		config.Port = 5432
	}
	if config.SSLMode == "" {
		config.SSLMode = "disable"
	}

	return &Client{
		config: config,
	}
}

// Connect 连接到TimescaleDB
func (c *Client) Connect() error {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.config.Host,
		c.config.Port,
		c.config.User,
		c.config.Password,
		c.config.Database,
		c.config.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	c.db = db
	log.Printf("Connected to TimescaleDB at %s:%d", c.config.Host, c.config.Port)
	return nil
}

// Close 关闭数据库连接
func (c *Client) Close() error {
	if c.db != nil {
		log.Println("Closing TimescaleDB connection...")
		return c.db.Close()
	}
	return nil
}

// InitTable 初始化数据表（创建hypertable）
func (c *Client) InitTable(tableName string) error {
	ctx := context.Background()

	// 创建表
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			time        TIMESTAMPTZ NOT NULL,
			source_addr VARCHAR(50),
			data_size   INTEGER,
			raw_data    BYTEA,
			created_at  TIMESTAMPTZ DEFAULT NOW()
		)
	`, tableName)

	if _, err := c.db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// 创建hypertable（如果还不是hypertable）
	createHypertableSQL := fmt.Sprintf(`
		SELECT create_hypertable('%s', 'time', if_not_exists => TRUE)
	`, tableName)

	if _, err := c.db.ExecContext(ctx, createHypertableSQL); err != nil {
		// 如果表已经是hypertable，会返回错误，可以忽略
		log.Printf("Note: %v (this is normal if table already exists)", err)
	}

	log.Printf("Table '%s' initialized as hypertable", tableName)
	return nil
}

// InsertBinaryData 插入二进制数据
func (c *Client) InsertBinaryData(tableName string, sourceAddr string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (time, source_addr, data_size, raw_data)
		VALUES ($1, $2, $3, $4)
	`, tableName)

	_, err := c.db.ExecContext(
		ctx,
		insertSQL,
		time.Now(),
		sourceAddr,
		len(data),
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to insert data: %w", err)
	}

	return nil
}

// StartBatchWriter 启动批量写入协程
func (c *Client) StartBatchWriter(tableName string, dataChan <-chan DataEntry, batchSize int, flushInterval time.Duration) {
	go c.batchWriter(tableName, dataChan, batchSize, flushInterval)
}

// DataEntry 数据条目
type DataEntry struct {
	SourceAddr string
	Data       []byte
}

// batchWriter 批量写入逻辑
func (c *Client) batchWriter(tableName string, dataChan <-chan DataEntry, batchSize int, flushInterval time.Duration) {
	batch := make([]DataEntry, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		if err := c.batchInsert(tableName, batch); err != nil {
			log.Printf("Failed to batch insert: %v", err)
		} else {
			log.Printf("Batch inserted %d records to TimescaleDB", len(batch))
		}
		batch = batch[:0]
	}

	for {
		select {
		case entry, ok := <-dataChan:
			if !ok {
				// Channel关闭，刷新剩余数据
				flush()
				log.Println("TimescaleDB batch writer stopped")
				return
			}

			batch = append(batch, entry)
			if len(batch) >= batchSize {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}

// batchInsert 批量插入数据
func (c *Client) batchInsert(tableName string, entries []DataEntry) error {
	if len(entries) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (time, source_addr, data_size, raw_data)
		VALUES ($1, $2, $3, $4)
	`, tableName)

	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	timestamp := time.Now()
	for _, entry := range entries {
		_, err := stmt.ExecContext(
			ctx,
			timestamp,
			entry.SourceAddr,
			len(entry.Data),
			entry.Data,
		)
		if err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// QueryRecentData 查询最近的数据
func (c *Client) QueryRecentData(tableName string, limit int) ([]DataEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	querySQL := fmt.Sprintf(`
		SELECT source_addr, raw_data
		FROM %s
		ORDER BY time DESC
		LIMIT $1
	`, tableName)

	rows, err := c.db.QueryContext(ctx, querySQL, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}
	defer rows.Close()

	var results []DataEntry
	for rows.Next() {
		var entry DataEntry
		if err := rows.Scan(&entry.SourceAddr, &entry.Data); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}
