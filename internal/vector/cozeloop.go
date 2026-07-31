package vector

import (
	"sync"

	ccb "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"
)

// callback/main.go 里新增

var (
	loopOnce    sync.Once
	loopHandler callbacks.Handler
	loopClient  *cozeloop.Client
)

func InitCozeLoopHandler() callbacks.Handler {
	loopOnce.Do(func() {
		// cozeloop.NewClient() 返回 Coze Loop 的 SDK 客户端
		// ccb.NewLoopHandler(client) 将其包装为 Eino callbacks.Handler
		client, err := cozeloop.NewClient()
		if err != nil {
			panic(err)
		}
		loopClient = &client
		loopHandler = ccb.NewLoopHandler(client)
	})
	return loopHandler
}
