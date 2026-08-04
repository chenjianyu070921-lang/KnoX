// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"context"

	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/config"
	"github.com/chenjianyu070921-lang/KnoX/internal/agent"
	"github.com/chenjianyu070921-lang/KnoX/internal/analytics"
	"github.com/chenjianyu070921-lang/KnoX/internal/model"
	"github.com/chenjianyu070921-lang/KnoX/internal/repository"
	"github.com/chenjianyu070921-lang/KnoX/internal/session"
	"github.com/chenjianyu070921-lang/KnoX/internal/vector"
	"github.com/chenjianyu070921-lang/KnoX/pkg/clickhouse"
	"github.com/chenjianyu070921-lang/KnoX/pkg/database"
	"github.com/chenjianyu070921-lang/KnoX/pkg/redisx"
	"github.com/chenjianyu070921-lang/KnoX/pkg/redisx/distlock"

	"github.com/IBM/sarama"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/tool"
	"github.com/redis/go-redis/v9"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/zeromicro/go-zero/core/logx"
	gozeredis "github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config        config.Config
	DB            *gorm.DB
	Redis         *redis.Client    // go-redis，供 session / distlock / 业务锁
	GoZeroRedis   *gozeredis.Redis // go-zero 原生 Redis，供 PeriodLimit 限流
	KafkaProducer sarama.SyncProducer
	ReActAgent    *agent.ReActAgent
	ChatModel     *ark.ChatModel
	SessionStore  *session.Store
	Lock          *distlock.DistLock
	DocRepo       *repository.DocumentRepository
	Analytics     *analytics.Analytics // ClickHouse 统计埋点（nil 时降级为空操作）
}

func NewServiceContext(c config.Config) *ServiceContext {
	ctx := context.Background()
	//1.连接mysql
	db := database.MysqlInit(c.Mysql.DSN)
	if err := db.AutoMigrate(&model.Document{}); err != nil {
		panic("failed to auto migrate: " + err.Error())
	}
	//2.连接redis
	client := redisx.GetClient(c.Redis.Addr)
	if err := client.Ping(context.Background()).Err(); err != nil {
		panic("failed to connect redis: " + err.Error())
	}
	gzRedis := gozeredis.MustNewRedis(gozeredis.RedisConf{
		Host: c.Redis.Addr,
		Type: gozeredis.NodeType,
	})
	lock := distlock.NewDistLock(client)

	//3.连接kafka生产者
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	producer, err := sarama.NewSyncProducer(c.Kafka.Brokers, kafkaConfig)
	if err != nil {
		panic("failed to connect kafka: " + err.Error())
	}
	//4.初始化chatModel(ark)
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		BaseURL:  c.ARK.BaseURL,
		APIKey:   c.ARK.APIKey,
		Model:    c.ARK.ModelID,
		Thinking: &arkModel.Thinking{Type: arkModel.ThinkingTypeDisabled},
	})
	if err != nil {
		panic("failed to create chat model: " + err.Error())
	}
	//5.初始化agent
	milvusClient := vector.NewMilvusClient(ctx)
	emb := vector.GetEmbeddingClient(ctx)
	retrieverClient := vector.RetrieverClient(ctx, emb, milvusClient)
	t := agent.NewTools(retrieverClient)
	tools := []tool.InvokableTool{t.SearchTool()}
	reactAgent := agent.NewReActAgent(tools)
	//6.初始化历史消息
	store := session.NewStore(client)
	//7.初始化 ClickHouse 统计（非关键路径，失败不 panic）
	var analyticsClient *analytics.Analytics
	chCfg := clickhouse.Config{
		Addr:     c.ClickHouse.Addr,
		Database: c.ClickHouse.Database,
		Username: c.ClickHouse.Username,
		Password: c.ClickHouse.Password,
	}
	chDB := clickhouse.Init(chCfg)
	if chDB != nil {
		analyticsClient = analytics.New(chDB)
		if err = analyticsClient.InitSchema(); err != nil {
			logx.Errorf("clickhouse init schema failed: %v", err)
			analyticsClient = analytics.New(nil) // 建表失败则整体降级，避免每条写入都失败
		}
	} else {
		logx.Errorf("clickhouse init failed: %v, analytics disabled", clickhouse.Health())
		analyticsClient = analytics.New(nil) // nil-safe
	}
	return &ServiceContext{
		Config:        c,
		DB:            db,
		Redis:         client,
		GoZeroRedis:   gzRedis,
		KafkaProducer: producer,
		ChatModel:     chatModel,
		ReActAgent:    reactAgent,
		SessionStore:  store,
		Lock:          lock,
		DocRepo:       repository.NewDocumentRepository(db),
		Analytics:     analyticsClient,
	}
}
