package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// 仅测试类型：具备Error方法即可隐式实现error，不用于业务封装。
type lessonError struct{ message string }

func (e lessonError) Error() string { return e.message }

var _ error = lessonError{}

func TestErrorInterfaceAndIdentity(t *testing.T) {
	var err error = lessonError{message: "context canceled"}
	if err.Error() != "context canceled" {
		t.Fatal(err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatal("same text must not imply cancellation identity")
	}
	wrapped := fmt.Errorf("analyze task: %w", context.Canceled)
	if wrapped == context.Canceled || !errors.Is(wrapped, context.Canceled) {
		t.Fatal("wrapped cancellation must be detected without direct equality")
	}
	textOnly := fmt.Errorf("analyze task: %v", context.Canceled)
	if textOnly.Error() != wrapped.Error() || errors.Is(textOnly, context.Canceled) || errors.Unwrap(textOnly) != nil {
		t.Fatal("same visible text must not imply the same error chain")
	}
	twiceWrapped := fmt.Errorf("invoke proposal workflow: %w", wrapped)
	if !errors.Is(twiceWrapped, context.Canceled) || errors.Unwrap(twiceWrapped) != wrapped {
		t.Fatal("additional context must preserve cancellation identity")
	}
}

func TestNodeCancellationChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := analyzeTask(ctx, proposalRequest{Task: "Go"})
	if err == nil || err == context.Canceled || !errors.Is(err, context.Canceled) || errors.Unwrap(err) != context.Canceled {
		t.Fatalf("analyze cancellation chain: %v", err)
	}
	_, err = buildProposal(ctx, taskAnalysis{Task: "Go"})
	if err == nil || err == context.Canceled || !errors.Is(err, context.Canceled) {
		t.Fatalf("build cancellation chain: %v", err)
	}
	wf, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = wf.Invoke(ctx, proposalRequest{Task: "Go"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("workflow cancellation chain: %v", err)
	}
}
