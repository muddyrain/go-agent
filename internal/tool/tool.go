package tool

import (
	"context"
	"encoding/json"
)

type Tool interface {
	// Name 返回工具名，必须和模型返回的 tool_calls.function.name 一致
	Name() string

	// Description 返回工具描述，告诉模型这个工具干什么用
	Description() string

	// Parameters 返回 JSON Schema，告诉模型参数格式
	Parameters() json.RawMessage

	// Execute 真正执行工具，arguments 是模型传的 JSON 字符串
	Execute(ctx context.Context, arguments json.RawMessage) (string, error)
}
