package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yourname/know/cmd/reporter/internal/config"
	"github.com/yourname/know/cmd/reporter/internal/handler"
	"github.com/yourname/know/cmd/reporter/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "cmd/reporter/etc/reporter.yaml", "config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 环境变量覆盖，避免密钥写进文件
	if v := os.Getenv("KNOX_CLICKHOUSE_DSN"); v != "" {
		c.ClickhouseDSN = v
	}

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterRoutes(server, ctx)

	fmt.Printf("Starting Reporter at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
