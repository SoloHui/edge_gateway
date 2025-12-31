package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"crossocean.com/edge_gateway/pkg/timescaledb"
	"crossocean.com/edge_gateway/pkg/udp"
)

const (
	// UDP配置
	udpAddr = "127.0.0.2:8888"

	// TimescaleDB配置
	dbHost     = "192.168.3.169"
	dbPort     = 5432
	dbUser     = "postgres"
	dbPassword = "ipac1234"
	dbName     = "ship_data"
	tableName  = "udp_binary_data"

	// 批量写入配置
	batchSize     = 100             // 批量大小
	flushInterval = 5 * time.Second // 刷新间隔
)

func main() {
	log.Println("Starting Gateway TimescaleDB Service...")

	// 创建数据通道
	rawDataChan := make(chan []byte, 100)
	dbDataChan := make(chan timescaledb.DataEntry, 200)

	// 创建并连接TimescaleDB客户端
	tsClient := timescaledb.NewClient(timescaledb.Config{
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPassword,
		Database: dbName,
		SSLMode:  "disable",
	})

	if err := tsClient.Connect(); err != nil {
		log.Fatalf("Failed to connect TimescaleDB: %v", err)
	}
	defer tsClient.Close()

	// 初始化数据表
	if err := tsClient.InitTable(tableName); err != nil {
		log.Fatalf("Failed to initialize table: %v", err)
	}

	// 启动批量写入协程
	tsClient.StartBatchWriter(tableName, dbDataChan, batchSize, flushInterval)

	// 启动数据转换协程（将UDP数据转换为TimescaleDB数据格式）
	go dataConverter(rawDataChan, dbDataChan)

	// 创建并启动UDP监听器
	udpListener := udp.NewListener(udpAddr, rawDataChan)
	if err := udpListener.Start(); err != nil {
		log.Fatalf("Failed to start UDP listener: %v", err)
	}
	defer udpListener.Close()

	log.Printf("Gateway service started, listening on UDP %s", udpAddr)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down service...")

	// 关闭通道，触发优雅关闭
	close(rawDataChan)
	close(dbDataChan)

	// 等待处理完成
	time.Sleep(2 * time.Second)

	log.Println("Service stopped")
}

// dataConverter 将UDP原始数据转换为TimescaleDB数据格式
func dataConverter(rawDataChan <-chan []byte, dbDataChan chan<- timescaledb.DataEntry) {
	for data := range rawDataChan {
		entry := timescaledb.DataEntry{
			SourceAddr: udpAddr, // 可以根据实际情况从UDP包中获取源地址
			Data:       data,
		}

		select {
		case dbDataChan <- entry:
			// 数据发送成功
		default:
			log.Println("Warning: dbDataChan is full, dropping data")
		}
	}
	log.Println("Data converter stopped")
}
