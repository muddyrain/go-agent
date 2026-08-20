package main

import (
	"context"
	"encoding/json"
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
	tools := registry.ToLLMTools()

	messages := []llm.Message{
		{Role: "user", Content: "现在几点了？"},
	}

	fmt.Println("=== 第一次调用（请求工具）===")

	msg, err := model.Chat(context.Background(), messages, tools)

	if err != nil {
		fmt.Printf("Error:", err)
		return
	}

	fmt.Printf("模型返回: Content=%q, ToolCalls=%d\n", msg.Content, len(msg.ToolCalls))

	messages = append(messages, msg)

	for _, tc := range msg.ToolCalls {
		fmt.Printf("执行工具: %s, 参数: %s\n", tc.Function.Name, tc.Function.Arguments)

		// 从 Registry 找工具

		t, ok := registry.Get(tc.Function.Name)
		if !ok {
			fmt.Printf("工具 %s 不存在\n", tc.Function.Name)
			continue
		}
		// 执行工具，arguments 是 JSON 字符串，转成 json.RawMessage

		result, err := t.Execute(context.Background(), json.RawMessage(tc.Function.Arguments))

		if err != nil {
			fmt.Printf("工具执行失败: %v\n", err)
			continue
		}
		fmt.Printf("工具结果: %s\n", result)
		// 回填 tool 结果消息
		messages = append(messages, llm.Message{
			Role:       "tool",
			Content:    result,
			ToolCallID: tc.ID, // 必须和 tool_call 的 ID 一致
		})
		fmt.Println("\n=== 第二次调用（根据工具结果回答）===")

		finalMsg, err := model.Chat(context.Background(), messages, tools)

		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Printf("最终回答: %s\n", finalMsg.Content)
	}
}
