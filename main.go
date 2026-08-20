package main

import (
	"context"
	"fmt"
	"os"

	"go-agent/internal/agent"
	"go-agent/internal/llm"
	"go-agent/internal/tool"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	baseURL := os.Getenv("LLM_BASE_URL")
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: LLM_API_KEY is not set")
		return
	}
	modelID := os.Getenv("LLM_MODEL")
	var model llm.Model = llm.NewClient(baseURL, apiKey, modelID)

	registry := tool.NewRegistry()
	registry.Register(&tool.TimeTool{})

	// 构造 Agent
	a := agent.New(model, registry, 10) // 最多 10 轮

	// 运行
	answer, err := a.Run(context.Background(), "现在几点了？")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Agent:", answer)
}
