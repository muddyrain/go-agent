package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
)

// newOpenAIChatModelFromEnv 从进程环境创建 OpenAI 兼容 ChatModel。
// 它只负责模型组件配置，不加载 .env；配置来源与优先级由应用入口先通过
// loadLocalEnv 处理，从而让该函数在测试和其他入口中保持职责单一。
func newOpenAIChatModelFromEnv(
	ctx context.Context,
) (*openai.ChatModel, error) {
	// 在创建组件和发起任何网络请求前校验必填配置，避免把明显的本地
	// 配置问题延迟成难定位的远程鉴权或请求错误。
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
		// Timeout 约束单次模型 HTTP 请求，不限制整个 CLI 会话，也不替代
		// writeAssistantStream 对响应流读取错误的处理。
		Timeout: 60 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("create OpenAI-compatible chat model: %w", err)
	}
	return model, nil
}
