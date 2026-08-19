package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"go-agent/internal/llm"

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

	messages := []llm.Message{
		{
			Role:    "user",
			Content: "你好，请用一句话介绍你自己",
		},
	}

	for {
		stdin := bufio.NewReader(os.Stdin)
		fmt.Print("User: ")
		userInput, _ := stdin.ReadString('\n')
		userInput = userInput[:len(userInput)-1] // 去掉换行符

		if userInput == "exit" {
			break
		}

		messages = append(messages, llm.Message{
			Role:    "user",
			Content: userInput,
		})

		reply, err := model.Chat(context.Background(), messages)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Agent:", reply)

		messages = append(messages, llm.Message{
			Role:    "assistant",
			Content: reply,
		})
	}

}
