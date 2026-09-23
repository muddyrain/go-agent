package techproposal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/compose"
)

const (
	nodeAnalyzeTask     = "analyze_task"
	nodeCollectEvidence = "collect_evidence"
)

const maxEvidenceFileSize = 128 << 10 // 128 KiB，防止证据节点读入巨型文件

// EvidenceFile 是一份已读取的代码证据，Path 会在方案里被引用。
type EvidenceFile struct {
	Path    string
	Content string
}

// CollectedEvidence 把分析结果和证据一起交给后续节点。
// Workflow 里数据只顺流不回流，所以后续方案节点需要的章节要求、交付物定义
// 也要随证据一起打包带下去，不能再回到 analyze_task 去取。
type CollectedEvidence struct {
	Analysis TaskAnalysis
	Files    []EvidenceFile
}

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
	RequiredFiles        []string // 由分析节点决定证据节点要读哪些文件
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
		RequiredFiles: []string{
			"internal/knowledge/document.go",
			"internal/knowledge/chunker.go",
			"internal/knowledge/embedder.go",
			"internal/knowledge/search_tool.go",
			"internal/httpapi/document_handler.go",
			"configs/migrations/000002_create_documents.up.sql",
		},
	}, nil
}

// collectEvidence 按分析结果固定读取项目文件，产出带路径的代码证据。
// rootDir 是项目根，由构造函数闭包绑定，不依赖进程工作目录。
func collectEvidence(
	ctx context.Context,
	analysis TaskAnalysis,
	rootDir string,
) (CollectedEvidence, error) {
	if err := ctx.Err(); err != nil {
		return CollectedEvidence{}, fmt.Errorf("collect evidence: %w", err)
	}
	files := make([]EvidenceFile, 0, len(analysis.RequiredFiles))
	for _, path := range analysis.RequiredFiles {
		abs := filepath.Join(rootDir, path)

		rel, err := filepath.Rel(rootDir, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return CollectedEvidence{}, fmt.Errorf(
				"collect evidence path escapes root: %q", path,
			)
		}
		content, err := os.ReadFile(abs)
		if err != nil {
			return CollectedEvidence{}, fmt.Errorf(
				"collect evidence read %q: %w", path, err,
			)
		}
		if len(content) > maxEvidenceFileSize {
			return CollectedEvidence{}, fmt.Errorf(
				"collect evidence file %q exceeds size limit: %d bytes",
				path, len(content),
			)
		}

		files = append(files, EvidenceFile{
			Path:    path,
			Content: string(content),
		})
	}
	return CollectedEvidence{
		Analysis: analysis,
		Files:    files,
	}, nil
}

// NewAnalysisWorkflow 创建"分析任务 → 收集代码证据"的两节点 Workflow。
func NewAnalysisWorkflow(
	ctx context.Context,
	rootDir string,
) (
	compose.Runnable[ProposalRequest, CollectedEvidence],
	error,
) {
	workflow := compose.NewWorkflow[
		ProposalRequest,
		CollectedEvidence,
	]()

	workflow.AddLambdaNode(
		nodeAnalyzeTask,
		compose.InvokableLambda(analyzeTask),
	).AddInput(compose.START)

	// 闭包把 rootDir 绑进节点；真实逻辑在包级 collectEvidence 里，便于单测。
	collect := func(ctx context.Context, analysis TaskAnalysis) (CollectedEvidence, error) {
		return collectEvidence(ctx, analysis, rootDir)
	}

	workflow.AddLambdaNode(
		nodeCollectEvidence,
		compose.InvokableLambda(collect),
	).AddInput(nodeAnalyzeTask)

	workflow.End().AddInput(nodeCollectEvidence)

	runnable, err := workflow.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"compile proposal analysis workflow: %w",
			err,
		)
	}

	return runnable, nil
}
