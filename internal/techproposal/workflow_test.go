package techproposal

import (
	"context"
	"strings"
	"testing"
)

func TestAnalysisWorkflowProducesStructuredRequirements(t *testing.T) {
	ctx := context.Background()

	runnable, err := NewAnalysisWorkflow(ctx)
	if err != nil {
		t.Fatalf("NewAnalysisWorkflow() error = %v", err)
	}

	const task = "  请设计知识库文档删除与重新导入技术方案  "
	analysis, err := runnable.Invoke(ctx, ProposalRequest{Task: task})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	if analysis.OriginalTask != strings.TrimSpace(task) {
		t.Fatalf("OriginalTask = %q, want %q", analysis.OriginalTask, strings.TrimSpace(task))
	}
	if analysis.Deliverable != "知识库文档删除与重新导入技术方案" {
		t.Fatalf("Deliverable = %q", analysis.Deliverable)
	}
	if len(analysis.RequiredSections) != 6 {
		t.Fatalf("RequiredSections length = %d, want 6", len(analysis.RequiredSections))
	}
	if len(analysis.EvidenceRequirements) != 3 {
		t.Fatalf("EvidenceRequirements length = %d, want 3", len(analysis.EvidenceRequirements))
	}
}

func TestAnalysisWorkflowRejectsBlankTask(t *testing.T) {
	ctx := context.Background()

	runnable, err := NewAnalysisWorkflow(ctx)
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

	runnable, err := NewAnalysisWorkflow(context.Background())
	if err != nil {
		t.Fatalf("NewAnalysisWorkflow() error = %v", err)
	}

	_, err = runnable.Invoke(ctx, ProposalRequest{Task: "设计技术方案"})
	if err == nil {
		t.Fatal("Invoke() error = nil, want context cancellation")
	}
}
