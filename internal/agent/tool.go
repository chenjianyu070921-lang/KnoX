package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

type Tool struct {
	retriver *milvus2.Retriever
}

func NewTools(r *milvus2.Retriever) *Tool {
	return &Tool{retriver: r}
}
func (t *Tool) SearchTool() tool.InvokableTool {
	SearchTool := utils.NewTool(&schema.ToolInfo{
		Name:  "knowledge_search",
		Desc:  "搜索知识库中的文档，找到与用户问题相关的内容片段。当用户问到文档里可能有的信息时使用此工具。",
		Extra: nil,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "用户的问题，用于知识库向量检索",
				Required: true,
			},
		}),
	}, t.SearchHandler)
	return SearchTool
}
func (t *Tool) SearchHandler(ctx context.Context, query string) (string, error) {
	docs, err := t.retriver.Retrieve(ctx, query)
	if err != nil {
		return "", err
	}

	result := "找到以下相关内容：\n\n"
	for i, doc := range docs {
		result += fmt.Sprintf("[%d] %s\n\n", i+1, doc.Content)
	}

	return result, nil
}
