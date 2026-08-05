package svc

import (
	"database/sql"
	"log"

	_ "github.com/yourname/know/pkg/clickhouse" // 注册 clickhouse driver

	"github.com/yourname/know/cmd/reporter/internal/config"
	"github.com/yourname/know/internal/analytics"
)

type ServiceContext struct {
	Config    config.Config
	Analytics *analytics.Analytics
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.ClickhouseDSN == "" {
		log.Println("[Reporter] CLICKHOUSE_DSN 未配置，报表接口将返回空数据")
		return &ServiceContext{Config: c}
	}
	db, err := sql.Open("clickhouse", c.ClickhouseDSN)
	if err != nil {
		log.Printf("[Reporter] ClickHouse 连接失败: %v，报表接口将返回空数据", err)
		return &ServiceContext{Config: c}
	}
	if err := db.Ping(); err != nil {
		log.Printf("[Reporter] ClickHouse Ping 失败: %v，报表接口将返回空数据", err)
	}
	return &ServiceContext{
		Config:    c,
		Analytics: analytics.New(db),
	}
}
