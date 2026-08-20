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

	// 获取当前项目main.go所在的目录作为工作目录
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return
	}

	registry := tool.NewRegistry()
	registry.Register(&tool.TimeTool{})
	registry.Register(&tool.ListDirTool{WorkDir: workDir})
	registry.Register(&tool.ReadFileTool{WorkDir: workDir})
	registry.Register(&tool.SearchCodeTool{WorkDir: workDir})

	a := agent.New(model, registry, 10)

	sm := agent.NewSessionManager()
	session := sm.GetOrCreate("user-1")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("You > ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input = strings.TrimSpace(input)
		if input == "exit" || input == "quit" {
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
