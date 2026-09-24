package techproposal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	nodeAnalyzeTask      = "analyze_task"
	nodeCollectEvidence  = "collect_evidence"
	nodeChatModel        = "chat_model"
	nodeBuildPrompt      = "build_prompt"
	nodeValidateProposal = "validate_proposal"
)

const maxEvidenceFileSize = 128 << 10 // 128 KiB，防止证据节点读入巨型文件

// proposalSections 是技术方案必须包含的章节，analyzeTask 与 validateProposal 共用。
var proposalSections = []string{
	"当前数据模型与调用链依据",
	"删除与重新导入 API",
	"事务一致性",
	"失败处理",
	"最小测试方案",
	"本次不做的范围",
}

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

// ProposalResult 是校验通过后的方案输出。
type ProposalResult struct {
	Content string
	Review  string
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
		OriginalTask:     task,
		Deliverable:      "知识库文档删除与重新导入技术方案",
		RequiredSections: proposalSections,
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

// buildPrompt 把分析要求与代码证据组装成模型可直接消费的消息列表。
func buildPrompt(
	ctx context.Context,
	evidence CollectedEvidence,
) ([]*schema.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	systemMsg := schema.SystemMessage(
		"你是一名资深工程师，正在根据已读取的代码证据生成技术方案。" +
			"必须严格按给定章节组织内容，且每个结论后标注实际依据的文件路径。" +
			"不得编造代码中不存在的能力。",
	)

	var sb strings.Builder

	sb.WriteString("请基于以下代码证据，生成")
	sb.WriteString(evidence.Analysis.Deliverable)
	sb.WriteString("。\n\n必须包含以下章节：\n")
	for i, section := range evidence.Analysis.RequiredSections {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, section)
	}

	sb.WriteString("\n证据要求：\n")
	for _, req := range evidence.Analysis.EvidenceRequirements {
		sb.WriteString("- ")
		sb.WriteString(req)
		sb.WriteString("\n")
	}

	sb.WriteString("\n代码证据：\n")
	for _, f := range evidence.Files {
		fmt.Fprintf(&sb, "--- 文件: %s ---\n%s\n", f.Path, f.Content)
	}

	userMsg := schema.UserMessage(sb.String())

	return []*schema.Message{systemMsg, userMsg}, nil
}

// validateProposal 检查模型输出是否包含全部必需章节，缺章节则报错。
func validateProposal(
	ctx context.Context,
	msg *schema.Message,
) (ProposalResult, error) {
	if err := ctx.Err(); err != nil {
		return ProposalResult{}, fmt.Errorf("validate proposal: %w", err)
	}

	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return ProposalResult{}, fmt.Errorf("proposal content is empty")
	}

	var missing []string
	for _, section := range proposalSections {
		if !strings.Contains(content, section) {
			missing = append(missing, section)
		}
	}

	if len(missing) > 0 {
		return ProposalResult{}, fmt.Errorf(
			"proposal missing sections: %s",
			strings.Join(missing, ", "),
		)
	}

	return ProposalResult{Content: content}, nil
}

// NewAnalysisWorkflow 创建"分析任务 → 收集代码证据"的两节点 Workflow。
func NewAnalysisWorkflow(
	ctx context.Context,
	rootDir string,
	chatModel model.BaseChatModel,
) (
	compose.Runnable[ProposalRequest, ProposalResult],
	error,
) {
	workflow := compose.NewWorkflow[
		ProposalRequest,
		ProposalResult,
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

	workflow.AddLambdaNode(
		nodeBuildPrompt,
		compose.InvokableLambda(buildPrompt),
	).AddInput(nodeCollectEvidence)

	workflow.AddChatModelNode(
		nodeChatModel,
		chatModel,
	).AddInput(nodeBuildPrompt)

	workflow.AddLambdaNode(
		nodeValidateProposal,
		compose.InvokableLambda(validateProposal),
	).AddInput(nodeChatModel)

	review := func(ctx context.Context, result ProposalResult) (ProposalResult, error) {
		return reviewProposal(ctx, result, chatModel)
	}

	workflow.AddLambdaNode(
		"review_proposal",
		compose.InvokableLambda(review),
	).AddInput(nodeValidateProposal)

	workflow.End().AddInput("review_proposal")

	runnable, err := workflow.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"compile proposal analysis workflow: %w",
			err,
		)
	}

	return runnable, nil
}

// reviewProposal 由第二个 Agent 审查技术方案质量。
// 输入是 validate_proposal 输出的 ProposalResult，输出带评审意见的 ProposalResult。
func reviewProposal(ctx context.Context, result ProposalResult, chatModel model.BaseChatModel) (ProposalResult, error) {
	prompt := fmt.Sprintf(`你是一位资深技术评审专家。请审查以下技术方案，判断它是否：
1. 章节结构完整、逻辑清晰
2. 引用了实际代码证据，没有编造
3. 覆盖了所有必要章节（数据模型、API、事务、失败处理、测试方案、非目标）

方案内容：
%s

请给出评审结论：通过 / 打回，并说明理由。`, result.Content)

	reviewMsg, err := chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是一位严格的技术评审专家，负责审查技术方案质量。"),
		schema.UserMessage(prompt),
	})

	if err != nil {
		return ProposalResult{}, fmt.Errorf("review proposal: %w", err)
	}

	return ProposalResult{
		Content: result.Content,    // 原始方案
		Review:  reviewMsg.Content, // 评审意见
	}, nil
}
