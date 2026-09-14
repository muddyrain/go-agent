package main

import (
	"agenthub/internal/cli"
	"agenthub/internal/httpapi"
	"agenthub/internal/mcpclient"
	"agenthub/internal/mcpserver"
	"agenthub/internal/projecttool"
	"agenthub/internal/toolcatalog"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

const (
	httpServerMode = "serve"
	mcpServerMode  = "mcp-server"
)

func main() {
	// 同一个可执行文件支持两种进程角色：
	// 1. 默认模式运行交互式 CLI；
	// 2. mcp-server 模式作为独立子进程，通过 stdio 提供 MCP 工具。
	// 父进程在 run 中使用 os.Executable 再次启动当前程序，避免为教学 Server
	// 额外维护第二个二进制入口。
	if len(os.Args) > 1 && os.Args[1] == mcpServerMode {
		if len(os.Args) < 3 {
			log.Fatal("MCP server ID is required")
		}

		if err := mcpserver.ServeStdio(os.Args[2]); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// run 是应用组装入口：创建模型、工具与 Agent，再根据启动参数选择
	// HTTP 或 CLI 入口。Eino ReAct Agent 只负责 Model → Tool → Model 的
	// 执行闭环，不承担 HTTP 协议、终端交互或会话存储职责。
	if err := loadLocalEnv(".env"); err != nil {
		return err
	}
	ctx := context.Background()

	chatModel, err := newOpenAIChatModelFromEnv(ctx)
	if err != nil {
		return err
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get project root: %w", err)
	}

	projectFileTool, err := projecttool.New(projectRoot)
	if err != nil {
		return fmt.Errorf("create project file tool: %w", err)
	}

	// 教学用 MCP Server 与 CLI 共用当前二进制：Manager 会分别启动
	// calculator 和 accounting 子进程，并为每条连接建立一个 Session。
	// Server 配置属于应用组装决策，所以放在入口；连接、发现和关闭逻辑
	// 则封装在 internal/mcpclient 中。
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get AgentHub executable: %w", err)
	}

	mcpManager, err := mcpclient.OpenServers(
		ctx,
		[]mcpclient.ServerConfig{
			{
				Name:    mcpserver.ServerCalculator,
				Command: executable,
				Args: []string{
					mcpServerMode,
					mcpserver.ServerCalculator,
				},
			},
			{
				Name:    mcpserver.ServerAccounting,
				Command: executable,
				Args: []string{
					mcpServerMode,
					mcpserver.ServerAccounting,
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("open MCP servers: %w", err)
	}

	defer func() {
		// run 无论从正常退出还是错误路径返回，都要统一释放全部 MCP
		// Session。Manager 负责倒序关闭，入口只负责记录清理错误。
		if err := mcpManager.Close(); err != nil {
			log.Printf("close MCP servers: %v", err)
		}
	}()

	// Catalog 是模型工具集合与 CLI 诊断视图的单一来源：本地工具和
	// MCP 工具在这里统一登记来源、所属 Server 和启用状态；真正执行
	// ToolCall 的仍是 Eino ToolsNode，不是 Catalog。
	entries := []toolcatalog.Entry{
		{
			Tool:    projectFileTool,
			Source:  toolcatalog.SourceLocal,
			Enabled: true,
		},
	}

	for _, discovered := range mcpManager.Tools() {
		entries = append(
			entries,
			toolcatalog.Entry{
				Tool:    discovered.Tool,
				Source:  toolcatalog.SourceMCP,
				Server:  discovered.Server,
				Enabled: true,
			},
		)
	}

	toolCatalog, err := toolcatalog.New(entries...)
	if err != nil {
		return fmt.Errorf("create tool catalog: %w", err)
	}

	reactAgent, err := react.NewAgent(
		ctx,
		&react.AgentConfig{
			ToolCallingModel: chatModel,
			ToolsConfig: compose.ToolsNodeConfig{
				// 只有启用的工具进入模型 Schema 和 ToolsNode 路由；Catalog
				// 中的禁用条目仍可由 /tools 展示，但模型无法调用。
				Tools: toolCatalog.EnabledTools(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("create ReAct agent: %w", err)
	}

	const systemPrompt = "你是 AgentHub 项目助手。" +
		"当用户要求读取、查看、分析或总结项目文件时，" +
		"必须直接调用 read_project_file 工具，" +
		"不要在工具调用前输出计划或说明文字。" +
		"如果回答不依赖项目文件，则直接回答，不要调用工具。"

	sessionManager := httpapi.NewSessionManager(20) // 每个会话保留最近 20 条消息

	if len(os.Args) > 1 && os.Args[1] == httpServerMode {
		return httpapi.Run(
			reactAgent,
			systemPrompt,
			sessionManager,
		)
	}

	return cli.Run(
		ctx,
		reactAgent,
		toolCatalog,
		systemPrompt,
		os.Stdin,
		os.Stdout,
	)

}
