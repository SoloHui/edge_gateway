# Gateway TimescaleDB 示例

这个示例展示了如何使用 TimescaleDB 模块存储 UDP 传入的二进制数据。

## 功能特性

- UDP 数据接收
- TimescaleDB 时序数据存储
- 批量写入优化
- 优雅关闭

## 配置说明

在 `main.go` 中修改以下配置：

```go
// UDP配置
udpAddr = "127.0.0.2:8888"

// TimescaleDB配置
dbHost     = "localhost"
dbPort     = 5432
dbUser     = "postgres"
dbPassword = "your_password"
dbName     = "ship_data"
tableName  = "udp_binary_data"

// 批量写入配置
batchSize     = 100              // 批量大小
flushInterval = 5 * time.Second  // 刷新间隔
```

## 数据库准备

### 1. 安装 TimescaleDB

参考官方文档：https://docs.timescale.com/install/latest/

### 2. 创建数据库

```sql
CREATE DATABASE ship_data;
```

### 3. 启用 TimescaleDB 扩展

```sql
\c ship_data
CREATE EXTENSION IF NOT EXISTS timescaledb;
```

### 4. 表结构

程序会自动创建表结构和 hypertable：

```sql
CREATE TABLE udp_binary_data (
    time        TIMESTAMPTZ NOT NULL,
    source_addr VARCHAR(50),
    data_size   INTEGER,
    raw_data    BYTEA,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

SELECT create_hypertable('udp_binary_data', 'time');
```

## 运行

```bash
# 编译
go build -o gateway_timescale.exe

# 运行
./gateway_timescale.exe
```

## 数据查询示例

```sql
-- 查询最近10条数据
SELECT time, source_addr, data_size, created_at
FROM udp_binary_data
ORDER BY time DESC
LIMIT 10;

-- 查询特定时间范围的数据
SELECT time, source_addr, data_size
FROM udp_binary_data
WHERE time > NOW() - INTERVAL '1 hour'
ORDER BY time DESC;

-- 统计每小时的数据量
SELECT time_bucket('1 hour', time) AS hour,
       COUNT(*) as count,
       SUM(data_size) as total_bytes
FROM udp_binary_data
WHERE time > NOW() - INTERVAL '24 hours'
GROUP BY hour
ORDER BY hour DESC;
```

## 性能优化

1. **批量写入**：通过 `batchSize` 和 `flushInterval` 控制批量写入策略
2. **连接池**：自动配置数据库连接池参数
3. **异步处理**：使用 channel 解耦 UDP 接收和数据库写入

## 依赖

需要添加 PostgreSQL 驱动：

```bash
go get github.com/lib/pq
```
