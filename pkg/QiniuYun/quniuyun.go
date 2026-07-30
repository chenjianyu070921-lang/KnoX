package QiniuYun

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader"
)

// Config 七牛云上传配置，由 cmd/gateway/etc/doc.yaml 的 qiniu 段注入。
type Config struct {
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string // 区域ID，如 z0=华东 z1=华北 z2=华南 na0=北美 as0=东南亚
	Domain    string // 外链默认域名（含协议），如 https://xxx.clouddn.com
}

// QiniuYunUpload 将上传文件推送到七牛云，返回可访问 URL 与文件内容（用于落库）。
// key 为对象名，建议传入 ASCII 安全字符串（如 docID+ext），避免中文/空格导致 URL 打不开。
func QiniuYunUpload(file multipart.File, header *multipart.FileHeader, cfg Config, key string) (string, string, error) {
	mac := credentials.NewCredentials(cfg.AccessKey, cfg.SecretKey)

	// 带超时的 context，避免无网络时无限挂起（原代码用 context.Background 会一直卡到 SDK 默认超时）。
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uploadManager := uploader.NewUploadManager(&uploader.UploadManagerOptions{
		Options: http_client.Options{
			Credentials: mac,
			Regions:     region.GetRegionByID(cfg.Region, true), // 显式指定区域，避免自动解析 uc.qbox.me 带来的额外外网依赖
		},
	})

	// 读取文件内容（优先用调用方传入的 file，避免重复 header.Open 导致句柄泄漏）
	var data []byte
	var err error
	if file != nil {
		data, err = io.ReadAll(file)
		if err != nil {
			return "", "", err
		}
	} else {
		open, openErr := header.Open()
		if openErr != nil {
			return "", "", openErr
		}
		defer open.Close()
		data, err = io.ReadAll(open)
		if err != nil {
			return "", "", err
		}
	}

	// 推断 Content-Type，确保浏览器能正确解析（.md -> text/markdown; charset=utf-8）
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(path.Ext(header.Filename))
	}

	err = uploadManager.UploadReader(ctx, io.NopCloser(bytes.NewReader(data)), &uploader.ObjectOptions{
		BucketName:  cfg.Bucket,
		ObjectName:  &key,
		FileName:    header.Filename,
		ContentType: contentType,
	}, nil)
	if err != nil {
		return "", "", err
	}

	domain := cfg.Domain

	// 规范化为 协议://域名/key，并对 key 做路径转义，杜绝中文/特殊字符导致的 404
	objectURL := strings.TrimRight(domain, "/") + "/" + url.PathEscape(key)

	return objectURL, string(data), nil
}
