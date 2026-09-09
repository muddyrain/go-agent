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
	if len(os.Args) > 1 && os.Args[1] == mcpServerMode {
		if err := mcpserver.ServeStdio(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
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

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get AgentHub executable: %w", err)
	}
	mcpSession, err := mcpclient.OpenStdio(
		ctx,
		executable,
		mcpServerMode,
	)
	if err != nil {
		return fmt.Errorf("open MCP session: %w", err)
	}
	defer func() {
		if err := mcpSession.Close(); err != nil {
			log.Printf("close MCP session: %v", err)
		}
	}()
	entries := []toolcatalog.Entry{
		{
			Tool:    projectFileTool,
			Source:  toolcatalog.SourceLocal,
			Enabled: true,
		},
	}
	for _, mcpTool := range mcpSession.Tools() {
		entries = append(entries, toolcatalog.Entry{
			Tool:    mcpTool,
			Source:  toolcatalog.SourceMCP,
			Enabled: true,
		})
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
				Tools: toolCatalog.EnabledTools(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("create ReAct agent: %w", err)
	}

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

		contextView := contextWindow.BuildModelView(history)

		fmt.Printf(
			"context: history=%d model=%d dropped=%d turns=%d reason=%s\n",
			contextView.TotalMessages,
			contextView.KeptMessages,
			contextView.DroppedMessages,
			contextView.KeptTurns,
			contextView.Reason,
		)

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
			history = history[:len(history)-1]
			continue
		}

		fmt.Printf("user: %s\n", input)
		fmt.Print("assistant: ")

		chunks, err := writeAssistantStream(os.Stdout, stream)
		if err != nil {
			fmt.Printf("\nerror: %v\n", err)
			history = history[:len(history)-1]
			continue
		}

		fmt.Println()
		fmt.Printf("chunks: %d\n", chunks)

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
