// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/tool"
	"github.com/redis/go-redis/v9"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/yourname/know/cmd/gateway/internal/config"
	"github.com/yourname/know/internal/agent"
	"github.com/yourname/know/internal/model"
	"github.com/yourname/know/internal/repository"
	"github.com/yourname/know/internal/session"
	"github.com/yourname/know/internal/vector"
	"github.com/yourname/know/pkg/database"
	"github.com/yourname/know/pkg/redisx"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config        config.Config
	DB            *gorm.DB
	Redis         *redis.Client
	KafkaProducer sarama.SyncProducer
	ReActAgent    *agent.ReActAgent
	ChatModel     *ark.ChatModel
	SessionStore  *session.Store
	DocRepo       *repository.DocumentRepository
}

func NewServiceContext(c config.Config) *ServiceContext {
	ctx := context.Background()
	//1.连接mysql
	db := database.GetDB(c.Mysql.DSN)
	db.AutoMigrate(&model.Document{})
	//2.连接redis
	client := redisx.GetClient(c.Redis.Addr)
	if err := client.Ping(context.Background()).Err(); err != nil {
		panic("failed to connect redis: " + err.Error())
	}
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
	return &ServiceContext{
		Config:        c,
		DB:            db,
		Redis:         client,
		KafkaProducer: producer,
		ChatModel:     chatModel,
		ReActAgent:    reactAgent,
		SessionStore:  store,
		DocRepo:       repository.NewDocumentRepository(db),
	}
}
