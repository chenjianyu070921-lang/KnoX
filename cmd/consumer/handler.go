package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/yourname/know/internal/model"
	"github.com/yourname/know/internal/vector"
	"gorm.io/gorm"
)

var docIndexer *milvus2.Indexer

func InitIndexer() {
	ctx := context.Background()
	emb := vector.EmbeddingClient(ctx)
	milvusClient := vector.NewMilvusClient(ctx)
	docIndexer = vector.GetIndexerClient(ctx, milvusClient, emb)
	log.Println("indexer initialized")
}

type embedTask struct {
	DocID   string `json:"docId"`
	DocType string `json:"docType"`
}

// consumerInsertDocInMilvus 启动 Kafka 消费循环
func consumerInsertDocInMilvus(ctx context.Context, db *gorm.DB, consumer sarama.ConsumerGroup) {
	handler := &messageHandler{db: db}
	for {
		err := consumer.Consume(ctx, []string{"doc-embedding"}, handler)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Printf("consume error: %v", err)
		}
	}
	log.Println("consumer stopped")
}

type messageHandler struct {
	db *gorm.DB
}

func (h *messageHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *messageHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *messageHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.handleMessage(msg.Value)
		session.MarkMessage(msg, "")
	}
	return nil
}

func (h *messageHandler) handleMessage(data []byte) {
	var task embedTask
	if err := json.Unmarshal(data, &task); err != nil {
		log.Printf("invalid message: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. 从 MySQL 取文档内容
	var doc model.Document
	if err := h.db.WithContext(ctx).Where("doc_id = ?", task.DocID).First(&doc).Error; err != nil {
		log.Printf("doc %s not found: %v", task.DocID, err)
		return
	}

	// 2. 分块
	chunks, err := vector.Chunk(ctx, doc.DocID, doc.Content, doc.DocType)
	if err != nil {
		log.Printf("chunk failed: %v", err)
		return
	}
	if len(chunks) == 0 {
		log.Printf("no chunks for doc %s", task.DocID)
		return
	}

	// 3. 索引到 Milvus（Indexer 内部自动做 embedding + 存储）
	if _, err := docIndexer.Store(ctx, chunks); err != nil {
		log.Printf("index doc %s failed: %v", task.DocID, err)
		return
	}

	log.Printf("doc %s indexed successfully (%d chunks)", task.DocID, len(chunks))
}
