package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Function struct {
	definition Definition
	handler    Handler
}

type Handler func(
	ctx context.Context,
	arguments json.RawMessage,
) (string, error)

func NewFunction(
	definition Definition,
	handler Handler,
) (*Function, error) {
	definition.Name = strings.TrimSpace(definition.Name)
	definition.Description = strings.TrimSpace(definition.Description)

	if definition.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}

	if definition.Description == "" {
		return nil, fmt.Errorf("tool description is required")
	}

	if len(definition.Parameters) == 0 {
		return nil, fmt.Errorf("tool parameters are required")
	}

	if !json.Valid(definition.Parameters) {
		return nil, fmt.Errorf("tool parameters must be valid JSON")
	}

	if handler == nil {
		return nil, fmt.Errorf("tool handler is required")
	}

	return &Function{
		definition: definition,
		handler:    handler,
	}, nil
}

func (f *Function) Definition() Definition {
	return f.definition
}

func (f *Function) Execute(
	ctx context.Context,
	arguments json.RawMessage,
) (string, error) {
	return f.handler(ctx, arguments)
}

var _ Tool = (*Function)(nil)
