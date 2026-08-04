// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/yourname/know/cmd/gateway/internal/config"
	"github.com/yourname/know/cmd/gateway/internal/handler"
	"github.com/yourname/know/cmd/gateway/internal/svc"
	"github.com/yourname/know/internal/errcode"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/doc.yaml", "config file")

func main() {

	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 环境变量优先，避免密钥写进配置文件进 git
	if v := os.Getenv("KNOX_QINIU_ACCESS_KEY"); v != "" {
		c.Qiniu.AccessKey = v
	}
	if v := os.Getenv("KNOX_QINIU_SECRET_KEY"); v != "" {
		c.Qiniu.SecretKey = v
	}
	if v := os.Getenv("KNOX_ARK_API_KEY"); v != "" {
		c.ARK.APIKey = v
	}
	if v := os.Getenv("KNOX_MYSQL_DSN"); v != "" {
		c.Mysql.DSN = v
	}
	if v := os.Getenv("KNOX_CLICKHOUSE_PASSWORD"); v != "" {
		c.ClickHouse.Password = v
	}

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)
	// 注册自定义错误处理器：将 BizError 转成统一格式
	httpx.SetErrorHandler(func(err error) (int, interface{}) {
		if bizErr, ok := err.(*errcode.BizError); ok {
			status := http.StatusOK
			switch bizErr.Code {
			case errcode.BadRequest, errcode.InvalidParam:
				status = http.StatusBadRequest
			case errcode.RateLimitExceeded:
				status = http.StatusTooManyRequests
			case errcode.CircuitBreakerOpen:
				status = http.StatusServiceUnavailable
			}
			return status, map[string]interface{}{
				"code":    bizErr.Code,
				"message": bizErr.Message,
			}
		}
		// 非业务错误，保留原始错误信息
		return http.StatusInternalServerError, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		}
	})
	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
