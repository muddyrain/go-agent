package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// 编译期检查：本节通过指针接收者实现error，不调用nil对象的Error方法。
var _ error = (*proposalInputError)(nil)

func TestProposalInputErrorFieldsAndAs(t *testing.T) {
	original := &proposalInputError{Field: "task", Reason: "is required"}
	if original.Error() != "task is required" {
		t.Fatal(original.Error())
	}
	wrapped := fmt.Errorf("invoke proposal workflow: %w", original)
	var target *proposalInputError
	if _, ok := wrapped.(*proposalInputError); ok {
		t.Fatal("direct assertion must not see a different outer wrapper type")
	}
	if !errors.As(wrapped, &target) || target != original || target.Field != "task" || target.Reason != "is required" {
		t.Fatalf("target=%v", target)
	}
	if wrapped == original || errors.Unwrap(wrapped) != original {
		t.Fatal("As must not replace the original outer error")
	}
	for _, err := range []error{nil, context.Canceled, errors.New("task is required"), fmt.Errorf("invoke: %v", original)} {
		target = nil
		if errors.As(err, &target) || target != nil {
			t.Fatalf("unexpected typed input error: %v", err)
		}
	}
	other := &proposalInputError{Field: "different_field", Reason: "different_reason"}
	if !errors.As(fmt.Errorf("outer: %w", other), &target) || target != other || target.Field != "different_field" || target.Reason != "different_reason" {
		t.Fatalf("fields must come from the found object, not parsing fixed text: %v", target)
	}
}

func TestProposalInputErrorThroughWorkflow(t *testing.T) {
	wf, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range []string{"", " \t\n "} {
		analysis, err := analyzeTask(context.Background(), proposalRequest{Task: task})
		var inputErr *proposalInputError
		if !errors.As(err, &inputErr) || inputErr.Field != "task" || inputErr.Reason != "is required" || analysis.Task != "" || analysis.Steps != nil {
			t.Fatalf("analysis=%v error=%v input=%v", analysis, err, inputErr)
		}
		result, err := wf.Invoke(context.Background(), proposalRequest{Task: task})
		inputErr = nil
		if !errors.As(err, &inputErr) || inputErr.Field != "task" || inputErr.Reason != "is required" || result.Content != "" {
			t.Fatalf("result=%v error=%v input=%v", result, err, inputErr)
		}
		var wrappedInput *proposalInputError
		if !errors.As(fmt.Errorf("CLI diagnostic: %w", err), &wrappedInput) || wrappedInput != inputErr {
			t.Fatal("additional wrapping must preserve the same input error object")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = wf.Invoke(ctx, proposalRequest{Task: ""})
	var inputErr *proposalInputError
	if !errors.Is(err, context.Canceled) || errors.As(err, &inputErr) {
		t.Fatalf("cancellation misclassified: %v", err)
	}
	result, err := wf.Invoke(context.Background(), proposalRequest{Task: "Go"})
	if err != nil || result.Content == "" {
		t.Fatalf("following invocation failed: result=%v error=%v", result, err)
	}
}
