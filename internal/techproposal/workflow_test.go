package techproposal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// projectRoot 返回 go test 运行时的项目根（包目录往上两级）。
func projectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	return root
}

// fakeChatModel 是不打网络的测试模型。
// 它记录收到的 messages，让测试能断言 build_prompt 节点真的把证据拼进去了。
type fakeChatModel struct {
	replyContent string
	lastInput    []*schema.Message
}

func (m *fakeChatModel) Generate(
	ctx context.Context,
	input []*schema.Message,
	opts ...model.Option,
) (*schema.Message, error) {
	m.lastInput = input
	return schema.AssistantMessage(m.replyContent, nil), nil
}

func (m *fakeChatModel) Stream(
	ctx context.Context,
	input []*schema.Message,
	opts ...model.Option,
) (*schema.StreamReader[*schema.Message], error) {
	// Invoke 路径只走 Generate；Stream 仅为满足接口，用 Pipe 立即发出单条消息。
	msg, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	reader, writer := schema.Pipe[*schema.Message](1)
	writer.Send(msg, nil)
	writer.Close()
	return reader, nil
}

func TestAnalysisWorkflowEndToEnd(t *testing.T) {
	ctx := context.Background()
	// fake 回复必须包含全部章节关键词，validate_proposal 才会放行。
	reply := "技术方案：当前数据模型与调用链依据已分析。删除与重新导入 API 已设计。" +
		"事务一致性有保障。失败处理已覆盖。最小测试方案已写。本次不做的范围已说明。"
	fake := &fakeChatModel{replyContent: reply}

	runnable, err := NewAnalysisWorkflow(ctx, projectRoot(t), fake)
	if err != nil {
		t.Fatalf("NewAnalysisWorkflow() error = %v", err)
	}

	result, err := runnable.Invoke(ctx, ProposalRequest{Task: "设计知识库文档删除与重新导入技术方案"})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	// 最终输出是校验通过后的 ProposalResult.Content。
	if result.Content != reply {
		t.Fatalf("result.Content = %q", result.Content)
	}

	// fake model 应收到 system + user 两条消息。
	if len(fake.lastInput) != 2 {
		t.Fatalf("fake received %d messages, want 2", len(fake.lastInput))
	}

	// user message 必须带上证据文件路径和章节要求，证明 build_prompt 真的拼进去了。
	userContent := fake.lastInput[1].Content
	if !strings.Contains(userContent, "internal/knowledge/document.go") {
		t.Fatalf("user prompt missing evidence path; head: %q", truncate(userContent, 200))
	}
	if !strings.Contains(userContent, "必须包含以下章节") {
		t.Fatal("user prompt missing required sections")
	}
}

func TestValidateProposalRejectsMissingSections(t *testing.T) {
	ctx := context.Background()
	// 只写了两章，其余缺失。
	fake := &fakeChatModel{replyContent: "当前数据模型与调用链依据已分析。删除与重新导入 API 已设计。"}

	runnable, err := NewAnalysisWorkflow(ctx, projectRoot(t), fake)
	if err != nil {
		t.Fatalf("NewAnalysisWorkflow() error = %v", err)
	}

	_, err = runnable.Invoke(ctx, ProposalRequest{Task: "设计技术方案"})
	if err == nil {
		t.Fatal("Invoke() error = nil, want missing-sections error")
	}
	if !strings.Contains(err.Error(), "missing sections") {
		t.Fatalf("err = %q, want missing-sections detail", err)
	}
}

func TestValidateProposalRejectsEmptyContent(t *testing.T) {
	ctx := context.Background()
	fake := &fakeChatModel{replyContent: "   "}

	runnable, err := NewAnalysisWorkflow(ctx, projectRoot(t), fake)
	if err != nil {
		t.Fatalf("NewAnalysisWorkflow() error = %v", err)
	}

	_, err = runnable.Invoke(ctx, ProposalRequest{Task: "设计技术方案"})
	if err == nil {
		t.Fatal("Invoke() error = nil, want empty-content error")
	}
	if !strings.Contains(err.Error(), "proposal content is empty") {
		t.Fatalf("err = %q, want empty-content detail", err)
	}
}

func TestAnalysisWorkflowRejectsBlankTask(t *testing.T) {
	ctx := context.Background()
	fake := &fakeChatModel{replyContent: "不应被调用"}

	runnable, err := NewAnalysisWorkflow(ctx, projectRoot(t), fake)
	if err != nil {
		t.Fatalf("NewAnalysisWorkflow() error = %v", err)
	}

	_, err = runnable.Invoke(ctx, ProposalRequest{Task: " \n\t "})
	if err == nil {
		t.Fatal("Invoke() error = nil, want blank-task error")
	}
	if !strings.Contains(err.Error(), "proposal task is required") {
		t.Fatalf("Invoke() error = %q, want blank-task detail", err)
	}
}

func TestAnalysisWorkflowPropagatesCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fake := &fakeChatModel{replyContent: "不应被调用"}
	runnable, err := NewAnalysisWorkflow(context.Background(), projectRoot(t), fake)
	if err != nil {
		t.Fatalf("NewAnalysisWorkflow() error = %v", err)
	}

	_, err = runnable.Invoke(ctx, ProposalRequest{Task: "设计技术方案"})
	if err == nil {
		t.Fatal("Invoke() error = nil, want context cancellation")
	}
}

// 以下三个用例直接调用包级 collectEvidence，用 t.TempDir() 隔离文件系统。

func TestCollectEvidenceRejectsEscapingPath(t *testing.T) {
	dir := t.TempDir()

	_, err := collectEvidence(context.Background(), TaskAnalysis{
		RequiredFiles: []string{"../outside.txt"},
	}, dir)
	if err == nil {
		t.Fatal("collectEvidence() error = nil, want escaping-path error")
	}
	if !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("err = %q, want escapes-root detail", err)
	}
}

func TestCollectEvidenceReportsMissingFile(t *testing.T) {
	dir := t.TempDir()

	_, err := collectEvidence(context.Background(), TaskAnalysis{
		RequiredFiles: []string{"does_not_exist.go"},
	}, dir)
	if err == nil {
		t.Fatal("collectEvidence() error = nil, want missing-file error")
	}
	if !strings.Contains(err.Error(), "does_not_exist.go") {
		t.Fatalf("err = %q, want missing file path in error", err)
	}
}

func TestCollectEvidenceEnforcesSizeLimit(t *testing.T) {
	dir := t.TempDir()
	big := make([]byte, maxEvidenceFileSize+1)
	if err := os.WriteFile(filepath.Join(dir, "big.go"), big, 0o600); err != nil {
		t.Fatalf("write big fixture: %v", err)
	}

	_, err := collectEvidence(context.Background(), TaskAnalysis{
		RequiredFiles: []string{"big.go"},
	}, dir)
	if err == nil {
		t.Fatal("collectEvidence() error = nil, want size-limit error")
	}
	if !strings.Contains(err.Error(), "exceeds size limit") {
		t.Fatalf("err = %q, want size-limit detail", err)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
