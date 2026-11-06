package main

import (
	"context"
	"errors"
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

	cancelDemo := flag.Bool(
		"cancel-demo",
		false,
		"演示：在 CLI 执行前取消本次任务",
	)

	flag.Parse()

	if *serve && *cancelDemo {
		fmt.Fprintln(os.Stderr, "参数错误：-cancel-demo 仅用于 CLI，不能与 -serve 同时使用")
		os.Exit(1)
	}

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

	// 构建ctx与本次执行ctx分开；演示取消不影响服务启动或其他请求。
	runCtx, cancel := context.WithCancel(ctx)
	if *cancelDemo {
		cancel()
	}

	result, err := workflow.Invoke(runCtx, proposalRequest{Task: *task})
	// 正常或错误返回都释放本次ctx；os.Exit不会执行defer，因此在分支前调用。
	cancel()
	if err != nil {
		var inputErr *proposalInputError
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "任务已取消")
		} else if errors.As(err, &inputErr) {
			// 只读取已确认的输入错误字段；不把所有内部故障都当作输入错误。
			fmt.Fprintf(
				os.Stderr,
				"输入错误：字段 %q %s\n",
				inputErr.Field,
				inputErr.Reason,
			)
		} else {
			fmt.Fprintln(os.Stderr, "执行失败：", err)
		}
		os.Exit(1)
	}
	fmt.Print(result.Content)
}
