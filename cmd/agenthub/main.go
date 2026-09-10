package main

import (
	"agenthub/internal/mcpclient"
	"agenthub/internal/mcpserver"
	"agenthub/internal/session"
	"agenthub/internal/toolcatalog"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

const (
	maxContextTurns = 3
	mcpServerMode   = "mcp-server"
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
	// run 是当前 CLI 应用的组装入口：创建模型、工具与 Agent，
	// 然后维护终端输入输出和单用户内存会话。Eino ReAct Agent 只负责
	// Model → Tool → Model 的执行闭环，不应该承担终端交互或会话存储职责。
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

	projectFileTool, err := newProjectFileTool(projectRoot)
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

	// history 保存终端会话的完整消息事实；ContextWindow 每轮只基于它
	// 生成临时模型视图。不能用裁剪后的视图覆盖 history，否则被暂时丢弃
	// 的旧消息将永久丢失，未来也无法更换上下文策略或持久化完整会话。
	history := []*schema.Message{
		schema.SystemMessage(
			"你是 AgentHub 项目助手。" +
				"当用户要求读取、查看、分析或总结项目文件时，" +
				"必须直接调用 read_project_file 工具，" +
				"不要在工具调用前输出计划或说明文字。" +
				"如果回答不依赖项目文件，则直接回答，不要调用工具。",
		),
	}
	contextWindow, err := session.NewContextWindow(maxContextTurns)
	if err != nil {
		return fmt.Errorf("create context window: %w", err)
	}
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\n> ")

		input, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			fmt.Println("\nbye")
			return nil
		}
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.EqualFold(input, "/tools") {
			// /tools 是本地 CLI 控制命令，不属于用户与模型的对话。
			// 必须在创建 UserMessage 前截获，避免污染 history 或触发模型。
			if err := printToolCatalog(ctx, os.Stdout, toolCatalog); err != nil {
				fmt.Printf("error: %v\n", err)
			}
			continue
		}

		if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
			fmt.Println("bye")
			return nil
		}
		userMessage := schema.UserMessage(input)
		history = append(history, userMessage)

		// BuildModelView 只保留系统消息和最近若干个完整用户轮次。
		// 完整轮次以 UserMessage 为边界，避免裁剪后从 Assistant 或 Tool
		// 消息开始，导致模型看到缺少提问或 ToolCall 的残缺上下文。
		contextView := contextWindow.BuildModelView(history)

		fmt.Printf(
			"context: history=%d model=%d dropped=%d turns=%d reason=%s\n",
			contextView.TotalMessages,
			contextView.KeptMessages,
			contextView.DroppedMessages,
			contextView.KeptTurns,
			contextView.Reason,
		)

		// Stream 提供给终端的是最终 Assistant 文本流；MessageFuture 额外
		// 保留 ReAct 内部产生的完整 Assistant ToolCall、ToolResult 和最终
		// Assistant 消息，供本轮结束后写回会话历史。
		msgOpt, future := react.WithMessageFuture()

		stream, err := reactAgent.Stream(
			ctx,
			contextView.Messages,
			msgOpt,
			agent.WithComposeOptions(
				compose.WithCallbacks(newLifecycleCallback()),
			),
		)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			// 本轮还没有形成完整 Assistant 回复，撤回刚加入的 UserMessage，
			// 避免下轮 history 出现“只有提问、没有回答”的失败半轮。
			history = history[:len(history)-1]
			continue
		}

		fmt.Printf("user: %s\n", input)
		fmt.Print("assistant: ")

		chunks, err := writeAssistantStream(os.Stdout, stream)
		if err != nil {
			fmt.Printf("\nerror: %v\n", err)
			// 流中途失败时，用户可能已经看到部分文字，但它不是一条完整、
			// 可复用的 Assistant 消息，因此本轮 UserMessage 也不写入历史。
			history = history[:len(history)-1]
			continue
		}

		fmt.Println()
		fmt.Printf("chunks: %d\n", chunks)

		// 终端流只负责让用户尽快看到最终文本；完整历史必须从
		// MessageFuture 收集。一次工具闭环通常包含：Assistant ToolCall →
		// ToolResult → 最终 Assistant。每个消息自身也是增量流，需先用
		// ConcatMessages 合并，再按产生顺序写回 history。
		iter := future.GetMessageStreams()
		for {
			msgStream, hasNext, err := iter.Next()
			if err != nil {
				return fmt.Errorf("collect agent messages: %w", err)
			}
			if !hasNext {
				break
			}

			var roundMsgs []*schema.Message
			for {
				msg, err := msgStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return fmt.Errorf("read agent message stream: %w", err)
				}
				roundMsgs = append(roundMsgs, msg)
			}
			msgStream.Close()

			if len(roundMsgs) == 0 {
				continue
			}
			concated, err := schema.ConcatMessages(roundMsgs)
			if err != nil {
				return fmt.Errorf("concat agent message: %w", err)
			}
			history = append(history, concated)
		}
	}

}

// printMessage 是早期调试完整消息链的辅助函数；当前 CLI 的用户可见文本
// 由 writeAssistantStream 输出，工具执行过程主要通过 Callback 观察。
func printMessage(message *schema.Message) {
	switch message.Role {
	case schema.Assistant:
		if len(message.ToolCalls) > 0 {
			for _, toolCall := range message.ToolCalls {
				fmt.Printf(
					"tool_call: id=%s name=%s arguments=%s\n",
					toolCall.ID,
					toolCall.Function.Name,
					toolCall.Function.Arguments,
				)
			}
			return
		}

		fmt.Printf("assistant: %s\n", message.Content)

	case schema.Tool:
		fmt.Printf(
			"tool_result: id=%s name=%s content=%s\n",
			message.ToolCallID,
			message.ToolName,
			message.Content,
		)
	}
}

// newLifecycleCallback 只观察 ChatModel 和 Tool 两类关键组件，帮助学习
// ReAct 的 Model → Tool → Model 顺序；Callback 不参与业务控制和消息保存。
func newLifecycleCallback() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			_ callbacks.CallbackInput,
		) context.Context {
			if shouldObserve(info) {
				fmt.Printf(
					"callback: start component=%s name=%s\n",
					info.Component,
					info.Name,
				)
			}
			return ctx
		}).
		OnEndFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			_ callbacks.CallbackOutput,
		) context.Context {
			if shouldObserve(info) {
				fmt.Printf(
					"callback: end component=%s name=%s\n",
					info.Component,
					info.Name,
				)
			}
			return ctx
		}).
		OnEndWithStreamOutputFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			output *schema.StreamReader[callbacks.CallbackOutput],
		) context.Context {
			defer output.Close()

			if shouldObserve(info) {
				fmt.Printf(
					"callback: stream_ready component=%s name=%s\n",
					info.Component,
					info.Name,
				)
			}
			return ctx
		}).
		OnErrorFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			err error,
		) context.Context {
			if shouldObserve(info) {
				fmt.Printf(
					"callback: error component=%s name=%s error=%v\n",
					info.Component,
					info.Name,
					err,
				)
			}
			return ctx
		}).
		Build()
}

func shouldObserve(info *callbacks.RunInfo) bool {
	if info == nil {
		return false
	}

	return info.Component == components.ComponentOfChatModel ||
		info.Component == components.ComponentOfTool
}
