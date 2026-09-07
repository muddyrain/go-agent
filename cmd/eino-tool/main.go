package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	weatherTool, err := newWeatherTool()
	if err != nil {
		return fmt.Errorf("create weather tool: %w", err)
	}

	reactAgent, err := react.NewAgent(
		ctx,
		&react.AgentConfig{
			ToolCallingModel: &demoChatModel{},
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

	messageOption, messageFuture := react.WithMessageFuture()
	userMessage := schema.UserMessage("杭州今天天气怎么样？")

	response, err := reactAgent.Generate(
		ctx,
		[]*schema.Message{userMessage},
		messageOption,
	)
	if err != nil {
		return fmt.Errorf("generate ReAct response: %w", err)
	}
	if response == nil {
		return fmt.Errorf("ReAct response is nil")
	}

	fmt.Printf("user: %s\n", userMessage.Content)

	steps := 0
	messages := messageFuture.GetMessages()

	for {
		message, ok, err := messages.Next()
		if err != nil {
			return fmt.Errorf("read ReAct message: %w", err)
		}
		if !ok {
			break
		}
		if message == nil {
			return fmt.Errorf("ReAct produced a nil message")
		}

		if message.Role == schema.Assistant {
			steps++
		}

		printMessage(message)
	}

	fmt.Printf("steps: %d\n", steps)
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
