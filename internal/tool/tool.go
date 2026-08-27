package tool

import (
	"context"
	"encoding/json"
)

type Definition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

type Call struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type Result struct {
	CallID  string
	Name    string
	Content string
	IsError bool
}

type Tool interface {
	Definition() Definition
	Execute(
		ctx context.Context,
		arguments json.RawMessage,
	) (string, error)
}
