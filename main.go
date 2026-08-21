package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"go-agent/internal/agent"
	"go-agent/internal/llm"
	"go-agent/internal/mcp"
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
	model, err := llm.NewEinoModelAdapter(context.Background(), baseURL, apiKey, modelID)

	if err != nil {
		fmt.Println("Error: create model:", err)
		return
	}

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

	// 启动 MCP Server 子进程
	mcpClient := &mcp.Client{}
	if err := mcpClient.Start(context.Background(), "./mcp-server"); err != nil {
		fmt.Println("Error: start mcp server:", err)
		return
	}
	defer mcpClient.Close() // 程序退出时关闭子进程

	// MCP 初始化握手
	if err := mcpClient.Initialize(); err != nil {
		fmt.Println("Error: mcp initialize:", err)
		return
	}

	// 列出 MCP Server 提供的工具
	mcpTools, err := mcpClient.ListTools()
	if err != nil {
		fmt.Println("Error: list mcp tools:", err)
		return
	}

	// 把每个 MCP 工具包装后注册到 Registry
	for _, info := range mcpTools {
		registry.Register(tool.NewMCPTool(mcpClient, info))
		fmt.Printf("已注册 MCP 工具: %s\n", info.Name)
	}

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
