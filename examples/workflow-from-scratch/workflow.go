package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
)

type proposalRequest struct {
	Task string
}

type taskAnalysis struct {
	Task  string
	Steps []string
}

type proposalResult struct {
	Content string
}

func analyzeTask(
	ctx context.Context,
	input proposalRequest,
) (taskAnalysis, error) {
	if err := ctx.Err(); err != nil {
		return taskAnalysis{}, fmt.Errorf("analyze task: %w", err)
	}
	task := strings.TrimSpace(input.Task)
	if task == "" {
		return taskAnalysis{}, errors.New("task is required")
	}

	return taskAnalysis{
		Task: task,
		Steps: []string{
			"明确输入与输出",
			"实现最小逻辑",
			"验证成功与错误路径",
		},
	}, nil
}

func buildProposal(
	ctx context.Context,
	analysis taskAnalysis,
) (proposalResult, error) {
	if err := ctx.Err(); err != nil {
		return proposalResult{}, fmt.Errorf("build proposal: %w", err)
	}
	// 固定规则模板，仅用于观察节点数据传递，不是模型生成的技术方案。
	var content strings.Builder
	fmt.Fprintf(
		&content,
		"任务：%s\n方案类型：规则模板（非模型生成）\n",
		analysis.Task,
	)

	for i, step := range analysis.Steps {
		fmt.Fprintf(&content, "%d. %s\n", i+1, step)
	}

	return proposalResult{
		Content: content.String(),
	}, nil
}

func newProposalWorkflow(ctx context.Context) (compose.Runnable[proposalRequest, proposalResult], error) {
	workflow := compose.NewWorkflow[proposalRequest, proposalResult]()

	workflow.AddLambdaNode(
		"analyze_task",
		compose.InvokableLambda(analyzeTask),
	).AddInput(compose.START)

	workflow.AddLambdaNode(
		"build_proposal",
		compose.InvokableLambda(buildProposal),
	).AddInput("analyze_task")

	workflow.End().AddInput("build_proposal")

	// Compile检查并构建接线，不执行具体任务；Invoke才运行节点。
	runnable, err := workflow.Compile(ctx)

	if err != nil {
		return nil, fmt.Errorf("compile proposal workflow: %w", err)
	}
	return runnable, nil
}
