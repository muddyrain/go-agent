package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()
	chatModel := &demoChatModel{}

	chain := compose.NewChain[
		[]*schema.Message,
		*schema.Message,
	]()

	chain.AppendChatModel(chatModel)

	runnable, err := chain.Compile(ctx)
	if err != nil {
		return fmt.Errorf("compile Eino chain: %w", err)
	}

	response, err := runnable.Invoke(
		ctx,
		[]*schema.Message{
			schema.UserMessage("你好"),
		},
	)
	if err != nil {
		return fmt.Errorf("invoke Eino chain: %w", err)
	}

	fmt.Printf(
		"assistant: %s\n",
		response.Content,
	)
	return nil
}
