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
	"github.com/yourname/know/cmd/consumer/internal/config"
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
	db            *gorm.DB
	lock          *distlock.DistLock
	recordRepo    *repository.ConsumeRecordRepository
	handleTimeout time.Duration
	lockTTL       time.Duration
	maxRetries    int
}

func newMessageHandler(db *gorm.DB, redisClient *redis.Client, c config.Config) *messageHandler {
	return &messageHandler{
		db:            db,
		lock:          distlock.NewDistLock(redisClient),
		recordRepo:    repository.NewConsumeRecordRepository(db),
		handleTimeout: c.Consumer.HandleTimeout,
		lockTTL:       c.Consumer.LockTTL,
		maxRetries:    c.Consumer.LockMaxRetries,
	}
}

func (h *messageHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if h.handleMessage(session.Context(), msg) {
			session.MarkMessage(msg, "")
		}
	}
	return nil
}

func (h *messageHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *messageHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *messageHandler) handleMessage(ctx context.Context, msg *sarama.ConsumerMessage) bool {
	var task embedTask
	if err := json.Unmarshal(msg.Value, &task); err != nil {
		log.Printf("invalid message: %v", err)
		return true // 格式错误直接跳过，避免死循环
	}

	topic := msg.Topic
	partition := msg.Partition
	offset := msg.Offset

	// 1. 查消费记录，done 直接跳过
	record, err := h.recordRepo.GetByKey(ctx, topic, partition, offset)
	if err != nil {
		log.Printf("[%s] check consume_record failed: %v", task.DocID, err)
		return false
	}
	if record != nil && record.Status == model.ConsumeStatusDone {
		log.Printf("[%s] already consumed (topic=%s partition=%d offset=%d)", task.DocID, topic, partition, offset)
		return true
	}
	if record != nil && record.Status == model.ConsumeStatusFailed && record.RetryCount >= h.maxRetries {
		log.Printf("[%s] give up after %d retries (topic=%s partition=%d offset=%d)", task.DocID, record.RetryCount, topic, partition, offset)
		return true
	}

	// 2. Redis 锁防重复建索引
	lockKey := fmt.Sprintf("lock:embed:%s", task.DocID)
	token := uuid.NewString()
	ok, err := h.lock.TryLock(ctx, lockKey, token, h.lockTTL)
	if err != nil {
		log.Printf("[%s] try lock failed: %v", task.DocID, err)
		return false
	}
	if !ok {
		log.Printf("[%s] lock busy, will retry", task.DocID)
		return false
	}
	defer h.lock.Unlock(ctx, lockKey, token)

	// 3. 锁内复查
	record, err = h.recordRepo.GetByKey(ctx, topic, partition, offset)
	if err != nil {
		log.Printf("[%s] double-check consume_record failed: %v", task.DocID, err)
		return false
	}
	if record != nil && record.Status == model.ConsumeStatusDone {
		log.Printf("[%s] double-check: already done", task.DocID)
		return true
	}
	if record != nil && record.Status == model.ConsumeStatusFailed && record.RetryCount >= h.maxRetries {
		log.Printf("[%s] double-check: give up after %d retries", task.DocID, record.RetryCount)
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
	inserted, err := h.recordRepo.CreateOrIgnore(ctx, newRecord)
	if err != nil {
		log.Printf("[%s] insert consume_record failed: %v", task.DocID, err)
		return false
	}
	if !inserted {
		log.Printf("[%s] consume_record already exists, skip", task.DocID)
		return true
	}

	// 5. 业务处理：查 Document → Chunk → Milvus
	ctx, cancel := context.WithTimeout(ctx, h.handleTimeout)
	defer cancel()

	var doc model.Document
	if err := h.db.WithContext(ctx).Where("doc_id = ?", task.DocID).First(&doc).Error; err != nil {
		log.Printf("[%s] doc not found: %v", task.DocID, err)
		h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusFailed, err.Error())
		return false
	}

	chunks, err := vector.Chunk(ctx, doc.DocID, doc.Content, doc.DocType)
	if err != nil {
		log.Printf("[%s] chunk failed: %v", task.DocID, err)
		h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusFailed, err.Error())
		return false
	}
	if len(chunks) == 0 {
		log.Printf("[%s] no chunks", task.DocID)
		h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusDone, "no chunks")
		return true
	}

	if _, err := docIndexer.Store(ctx, chunks); err != nil {
		log.Printf("[%s] index failed: %v", task.DocID, err)
		h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusFailed, err.Error())
		return false
	}

	// 6. 标记成功
	h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusDone, "")
	log.Printf("[%s] indexed successfully (%d chunks)", task.DocID, len(chunks))
	return true
}

func consumerInsertDocInMilvus(ctx context.Context, db *gorm.DB, redisClient *redis.Client, consumer sarama.ConsumerGroup, c config.Config) {
	handler := newMessageHandler(db, redisClient, c)

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
