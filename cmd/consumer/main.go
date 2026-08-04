package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/yourname/know/cmd/consumer/internal/config"
	"github.com/yourname/know/internal/model"
	"github.com/yourname/know/pkg/database"
	"github.com/yourname/know/pkg/redisx"
	"github.com/zeromicro/go-zero/core/conf"
)

func main() {
	configFile := flag.String("f", "etc/config.yaml", "config file path")
	flag.Parse()

	// 1. 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 环境变量优先，避免密钥写进配置文件进 git
	if v := os.Getenv("KNOX_MYSQL_DSN"); v != "" {
		c.Mysql.DSN = v
	}

	// 2. 连接 MySQL + AutoMigrate
	db := database.GetDB(c.Mysql.DSN)
	db.AutoMigrate(&model.Document{}, &model.ConsumeRecord{})

	// 3. 连接 Redis
	redisClient := redisx.GetClient(c.Redis.Addr)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		panic("failed to connect redis: " + err.Error())
	}

	// 4. 初始化向量索引器
	InitIndexer()

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

	fmt.Println("\n收到退出信号，准备优雅关闭消费者...")
	cancel()
	time.Sleep(1 * time.Second)
	fmt.Println("服务正常退出")
}
