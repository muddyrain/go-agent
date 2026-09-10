package session

import (
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// ContextView 是根据完整会话历史生成的单次模型输入视图。
// Messages 可能丢弃较旧轮次，但调用方持有的原始 history 不会被修改；
// 其余字段用于向 CLI 解释本轮保留和裁剪了多少上下文。
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

// ContextWindow 按最近完整用户轮次限制单次模型输入。
// 它只负责生成视图，不拥有也不保存完整会话历史。
type ContextWindow struct {
	maxTurns int
}

// NewContextWindow 创建按用户轮次数裁剪的上下文窗口。
// maxTurns 表示最多保留多少个以 UserMessage 开始的轮次。
func NewContextWindow(maxTurns int) (*ContextWindow, error) {
	if maxTurns <= 0 {
		return nil, fmt.Errorf("max turns must be greater than zero")
	}
	return &ContextWindow{maxTurns: maxTurns}, nil
}

// BuildModelView 从完整 history 中生成本轮发送给模型的消息视图。
// 裁剪边界只选择 UserMessage 的位置，因此同一轮中的 Assistant ToolCall、
// ToolResult 和最终 Assistant 不会被拆散；首条 SystemMessage 始终单独保留。
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

	// SystemMessage 不属于任何用户轮次。若它位于第一条，则跳过它寻找
	// UserMessage 边界，最后再把它单独放回模型视图开头。
	scanStart := 0
	hasSystem := history[0].Role == schema.System

	if hasSystem {
		scanStart = 1
	}

	// 每个 UserMessage 都代表一个新轮次的起点。记录起点而不是直接按
	// 消息条数截取，才能保留该轮后续完整的 ToolCall 与 ToolResult 链。
	turnStarts := make([]int, 0)
	for i := scanStart; i < len(history); i++ {
		if history[i].Role == schema.User {
			turnStarts = append(turnStarts, i)
		}
	}

	// 默认从第一条非系统消息开始保留；轮次超过上限时，移动到倒数
	// maxTurns 个 UserMessage 的位置。这样视图不会从 Assistant 或 Tool 开始。
	keepFrom := scanStart

	if len(turnStarts) > w.maxTurns {
		keepFrom = turnStarts[len(turnStarts)-w.maxTurns]
	}

	// 预估新切片容量只为减少扩容；messages 拥有独立的切片底层数组，
	// 其中仍然引用原 Message 指针，但 append 不会改写 history 的切片结构。
	capacity := len(history) - keepFrom
	if hasSystem {
		capacity++
	}
	messages := make([]*schema.Message, 0, capacity)
	if hasSystem {
		messages = append(messages, history[0])
	}
	// history[keepFrom:] 是从保留边界到末尾的切片；末尾的 ... 会把
	// 其中每个 Message 指针逐个追加到 messages，而不是追加一个切片对象。
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
