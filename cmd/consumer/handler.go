package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chenjianyu070921-lang/KnoX/cmd/consumer/internal/config"
	"github.com/chenjianyu070921-lang/KnoX/internal/model"
	"github.com/chenjianyu070921-lang/KnoX/internal/repository"
	"github.com/chenjianyu070921-lang/KnoX/internal/vector"
	"github.com/chenjianyu070921-lang/KnoX/pkg/redisx/distlock"

	"github.com/IBM/sarama"
	"github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

var docIndexer *milvus2.Indexer

func InitIndexer(c config.Config) {
	ctx := context.Background()
	embedder, err := vector.GetEmbeddingClient(ctx, c.Ollama.URL, c.Ollama.Model)
	if err != nil {
		panic("failed to create embedding client: " + err.Error())
	}
	client, err := vector.GetMilvusClient(ctx, c.Milvus.Addr, c.Milvus.DBName)
	if err != nil {
		panic("failed to create milvus client: " + err.Error())
	}
	indexer, err := vector.GetIndexerClient(ctx, client, embedder, c.Milvus.Collection, c.Milvus.VectorField, c.Ollama.Dimension)
	if err != nil {
		panic("failed to create indexer: " + err.Error())
	}
	docIndexer = indexer
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

func consumerLogger(ctx context.Context, docID, topic string, partition int32, offset int64) logx.Logger {
	return logx.WithContext(ctx).WithFields(
		logx.Field("docId", docID),
		logx.Field("topic", topic),
		logx.Field("partition", partition),
		logx.Field("offset", offset),
	)
}

func (h *messageHandler) handleMessage(ctx context.Context, msg *sarama.ConsumerMessage) bool {
	var task embedTask
	if err := json.Unmarshal(msg.Value, &task); err != nil {
		logx.WithContext(ctx).WithFields(
			logx.Field("topic", msg.Topic),
			logx.Field("partition", msg.Partition),
			logx.Field("offset", msg.Offset),
		).Errorf("invalid message: %v", err)
		return true // 格式错误直接跳过，避免死循环
	}

	topic := msg.Topic
	partition := msg.Partition
	offset := msg.Offset

	// 1. 查消费记录，done 直接跳过
	record, err := h.recordRepo.GetByKey(ctx, topic, partition, offset)
	if err != nil {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("check consume_record failed: %v", err)
		return false
	}
	if record != nil && record.Status == model.ConsumeStatusDone {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Infof("already consumed")
		return true
	}
	if record != nil && record.Status == model.ConsumeStatusFailed && record.RetryCount >= h.maxRetries {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("give up after %d retries", record.RetryCount)
		return true
	}

	// 2. Redis 锁防重复建索引
	lockKey := fmt.Sprintf("lock:embed:%s", task.DocID)
	token := uuid.NewString()
	ok, err := h.lock.TryLock(ctx, lockKey, token, h.lockTTL)
	if err != nil {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("try lock failed: %v", err)
		return false
	}
	if !ok {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Infof("lock busy, will retry")
		return false
	}
	defer h.lock.Unlock(ctx, lockKey, token)

	// 3. 锁内复查：done / 已放弃 直接跳过
	record, err = h.recordRepo.GetByKey(ctx, topic, partition, offset)
	if err != nil {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("double-check consume_record failed: %v", err)
		return false
	}
	if record != nil && record.Status == model.ConsumeStatusDone {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Infof("double-check: already done")
		return true
	}
	if record != nil && record.Status == model.ConsumeStatusFailed && record.RetryCount >= h.maxRetries {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("double-check: give up after %d retries", record.RetryCount)
		return true
	}

	// 4. 赢家裁决：只有把行置成 processing 的消费者才继续处理
	staleBefore := time.Now().Add(-h.lockTTL)
	var claimed bool

	if record == nil {
		newRecord := &model.ConsumeRecord{
			Topic:     topic,
			Partition: partition,
			Offset:    offset,
			DocID:     task.DocID,
			Status:    model.ConsumeStatusProcessing,
		}
		inserted, err := h.recordRepo.CreateOrIgnore(ctx, newRecord)
		if err != nil {
			consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("insert consume_record failed: %v", err)
			return false
		}
		if inserted {
			claimed = true
		} else {
			// 别人抢先插入了：重新查，按已有状态决定是接管还是让路
			record, err = h.recordRepo.GetByKey(ctx, topic, partition, offset)
			if err != nil {
				consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("re-check consume_record failed: %v", err)
				return false
			}
			if record != nil && (record.Status == model.ConsumeStatusDone ||
				(record.Status == model.ConsumeStatusFailed && record.RetryCount >= h.maxRetries)) {
				return true // 已完成或已放弃，直接标记消费
			}
		}
	}

	if !claimed {
		claimed, err = h.recordRepo.TakeOver(ctx, topic, partition, offset, h.maxRetries, staleBefore)
		if err != nil {
			consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("take over consume_record failed: %v", err)
			return false
		}
	}
	if !claimed {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Infof("another consumer is handling, skip")
		return false // 不标记，等真正的赢家结果
	}

	// 5. 业务处理：查 Document → Chunk → Milvus
	ctx, cancel := context.WithTimeout(ctx, h.handleTimeout)
	defer cancel()

	var doc model.Document
	if err := h.db.WithContext(ctx).Where("doc_id = ?", task.DocID).First(&doc).Error; err != nil {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("doc not found: %v", err)
		if updErr := h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusFailed, err.Error()); updErr != nil {
			consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("mark failed failed: %v", updErr)
		}
		return false
	}

	chunks, err := vector.Chunk(ctx, doc.DocID, doc.Content, doc.DocType)
	if err != nil {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("chunk failed: %v", err)
		if updErr := h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusFailed, err.Error()); updErr != nil {
			consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("mark failed failed: %v", updErr)
		}
		return false
	}
	if len(chunks) == 0 {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Infof("no chunks")
		if updErr := h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusDone, "no chunks"); updErr != nil {
			consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("mark done failed: %v", updErr)
			return false
		}
		return true
	}

	if _, err := docIndexer.Store(ctx, chunks); err != nil {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("index failed: %v", err)
		if updErr := h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusFailed, err.Error()); updErr != nil {
			consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("mark failed failed: %v", updErr)
		}
		return false
	}

	// 6. 标记成功
	if updErr := h.recordRepo.UpdateStatus(ctx, topic, partition, offset, model.ConsumeStatusDone, ""); updErr != nil {
		consumerLogger(ctx, task.DocID, topic, partition, offset).Errorf("mark done failed: %v", updErr)
		return false
	}
	consumerLogger(ctx, task.DocID, topic, partition, offset).Infof("indexed successfully (%d chunks)", len(chunks))
	return true
}

func consumerInsertDocInMilvus(ctx context.Context, db *gorm.DB, redisClient *redis.Client, consumer sarama.ConsumerGroup, c config.Config) {
	handler := newMessageHandler(db, redisClient, c)

	for {
		select {
		case <-ctx.Done():
			logx.Infof("消费者退出")
			return
		default:
			err := consumer.Consume(ctx, []string{c.Kafka.Topic}, handler)
			if err != nil && ctx.Err() == nil {
				logx.Errorf("Kafka consume error: %v", err)
				time.Sleep(1 * time.Second)
			}
		}
	}
}
