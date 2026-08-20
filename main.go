package main

import (
	"context"
	"fmt"
	"os"

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

	messages := []llm.Message{
		{Role: "user", Content: "用三句话介绍一下 Go 语言"},
	}

	ch, err := model.ChatStream(context.Background(), messages, nil)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Print("Agent: ")
	for chunk := range ch {
		if chunk.Err != nil {
			fmt.Println("\nError:", chunk.Err)
			return
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println()
}
