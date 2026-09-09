package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Executor struct {
	registry *Registry
}

func NewExecutor(registry *Registry) (*Executor, error) {
	if registry == nil {
		return nil, fmt.Errorf("tool registry is required")
	}

	return &Executor{
		registry: registry,
	}, nil
}

func (e *Executor) Execute(
	ctx context.Context,
	call Call,
) (Result, error) {
	call.ID = strings.TrimSpace(call.ID)
	call.Name = strings.TrimSpace(call.Name)

	if call.ID == "" {
		return Result{}, fmt.Errorf("tool call ID is required")
	}

	if call.Name == "" {
		return Result{}, fmt.Errorf("tool call name is required")
	}

	if len(call.Arguments) == 0 {
		return Result{}, fmt.Errorf("tool call arguments are required")
	}

	if !json.Valid(call.Arguments) {
		return Result{}, fmt.Errorf("tool call arguments must be valid JSON")
	}

	registeredTool, ok := e.registry.Get(call.Name)
	if !ok {
		return Result{}, fmt.Errorf(
			"tool %q is not registered",
			call.Name,
		)
	}

	// 参数不符合 Schema 属于模型可修正的调用错误，通过 Result 回传而不中断 Agent Loop。
	if err := e.registry.Validate(
		call.Name,
		call.Arguments,
	); err != nil {
		return Result{
			CallID:  call.ID,
			Name:    call.Name,
			Content: err.Error(),
			IsError: true,
		}, nil
	}

	content, err := registeredTool.Execute(ctx, call.Arguments)
	if err != nil {
		// 取消和超时属于流程控制信号，必须立即向上传播，不能包装成工具业务结果。
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return Result{}, err
		}

		// 普通工具错误交给模型解释或重试，因此转换为 IsError Result。
		return Result{
			CallID:  call.ID,
			Name:    call.Name,
			Content: err.Error(),
			IsError: true,
		}, nil
	}

	return Result{
		CallID:  call.ID,
		Name:    call.Name,
		Content: content,
		IsError: false,
	}, nil
}
