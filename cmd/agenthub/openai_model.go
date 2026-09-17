package main

import (
	"agenthub/internal/config"
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
)

// newOpenAIChatModelFromEnv 从进程环境创建 OpenAI 兼容 ChatModel。
// 它只负责模型组件配置，不加载 .env；配置来源与优先级由应用入口先通过
// loadLocalEnv 处理，从而让该函数在测试和其他入口中保持职责单一。
func newOpenAIChatModelFromEnv(
	ctx context.Context,
	cfg *config.Config,
) (*openai.ChatModel, error) {

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  cfg.ModelAPIKey,
		BaseURL: cfg.ModelBaseURL,
		Model:   cfg.ModelName,
		Timeout: 60 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("create OpenAI-compatible chat model: %w", err)
	}
	return model, nil
}
