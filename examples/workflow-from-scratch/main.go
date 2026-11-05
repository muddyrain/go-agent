package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	task := flag.String("task", "给聊天接口补一个测试", "本次练习任务")

	serve := flag.Bool(
		"serve",
		false,
		"启动 HTTP 服务，监听 127.0.0.1:8092",
	)

	flag.Parse()

	ctx := context.Background()
	// 两种入口共用一次构建；HTTP请求只Invoke，不在请求里重新Compile。
	workflow, err := newProposalWorkflow(ctx)

	if err != nil {
		fmt.Fprintln(os.Stderr, "初始化失败：", err)
		os.Exit(1)
	}

	if *serve {
		h := newProposalServer(workflow)
		h.Spin()
		return
	}

	result, err := workflow.Invoke(ctx, proposalRequest{Task: *task})
	if err != nil {
		fmt.Fprintln(os.Stderr, "执行失败：", err)
		os.Exit(1)
	}

	fmt.Print(result.Content)
}
