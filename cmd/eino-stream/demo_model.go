package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type demoChatModel struct {
	tools []*schema.ToolInfo
}

var _ model.ToolCallingChatModel = (*demoChatModel)(nil)

func (m *demoChatModel) WithTools(
	tools []*schema.ToolInfo,
) (model.ToolCallingChatModel, error) {
	boundModel := *m
	boundModel.tools = append(
		[]*schema.ToolInfo(nil),
		tools...,
	)

	return &boundModel, nil
}

func (m *demoChatModel) Generate(
	ctx context.Context,
	input []*schema.Message,
	_ ...model.Option,
) (*schema.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(input) == 0 {
		return nil, fmt.Errorf("model input messages are required")
	}

	lastMessage := input[len(input)-1]

	switch lastMessage.Role {
	case schema.User:
		if len(m.tools) == 0 {
			return nil, fmt.Errorf("no tools bound to the model")
		}
		var hasWeatherTool bool
		for _, tool := range m.tools {
			if tool.Name == "get_weather" {
				hasWeatherTool = true
				break
			}
		}
		if !hasWeatherTool {
			return nil, fmt.Errorf("tool 'get_weather' is not bound to the model")
		}
		return schema.AssistantMessage(
			"",
			[]schema.ToolCall{
				{
					ID: "call-weather-001",
					Function: schema.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"city":"杭州"}`,
					},
				},
			},
		), nil

	case schema.Tool:
		if lastMessage.ToolName != "get_weather" {
			return nil, fmt.Errorf(
				"unexpected tool name: %s",
				lastMessage.ToolName,
			)
		}
		if lastMessage.ToolCallID != "call-weather-001" {
			return nil, fmt.Errorf(
				"unexpected tool call ID: %s",
				lastMessage.ToolCallID,
			)
		}
		return schema.AssistantMessage(
			fmt.Sprintf(
				"根据天气工具的查询结果：%s",
				lastMessage.Content,
			),
			nil,
		), nil

	default:
		return nil, fmt.Errorf(
			"unexpected last message role: %s",
			lastMessage.Role,
		)
	}
}

func (m *demoChatModel) Stream(
	ctx context.Context,
	input []*schema.Message,
	opts ...model.Option,
) (*schema.StreamReader[*schema.Message], error) {
	response, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	// 第一轮模型返回的是 ToolCall。
	// ReAct 需要从流的首块判断是否应该进入工具节点，
	// 所以这里暂时保持为一个完整块。
	if len(response.ToolCalls) > 0 {
		return schema.StreamReaderFromArray(
			[]*schema.Message{response},
		), nil
	}

	reader, writer := schema.Pipe[*schema.Message](0)

	go func() {
		defer writer.Close()

		firstChunk := schema.AssistantMessage(
			"根据天气工具的查询结果：",
			nil,
		)

		if closed := writer.Send(firstChunk, nil); closed {
			return
		}

		timer := time.NewTimer(time.Second)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			writer.Send(nil, ctx.Err())
			return
		case <-timer.C:
		}

		secondChunk := schema.AssistantMessage(
			input[len(input)-1].Content,
			nil,
		)

		writer.Send(secondChunk, nil)
	}()

	return reader, nil
}
