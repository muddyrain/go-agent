package session

import (
	"fmt"

	"github.com/cloudwego/eino/schema"
)

type ContextView struct {
	// 本次真正发送给模型的消息
	Messages []*schema.Message
	// 完整历史消息数量
	TotalMessages int
	// 模型视图中的消息数量
	KeptMessages int
	// 本次没有发送给模型的旧消息数量
	DroppedMessages int
	// 实际保留的用户轮次数
	KeptTurns int
	// 为什么发生或没有发生裁剪
	Reason string
}

type ContextWindow struct {
	maxTurns int
}

func NewContextWindow(maxTurns int) (*ContextWindow, error) {
	if maxTurns <= 0 {
		return nil, fmt.Errorf("max turns must be greater than zero")
	}
	return &ContextWindow{maxTurns: maxTurns}, nil
}

func (w *ContextWindow) BuildModelView(
	history []*schema.Message,
) ContextView {
	view := ContextView{
		TotalMessages: len(history),
	}

	if len(history) == 0 {
		view.Messages = []*schema.Message{}
		view.Reason = "history is empty"
		return view
	}

	scanStart := 0
	hasSystem := history[0].Role == schema.System

	if hasSystem {
		scanStart = 1
	}

	turnStarts := make([]int, 0)
	for i := scanStart; i < len(history); i++ {
		if history[i].Role == schema.User {
			turnStarts = append(turnStarts, i)
		}
	}

	keepFrom := scanStart

	if len(turnStarts) > w.maxTurns {
		keepFrom = turnStarts[len(turnStarts)-w.maxTurns]
	}

	capacity := len(history) - keepFrom
	if hasSystem {
		capacity++
	}
	messages := make([]*schema.Message, 0, capacity)
	if hasSystem {
		messages = append(messages, history[0])
	}
	messages = append(messages, history[keepFrom:]...)

	keptTurns := len(turnStarts)
	if keptTurns > w.maxTurns {
		keptTurns = w.maxTurns
	}

	view.Messages = messages
	view.KeptMessages = len(messages)
	view.DroppedMessages = len(history) - len(messages)
	view.KeptTurns = keptTurns

	if view.DroppedMessages == 0 {
		view.Reason = "all messages fit in the context window"
		return view
	}
	view.Reason = fmt.Sprintf(
		"kept the system message and latest %d user turns",
		keptTurns,
	)
	return view
}
