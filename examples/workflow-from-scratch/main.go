package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	task := flag.String("task", "给聊天接口补一个测试", "本次练习任务")
	flag.Parse()

	ctx := context.Background()

	workflow, err := newProposalWorkflow(ctx)

	if err != nil {
		fmt.Fprintln(os.Stderr, "初始化失败：", err)
		os.Exit(1)
	}

	result, err := workflow.Invoke(ctx, proposalRequest{Task: *task})
	if err != nil {
		fmt.Fprintln(os.Stderr, "执行失败：", err)
		os.Exit(1)
	}

	fmt.Print(result.Content)
}
