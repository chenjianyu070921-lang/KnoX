package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/yourname/know/internal/model"
	"github.com/yourname/know/internal/repository"
	"github.com/yourname/know/internal/vector"
	"github.com/yourname/know/pkg/redisx/distlock"
	"gorm.io/gorm"
)

var docIndexer *milvus2.Indexer

func InitIndexer() {
	ctx := context.Background()
	embedder := vector.GetEmbeddingClient(ctx)
	client := vector.NewMilvusClient(ctx)
	docIndexer = vector.GetIndexerClient(ctx, client, embedder)
}

type embedTask struct {
	DocID   string `json:"docId"`
	DocType string `json:"docType"`
}

type messageHandler struct {
	db         *gorm.DB
	lock       *distlock.DistLock
	recordRepo *repository.ConsumeRecordRepository
	docRepo    *repository.DocumentRepository
}

func newMessageHandler(db *gorm.DB, redisClient *redis.Client) *messageHandler {
	return &messageHandler{
		db:         db,
		lock:       distlock.NewDistLock(redisClient),
		recordRepo: repository.NewConsumeRecordRepository(db),
		docRepo:    repository.NewDocumentRepository(db),
	}
}

func (h *messageHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if h.handleMessage(msg) {
			session.MarkMessage(msg, "")
		}
	}
	return nil
}

func (h *messageHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *messageHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *messageHandler) handleMessage(msg *sarama.ConsumerMessage) bool {
	var task embedTask
	if err := json.Unmarshal(msg.Value, &task); err != nil {
		log.Printf("invalid message: %v", err)
		return true // 格式错误直接跳过，避免死循环
	}

	topic := msg.Topic
	partition := msg.Partition
	offset := msg.Offset

	// 1. 查消费记录，done 直接跳过
	record, err := h.recordRepo.GetByKey(context.Background(), topic, partition, offset)
	if err != nil {
		log.Printf("[%s] check consume_record failed: %v", task.DocID, err)
		return false
	}
	if record != nil && record.Status == model.ConsumeStatusDone {
		log.Printf("[%s] already consumed (topic=%s partition=%d offset=%d)", task.DocID, topic, partition, offset)
		return true
	}

	// 2. Redis 锁防重复建索引
	lockKey := fmt.Sprintf("lock:embed:%s", task.DocID)
	token := uuid.NewString()
	ok, err := h.lock.TryLock(context.Background(), lockKey, token, 30*time.Second)
	if err != nil {
		log.Printf("[%s] try lock failed: %v", task.DocID, err)
		return false
	}
	if !ok {
		log.Printf("[%s] lock busy, will retry", task.DocID)
		return false
	}
	defer h.lock.Unlock(context.Background(), lockKey, token)

	// 3. 锁内复查
	record, err = h.recordRepo.GetByKey(context.Background(), topic, partition, offset)
	if err != nil {
		log.Printf("[%s] double-check consume_record failed: %v", task.DocID, err)
		return false
	}
	if record != nil && record.Status == model.ConsumeStatusDone {
		log.Printf("[%s] double-check: already done", task.DocID)
		return true
	}

	// 4. 插入 processing 记录（OnConflict DoNothing）
	newRecord := &model.ConsumeRecord{
		Topic:     topic,
		Partition: partition,
		Offset:    offset,
		DocID:     task.DocID,
		Status:    model.ConsumeStatusProcessing,
	}
	if err := h.recordRepo.CreateOrIgnore(context.Background(), newRecord); err != nil {
		log.Printf("[%s] insert consume_record failed: %v", task.DocID, err)
		return false
	}

	// 5. 业务处理：查 Document → Chunk → Milvus
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var doc model.Document
	if err := h.db.WithContext(ctx).Where("doc_id = ?", task.DocID).First(&doc).Error; err != nil {
		log.Printf("[%s] doc not found: %v", task.DocID, err)
		h.recordRepo.UpdateStatus(context.Background(), topic, partition, offset, model.ConsumeStatusFailed, err.Error())
		return false
	}

	chunks, err := vector.Chunk(ctx, doc.DocID, doc.Content, doc.DocType)
	if err != nil {
		log.Printf("[%s] chunk failed: %v", task.DocID, err)
		h.recordRepo.UpdateStatus(context.Background(), topic, partition, offset, model.ConsumeStatusFailed, err.Error())
		return false
	}
	if len(chunks) == 0 {
		log.Printf("[%s] no chunks", task.DocID)
		h.recordRepo.UpdateStatus(context.Background(), topic, partition, offset, model.ConsumeStatusFailed, "no chunks")
		return false
	}

	if _, err := docIndexer.Store(ctx, chunks); err != nil {
		log.Printf("[%s] index failed: %v", task.DocID, err)
		h.recordRepo.UpdateStatus(context.Background(), topic, partition, offset, model.ConsumeStatusFailed, err.Error())
		return false
	}

	// 6. 标记成功
	h.recordRepo.UpdateStatus(context.Background(), topic, partition, offset, model.ConsumeStatusDone, "")
	log.Printf("[%s] indexed successfully (%d chunks)", task.DocID, len(chunks))
	return true
}

func consumerInsertDocInMilvus(ctx context.Context, db *gorm.DB, redisClient *redis.Client, consumer sarama.ConsumerGroup) {
	handler := newMessageHandler(db, redisClient)

	for {
		select {
		case <-ctx.Done():
			log.Println("消费者退出")
			return
		default:
			err := consumer.Consume(ctx, []string{"doc-embedding"}, handler)
			if err != nil && ctx.Err() == nil {
				log.Printf("Kafka consume error: %v", err)
				time.Sleep(1 * time.Second)
			}
		}
	}
}
