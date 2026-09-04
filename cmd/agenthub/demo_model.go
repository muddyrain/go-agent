package main

import (
	"context"

	"agenthub/internal/llm"
)

type demoModel struct {
	reply string
}

var _ llm.Model = (*demoModel)(nil)

func (m *demoModel) Generate(
	ctx context.Context,
	_ llm.Request,
) (llm.Response, error) {
	if err := ctx.Err(); err != nil {
		return llm.Response{}, err
	}
	return llm.Response{
		Message:      llm.AssistantMessage(m.reply),
		FinishReason: "stop",
		Usage:        llm.Usage{},
	}, nil
}
