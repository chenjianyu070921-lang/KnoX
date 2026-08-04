package svc

import (
	"database/sql"

	_ "github.com/chenjianyu070921-lang/KnoX/pkg/clickhouse" // 注册 clickhouse driver

	"github.com/chenjianyu070921-lang/KnoX/cmd/reporter/internal/config"
	"github.com/chenjianyu070921-lang/KnoX/internal/analytics"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config    config.Config
	Analytics *analytics.Analytics
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.ClickhouseDSN == "" {
		logx.Infof("[Reporter] CLICKHOUSE_DSN 未配置，报表接口将返回空数据")
		return &ServiceContext{Config: c}
	}
	db, err := sql.Open("clickhouse", c.ClickhouseDSN)
	if err != nil {
		logx.Errorf("[Reporter] ClickHouse 连接失败: %v，报表接口将返回空数据", err)
		return &ServiceContext{Config: c}
	}
	if err := db.Ping(); err != nil {
		logx.Errorf("[Reporter] ClickHouse Ping 失败: %v，报表接口将返回空数据", err)
	}
	return &ServiceContext{
		Config:    c,
		Analytics: analytics.New(db),
	}
}
