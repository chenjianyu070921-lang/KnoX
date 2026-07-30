package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/yourname/know/pkg/database"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	// 1. 连接 MySQL
	db = database.GetDB("root:root@tcp(127.0.0.1:3306)/knox?charset=utf8mb4&parseTime=True&loc=Local")

	// 1.5 初始化索引器
	InitIndexer()

	// 2. 配置 Kafka 消费者
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer, err := sarama.NewConsumerGroup([]string{"127.0.0.1:9092"}, "doc-consumer-group", config)
	if err != nil {
		panic("failed to create kafka consumer: " + err.Error())
	}
	defer consumer.Close()

	// 全局上下文：用于收到信号后通知消费协程退出
	ctx, cancel := context.WithCancel(context.Background())

	// 3. 启动消费协程
	go func() {
		consumerInsertDocInMilvus(ctx, db, consumer)
	}()

	// =========【重点】主线程阻塞 放在main中！=========
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan // 主线程卡住在这里，进程不会退出

	fmt.Println("\n收到退出信号，准备优雅关闭消费者...")
	cancel() // 通知消费循环退出

	// 等待消费完成退出（可选，给处理消息预留时间）
	time.Sleep(1 * time.Second)
	fmt.Println("服务正常退出")
}

// 简化消费函数，不再内部处理信号，只依赖外部ctx控制
func kafkaConsumerDocEmbedding(ctx context.Context, db *gorm.DB, consumer sarama.ConsumerGroup) {
	fmt.Println("消费者已启动")
	consumerInsertDocInMilvus(ctx, db, consumer)
}
