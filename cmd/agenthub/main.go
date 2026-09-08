package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

func main() {
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

	weatherTool, err := newWeatherTool()
	if err != nil {
		return fmt.Errorf("create weather tool: %w", err)
	}

	reactAgent, err := react.NewAgent(
		ctx,
		&react.AgentConfig{
			ToolCallingModel: chatModel,
			ToolsConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					weatherTool,
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("create ReAct agent: %w", err)
	}

	userMessage := schema.UserMessage("杭州今天天气怎么样？")

	stream, err := reactAgent.Stream(
		ctx,
		[]*schema.Message{userMessage},
		agent.WithComposeOptions(
			compose.WithCallbacks(newLifecycleCallback()),
		),
	)

	if err != nil {
		return fmt.Errorf("stream ReAct response: %w", err)
	}
	defer stream.Close()

	fmt.Printf("user: %s\n", userMessage.Content)
	fmt.Print("assistant: ")
	var answer strings.Builder
	chunks := 0

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("receive ReAct stream: %w", err)
		}
		if chunk == nil {
			return fmt.Errorf("ReAct produced a nil stream chunk")
		}

		chunks++
		answer.WriteString(chunk.Content)
		fmt.Print(chunk.Content)
	}

	fmt.Println()
	fmt.Printf("chunks: %d\n", chunks)
	fmt.Printf("full_answer: %s\n", answer.String())
	return nil
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
