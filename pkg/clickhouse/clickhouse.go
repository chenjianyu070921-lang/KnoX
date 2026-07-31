package clickhouse

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// Config ClickHouse 连接配置
type Config struct {
	Addr     string // host:port，默认 127.0.0.1:9000
	Database string // 默认 knox
	Username string // 默认 default
	Password string // 默认空
	Debug    bool   // 是否开启日志
}

func (c *Config) dsn() string {
	addr := c.Addr
	if addr == "" {
		addr = "127.0.0.1:9000"
	}
	database := c.Database
	if database == "" {
		database = "knox"
	}
	username := c.Username
	if username == "" {
		username = "default"
	}
	return fmt.Sprintf("clickhouse://%s:%s@%s/%s?dial_timeout=5s&compress=lz4",
		username, c.Password, addr, database)
}

var (
	once   sync.Once
	conn   *sql.DB
	initErr error
)

// Init 初始化 ClickHouse 连接（单例）
func Init(cfg Config) *sql.DB {
	once.Do(func() {
		var err error
		conn, err = sql.Open("clickhouse", cfg.dsn())
		if err != nil {
			initErr = err
			return
		}
		conn.SetMaxOpenConns(20)
		conn.SetMaxIdleConns(5)
		conn.SetConnMaxLifetime(time.Hour)

		// 启动时探测连通性
		if err = conn.Ping(); err != nil {
			conn.Close()
			conn = nil
			initErr = fmt.Errorf("clickhouse ping failed: %w", err)
			return
		}
	})
	return conn
}

// GetDB 获取连接，如果未初始化则返回 nil
func GetDB() *sql.DB {
	return conn
}

// Health 健康检查
func Health() error {
	if conn == nil {
		return initErr
	}
	return conn.Ping()
}
