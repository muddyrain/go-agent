package tool

import (
	"context"
	"encoding/json"
	"time"
)

type TimeTool struct{}

func (t *TimeTool) Name() string {
	return "get_current_time"
}

func (t *TimeTool) Description() string {
	return "获取当前时间"
}

func (t *TimeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (t *TimeTool) Execute(ctx context.Context, arguments json.RawMessage) (string, error) {
	return time.Now().Format("2006-01-02 15:04:05"), nil
}
