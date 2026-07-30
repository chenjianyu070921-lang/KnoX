// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package main

import (
	"flag"
	"fmt"
	"net/http"

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

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)
	// 注册自定义错误处理器：将 BizError 转成统一格式
	httpx.SetErrorHandler(func(err error) (int, interface{}) {
		if bizErr, ok := err.(*errcode.BizError); ok {
			return http.StatusOK, map[string]interface{}{
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
