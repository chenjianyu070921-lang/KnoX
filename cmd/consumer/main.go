package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chenjianyu070921-lang/KnoX/cmd/consumer/internal/config"
	"github.com/chenjianyu070921-lang/KnoX/internal/model"
	"github.com/chenjianyu070921-lang/KnoX/pkg/database"
	"github.com/chenjianyu070921-lang/KnoX/pkg/redisx"

	"github.com/IBM/sarama"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

func main() {
	configFile := flag.String("f", "etc/config.yaml", "config file path")
	flag.Parse()

	// 1. 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.SetDefaults()

	// 环境变量优先，避免密钥写进配置文件进 git
	if v := os.Getenv("KNOX_MYSQL_DSN"); v != "" {
		c.Mysql.DSN = v
	}

	// 2. 连接 MySQL + AutoMigrate
	db := database.MysqlInit(c.Mysql.DSN)
	if err := db.AutoMigrate(&model.Document{}, &model.ConsumeRecord{}); err != nil {
		panic("failed to auto migrate: " + err.Error())
	}

	// 3. 连接 Redis
	redisClient := redisx.GetClient(c.Redis.Addr)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		panic("failed to connect redis: " + err.Error())
	}

	// 4. 初始化向量索引器
	InitIndexer(c)

	// 5. 配置 Kafka 消费者
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumer, err := sarama.NewConsumerGroup(c.Kafka.Brokers, c.Kafka.Group, saramaConfig)
	if err != nil {
		panic("failed to create kafka consumer: " + err.Error())
	}
	defer consumer.Close()

	// 6. 启动消费
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		consumerInsertDocInMilvus(ctx, db, redisClient, consumer, c)
	}()

	// 7. 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logx.Infof("收到退出信号，准备优雅关闭消费者...")
	cancel()
	time.Sleep(1 * time.Second)
	logx.Infof("服务正常退出")
}
