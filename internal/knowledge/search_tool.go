package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

// searchKnowledgeArguments 定义检索工具的输入参数。
// InferTool 会根据 json/jsonschema tag 自动生成 JSON Schema，
// 大模型看到 Schema 就知道这个工具需要传什么参数。
type searchKnowledgeArguments struct {
	Query string `json:"query" jsonschema:"required,description=要检索的关键词或问题，用自然语言描述用户想查什么"`
}

// NewSearchTool 创建知识库检索工具，注册到 Agent 的工具列表。
// 这个工具让 Agent 可以自动调用 DocumentStore.Search 检索私有文档。
func NewSearchTool(store *DocumentStore) (tool.InvokableTool, error) {
	return toolutils.InferTool(
		"search_knowledge",
		"当用户询问公司内部文档、项目规范、员工手册、产品说明等私有知识库内容时使用；必须先检索再回答；工具结果包含每个片段的来源，最终回答应在相关结论后标注实际使用的来源；没有来源时说明来源未知，不要编造来源",
		func(
			ctx context.Context,
			input searchKnowledgeArguments,
		) (string, error) {
			// 检查 context 是否被取消（客户端断开连接）
			if err := ctx.Err(); err != nil {
				return "", fmt.Errorf("context canceled: %w", err)
			}

			// 调用 DocumentStore.Search 做相似度检索
			// topK=5：返回最相关的 5 个文档片段
			results, err := store.Search(ctx, input.Query, 5)
			if err != nil {
				// 检索出错属于工具执行失败，返回 error 让 Eino 处理
				return "", fmt.Errorf("search knowledge: %w", err)
			}

			if len(results) == 0 {
				return "知识库中没有找到相关文档。", nil
			}

			// 把检索结果格式化成易读的文本，返回给大模型
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("检索到 %d 条相关文档：\n\n", len(results)))

			for i, result := range results {
				source := metadataString(result.Document.Metadata, "source")
				if source == "" {
					source = "来源未知"
				}
				documentID := metadataString(
					result.Document.Metadata,
					"document_id",
				)
				sb.WriteString(fmt.Sprintf(
					"【文档片段 %d】\n",
					i+1,
				))
				sb.WriteString(fmt.Sprintf(
					"来源：%s\n",
					source,
				))
				if documentID != "" {
					sb.WriteString(fmt.Sprintf(
						"文档 ID：%s\n",
						documentID,
					))
				}
				if chunkIndex, ok := metadataInt(
					result.Document.Metadata,
					"chunk_index",
				); ok {
					sb.WriteString(fmt.Sprintf(
						"分块序号：%d\n",
						chunkIndex,
					))
				}
				sb.WriteString(fmt.Sprintf(
					"相似度：%.2f\n",
					result.Score,
				))
				sb.WriteString("内容：\n")
				sb.WriteString(result.Document.Content)
				sb.WriteString("\n\n")
			}

			return sb.String(), nil
		},
	)
}

// metadataString 从文档 Metadata 中安全读取字符串字段。
// 字段不存在或类型不匹配时返回空字符串。
func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}

	value, ok := metadata[key].(string)
	if !ok {
		return ""
	}

	return value
}

// metadataInt 从文档 Metadata 中安全读取整数字段。
// JSON 解码到 map[string]any 后，数字通常表现为 float64。
func metadataInt(metadata map[string]any, key string) (int, bool) {
	if metadata == nil {
		return 0, false
	}

	switch value := metadata[key].(type) {
	case int:
		return value, true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}
