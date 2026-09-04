package main

import (
	"context"
	"encoding/json"
	"fmt"

	"agenthub/internal/apperr"
	"agenthub/internal/llm"
	"agenthub/internal/tool"
)

type demoModel struct{}

var _ llm.Model = (*demoModel)(nil)

func (m *demoModel) Generate(
	ctx context.Context,
	req llm.Request,
) (llm.Response, error) {
	if err := ctx.Err(); err != nil {
		return llm.Response{}, err
	}

	if len(req.Messages) == 0 {
		return llm.Response{}, fmt.Errorf("model request messages are required")
	}

	lastMessage := req.Messages[len(req.Messages)-1]

	switch lastMessage.Role {
	case llm.RoleUser:
		// 第一次模型调用。
		// 先确认 req.Tools 中存在 get_weather。
		// 然后返回 AssistantToolCalls。
		if len(req.Tools) == 0 || req.Tools[0].Name != "get_weather" {
			return llm.Response{}, fmt.Errorf(
				"expected get_weather tool in request",
			)
		}
		return llm.Response{
			Message: llm.AssistantToolCalls(
				tool.Call{
					ID:        "call-weather-001",
					Name:      "get_weather",
					Arguments: json.RawMessage(`{"city":"杭州"}`),
				},
			),
			FinishReason: "tool_calls",
			Usage:        llm.Usage{},
		}, nil

	case llm.RoleTool:
		if lastMessage.Name != "get_weather" ||
			lastMessage.ToolCallID != "call-weather-001" {
			return llm.Response{}, fmt.Errorf(
				"unexpected tool message: name=%q call_id=%q",
				lastMessage.Name,
				lastMessage.ToolCallID,
			)
		}
		return llm.Response{
			Message: llm.AssistantMessage(
				fmt.Sprintf(
					"根据天气工具的查询结果：%s",
					lastMessage.Content,
				),
			),
			FinishReason: "stop",
			Usage:        llm.Usage{},
		}, nil

	default:
		// 返回 unexpected last message role 错误。
		return llm.Response{}, apperr.New(
			apperr.CodeInternal,
			"unexpected last message role",
		)
	}
}
