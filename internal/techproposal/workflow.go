package techproposal

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
)

const nodeAnalyzeTask = "analyze_task"

// ProposalRequest 是技术方案 Workflow 的入口数据。
type ProposalRequest struct {
	Task string
}

// TaskAnalysis 是分析节点产生的结构化任务要求。
type TaskAnalysis struct {
	OriginalTask         string
	Deliverable          string
	RequiredSections     []string
	EvidenceRequirements []string
}

// analyzeTask 把原始请求转换为后续节点可检查的验收清单。
func analyzeTask(
	ctx context.Context,
	input ProposalRequest,
) (TaskAnalysis, error) {
	if err := ctx.Err(); err != nil {
		return TaskAnalysis{}, fmt.Errorf(
			"analyze proposal task: %w",
			err,
		)
	}

	task := strings.TrimSpace(input.Task)
	if task == "" {
		return TaskAnalysis{}, fmt.Errorf(
			"proposal task is required",
		)
	}

	return TaskAnalysis{
		OriginalTask: task,
		Deliverable:  "知识库文档删除与重新导入技术方案",
		RequiredSections: []string{
			"当前数据模型与调用链依据",
			"删除与重新导入 API",
			"事务一致性",
			"失败处理",
			"最小测试方案",
			"本次不做的范围",
		},
		EvidenceRequirements: []string{
			"必须引用实际读取的项目文件路径",
			"方案结论必须能够追溯到代码证据",
			"不能根据猜测补充当前实现中不存在的能力",
		},
	}, nil
}

// NewAnalysisWorkflow 创建只包含任务分析节点的第一版 Workflow。
func NewAnalysisWorkflow(
	ctx context.Context,
) (
	compose.Runnable[ProposalRequest, TaskAnalysis],
	error,
) {
	workflow := compose.NewWorkflow[
		ProposalRequest,
		TaskAnalysis,
	]()

	workflow.AddLambdaNode(
		nodeAnalyzeTask,
		compose.InvokableLambda(analyzeTask),
	).AddInput(compose.START)

	workflow.End().AddInput(nodeAnalyzeTask)

	runnable, err := workflow.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"compile proposal analysis workflow: %w",
			err,
		)
	}

	return runnable, nil
}
