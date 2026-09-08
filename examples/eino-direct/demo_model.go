package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type demoChatModel struct{}

var _ model.BaseChatModel = (*demoChatModel)(nil)

func (m *demoChatModel) Generate(
	ctx context.Context,
	input []*schema.Message,
	_ ...model.Option,
) (*schema.Message, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(input) == 0 {
		return nil, fmt.Errorf("model input messages are required")
	}

	lastMessage := input[len(input)-1]
	if lastMessage.Role != schema.User {
		return nil, fmt.Errorf("last message role must be user, got %s", lastMessage.Role)
	}

	return schema.AssistantMessage(
		"你好，我是 AgentHub 的 Eino 演示 Agent。",
		nil,
	), nil
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

	return schema.StreamReaderFromArray(
		[]*schema.Message{response},
	), nil
}
