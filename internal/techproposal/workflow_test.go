package techproposal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// projectRoot 返回 go test 运行时的项目根（包目录往上两级）。
// go test 跑包时工作目录固定是 internal/techproposal，因此可用相对路径定位仓库根。
func projectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	return root
}

func TestAnalysisWorkflowProducesStructuredRequirements(t *testing.T) {
	ctx := context.Background()

	runnable, err := NewAnalysisWorkflow(ctx, projectRoot(t))
	if err != nil {
		t.Fatalf("NewAnalysisWorkflow() error = %v", err)
	}

	const task = "  请设计知识库文档删除与重新导入技术方案  "
	evidence, err := runnable.Invoke(ctx, ProposalRequest{Task: task})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	// 分析结果现在包裹在 CollectedEvidence.Analysis 里。
	if evidence.Analysis.OriginalTask != strings.TrimSpace(task) {
		t.Fatalf("OriginalTask = %q, want %q",
			evidence.Analysis.OriginalTask, strings.TrimSpace(task))
	}
	if evidence.Analysis.Deliverable != "知识库文档删除与重新导入技术方案" {
		t.Fatalf("Deliverable = %q", evidence.Analysis.Deliverable)
	}
	if len(evidence.Analysis.RequiredSections) != 6 {
		t.Fatalf("RequiredSections length = %d, want 6",
			len(evidence.Analysis.RequiredSections))
	}
	if len(evidence.Analysis.EvidenceRequirements) != 3 {
		t.Fatalf("EvidenceRequirements length = %d, want 3",
			len(evidence.Analysis.EvidenceRequirements))
	}

	// 证据节点必须真的读到 6 个项目文件，且每个文件都有路径和内容。
	if len(evidence.Files) != 6 {
		t.Fatalf("Files length = %d, want 6", len(evidence.Files))
	}
	for _, f := range evidence.Files {
		if f.Path == "" {
			t.Fatal("evidence file has empty path")
		}
		if strings.TrimSpace(f.Content) == "" {
			t.Fatalf("evidence file %q has empty content", f.Path)
		}
	}
}

func TestAnalysisWorkflowRejectsBlankTask(t *testing.T) {
	ctx := context.Background()

	runnable, err := NewAnalysisWorkflow(ctx, projectRoot(t))
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

	runnable, err := NewAnalysisWorkflow(context.Background(), projectRoot(t))
	if err != nil {
		t.Fatalf("NewAnalysisWorkflow() error = %v", err)
	}

	_, err = runnable.Invoke(ctx, ProposalRequest{Task: "设计技术方案"})
	if err == nil {
		t.Fatal("Invoke() error = nil, want context cancellation")
	}
}

// 以下三个用例直接调用包级 collectEvidence，用 t.TempDir() 隔离文件系统，
// 不经过 Eino 图，专门固定证据节点本身的失败边界。

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
	// 写入一个刚超过 128 KiB 的文件。
	big := make([]byte, maxEvidenceFileSize+1)
	if err := os.WriteFile(
		filepath.Join(dir, "big.go"), big, 0o600,
	); err != nil {
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
