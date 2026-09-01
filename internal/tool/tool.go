package tool

import (
	"context"
	"encoding/json"
)

// Definition 是提供给模型的工具说明，仅描述名称、用途和参数协议。
type Definition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// Call 表示模型发起的一次具体工具调用；ID 用于关联后续执行结果。
type Call struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Result 是 Executor 的统一输出；业务失败通过 IsError 返回给模型，流程错误则使用 Go error。
type Result struct {
	CallID  string
	Name    string
	Content string
	IsError bool
}

// Tool 组合模型可见的定义和本地可执行能力。
type Tool interface {
	Definition() Definition
	Execute(
		ctx context.Context,
		arguments json.RawMessage,
	) (string, error)
}
