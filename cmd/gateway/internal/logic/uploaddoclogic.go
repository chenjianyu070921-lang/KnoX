// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/yourname/know/cmd/gateway/internal/svc"
	"github.com/yourname/know/cmd/gateway/internal/types"
	"github.com/yourname/know/internal/errcode"
	"github.com/yourname/know/internal/model"
	"github.com/yourname/know/internal/repository"
	"github.com/yourname/know/pkg/QiniuYun"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UploadDocLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadDocLogic {
	return &UploadDocLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadDocLogic) UploadDoc(req *types.UploadDocRequest, r *http.Request) (resp *types.UploadDocRespose, err error) {
	start := time.Now()
	var filename string
	var fileSize int64
	defer func() {
		l.svcCtx.Analytics.LogUpload(
			time.Since(start).Milliseconds(),
			err == nil,
			"",
			filename,
			fileSize,
		)
	}()

	//1.requestId幂等上传
	if req.RequestId == "" {
		return nil, errcode.New(errcode.DocUploadFailed, "请求id缺失")
	}
	token := uuid.NewString()
	redisKey := fmt.Sprintf("idem:upload:%s", req.RequestId)
	lock, err := l.svcCtx.Lock.TryLock(l.ctx, redisKey, token, time.Second*30)
	if err != nil {
		return nil, errcode.New(errcode.DocUploadFailed, "抢锁失败: "+err.Error())
	}
	if !lock {
		doc, err := l.svcCtx.DocRepo.GetByRequestID(l.ctx, req.RequestId)
		if err != nil {
			return nil, errcode.New(errcode.DocUploadFailed, "根据请求id查询文档失败: "+err.Error())
		}
		if doc == nil {
			return nil, errcode.New(errcode.DocUploadFailed, "请求处理中，请稍后重试")
		}
		// DB 有记录 → 幂等返回已有结果
		return &types.UploadDocRespose{
			DocId:   doc.DocID,
			Url:     doc.FileUrl,
			Version: doc.Version,
		}, nil
	}
	defer func() {
		if err != nil {
			_ = l.svcCtx.Lock.Unlock(l.ctx, redisKey, token)
		}
	}()
	existingDoc, dbErr := l.svcCtx.DocRepo.GetByRequestID(l.ctx, req.RequestId)
	if dbErr != nil {
		return nil, errcode.New(errcode.DocUploadFailed, "查询已有文档失败: "+dbErr.Error())
	}
	if existingDoc != nil {
		return &types.UploadDocRespose{
			DocId:   existingDoc.DocID,
			Url:     existingDoc.FileUrl,
			Version: existingDoc.Version,
		}, nil
	}
	//2.文件上传七牛云
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, errcode.New(errcode.DocUploadFailed, "读取上传文件失败: "+err.Error())
	}
	defer file.Close()

	filename = header.Filename
	fileSize = header.Size

	ext := path.Ext(header.Filename)
	docID := repository.GenDocID()
	key := docID + ext // ASCII 安全对象名，避免中文/空格导致 URL 打不开

	q := l.svcCtx.Config.Qiniu
	Url, content, err := QiniuYun.QiniuYunUpload(file, header, QiniuYun.Config{
		AccessKey: q.AccessKey,
		SecretKey: q.SecretKey,
		Bucket:    q.Bucket,
		Region:    q.Region,
		Domain:    q.Domain,
	}, key)
	if err != nil {
		return nil, errcode.New(errcode.DocUploadFailed, "七牛云上传失败: "+err.Error())
	}
	logx.Infof("qiniu upload success, key=%s, url=%s", key, Url)

	//3.详情信息存入mysql
	document := model.Document{
		DocID:     docID,
		Title:     header.Filename,
		Content:   content,
		DocType:   ext,
		FileUrl:   Url,
		RequestID: &req.RequestId,
	}
	if err = l.svcCtx.DocRepo.Create(l.ctx, &document); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, getErr := l.svcCtx.DocRepo.GetByRequestID(l.ctx, req.RequestId)
			if getErr == nil && existing != nil {
				return &types.UploadDocRespose{
					DocId:   existing.DocID,
					Url:     existing.FileUrl,
					Version: existing.Version,
				}, nil
			}
		}
		return nil, errcode.New(errcode.DocUploadFailed, err.Error())
	}
	//4.异步发送kafka
	go func() {
		task := map[string]string{
			"docId":    document.DocID,
			"filename": document.Title,
			"url":      Url,
			"docType":  document.DocType,
		}
		msgBytes, _ := json.Marshal(task)

		msg := &sarama.ProducerMessage{
			Topic: l.svcCtx.Config.Kafka.Topic,
			Value: sarama.ByteEncoder(msgBytes),
		}
		_, _, err = l.svcCtx.KafkaProducer.SendMessage(msg)
		if err != nil {
			logx.Errorf("send kafka message failed: %v", err)
		}
	}()
	return &types.UploadDocRespose{
		DocId:   document.DocID,
		Url:     Url,
		Version: document.Version,
	}, nil
}
