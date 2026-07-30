// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"
	"encoding/json"
	"net/http"
	"path"

	"github.com/IBM/sarama"
	"github.com/yourname/know/cmd/gateway/internal/svc"
	"github.com/yourname/know/cmd/gateway/internal/types"
	"github.com/yourname/know/internal/errcode"
	"github.com/yourname/know/internal/model"
	"github.com/yourname/know/internal/repository"
	"github.com/yourname/know/pkg/QiniuYun"
	"github.com/zeromicro/go-zero/core/logx"
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
	//1.文件上传七牛云
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, errcode.New(errcode.DocUploadFailed, "读取上传文件失败: "+err.Error())
	}
	defer file.Close()

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

	//2.详情信息存入mysql
	document := model.Document{
		DocID:   docID,
		Title:   header.Filename,
		Content: content,
		DocType: ext,
	}
	if err = l.svcCtx.DocRepo.Create(l.ctx, &document); err != nil {
		return nil, errcode.New(errcode.DocUploadFailed, err.Error())
	}
	//3.异步发送kafka
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
