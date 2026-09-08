package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
)

func newOpenAIChatModelFromEnv(
	ctx context.Context,
) (*openai.ChatModel, error) {
	// 读取三个环境变量
	// AGENTHUB_MODEL_API_KEY
	// AGENTHUB_MODEL_BASE_URL
	// AGENTHUB_MODEL_NAME

	apiKey := strings.TrimSpace(os.Getenv("AGENTHUB_MODEL_API_KEY"))
	baseURL := strings.TrimSpace(os.Getenv("AGENTHUB_MODEL_BASE_URL"))
	modelName := strings.TrimSpace(os.Getenv("AGENTHUB_MODEL_NAME"))

	if apiKey == "" {
		return nil, fmt.Errorf("AGENTHUB_MODEL_API_KEY is required")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("AGENTHUB_MODEL_BASE_URL is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("AGENTHUB_MODEL_NAME is required")
	}

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   modelName,
		Timeout: 60 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("create OpenAI-compatible chat model: %w", err)
	}
	return model, nil
}
