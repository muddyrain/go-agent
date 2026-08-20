package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

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

	sm := agent.NewSessionManager()

	session := sm.GetOrCreate("user-1")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("You > ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" {
			break
		}

		answer, err := a.Run(context.Background(), session, input)

		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		fmt.Println("Agent:", answer)
	}
}
