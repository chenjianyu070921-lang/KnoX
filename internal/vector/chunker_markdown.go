package vector

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
)

var splitter document.Transformer

var (
	chunkOnce     sync.Once
	chunkSplitter document.Transformer
	chunkErr      error
)

func NewMarkdownChunker(ctx context.Context) (document.Transformer, error) {
	var err error
	// 1. 切分文档
	splitter, err = markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{
		Headers: map[string]string{
			"#":   "h1",
			"##":  "h2",
			"###": "h3",
		},
		TrimHeaders: false, // 保留标题，保证语义完整
	})
	if err != nil {
		return nil, err
	}
	return splitter, nil
}

func Chunk(ctx context.Context, docID, content, docType string) ([]*schema.Document, error) {
	chunkOnce.Do(func() {
		chunkSplitter, chunkErr = NewMarkdownChunker(ctx)
	})
	if chunkErr != nil {
		return nil, chunkErr
	}
	splitter := chunkSplitter
	// 非 md 文件回退到简单分块
	if docType != ".md" && docType != "md" {
		return nil, errors.New("格式错误")
	}

	transform, err := splitter.Transform(context.Background(), []*schema.Document{
		{
			ID:       "",
			Content:  content,
			MetaData: nil,
		},
	})
	if err != nil {
		return nil, err
	}

	// 2. 加工分块：过滤无效块 + 拼接标题路径 + 补充元数据
	var validChunks []*schema.Document
	for i, doc := range transform {
		test := strings.TrimSpace(doc.Content)
		// 过滤过短的无效块
		if len(test) < 20 {
			continue
		}

		// 从 metadata 提取各级标题，拼接完整路径
		h1, _ := doc.MetaData["h1"].(string)
		h2, _ := doc.MetaData["h2"].(string)
		h3, _ := doc.MetaData["h3"].(string)

		titlePath := h1
		if h2 != "" {
			titlePath += " > " + h2
		}
		if h3 != "" {
			titlePath += " > " + h3
		}

		// 补充元数据，标题路径前置到正文，一起向量化
		doc.ID = fmt.Sprintf("%s_chunk_%03d", docID, i)
		doc.MetaData["title_path"] = titlePath
		doc.MetaData["chunk_index"] = i
		doc.Content = titlePath + "\n" + test

		validChunks = append(validChunks, doc)
	}
	return validChunks, nil
}
