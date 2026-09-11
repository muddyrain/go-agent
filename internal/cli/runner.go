package cli

import (
	"agenthub/internal/session"
	"agenthub/internal/toolcatalog"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

const maxContextTurns = 3

func Run(
	ctx context.Context,
	reactAgent *react.Agent,
	toolCatalog *toolcatalog.Catalog,
	systemPrompt string,
	input io.Reader,
	output io.Writer,
) error {
	// history 保存终端会话的完整消息事实；ContextWindow 每轮只基于它
	// 生成临时模型视图。不能用裁剪后的视图覆盖 history，否则被暂时丢弃
	// 的旧消息将永久丢失，未来也无法更换上下文策略或持久化完整会话。
	history := []*schema.Message{
		schema.SystemMessage(
			systemPrompt,
		),
	}
	contextWindow, err := session.NewContextWindow(maxContextTurns)
	if err != nil {
		return fmt.Errorf("create context window: %w", err)
	}
	reader := bufio.NewReader(input)

	for {
		fmt.Fprint(output, "\n> ")

		line, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(output, "\nbye")
			return nil
		}
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "/tools") {
			// /tools 是本地 CLI 控制命令，不属于用户与模型的对话。
			// 必须在创建 UserMessage 前截获，避免污染 history 或触发模型。
			if err := printToolCatalog(ctx, output, toolCatalog); err != nil {
				fmt.Fprintf(output, "error: %v\n", err)
			}
			continue
		}

		if strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit") {
			fmt.Fprintln(output, "bye")
			return nil
		}
		userMessage := schema.UserMessage(line)
		history = append(history, userMessage)

		// BuildModelView 只保留系统消息和最近若干个完整用户轮次。
		// 完整轮次以 UserMessage 为边界，避免裁剪后从 Assistant 或 Tool
		// 消息开始，导致模型看到缺少提问或 ToolCall 的残缺上下文。
		contextView := contextWindow.BuildModelView(history)

		fmt.Printf(
			"context: history=%d model=%d dropped=%d turns=%d reason=%s\n",
			contextView.TotalMessages,
			contextView.KeptMessages,
			contextView.DroppedMessages,
			contextView.KeptTurns,
			contextView.Reason,
		)

		// Stream 提供给终端的是最终 Assistant 文本流；MessageFuture 额外
		// 保留 ReAct 内部产生的完整 Assistant ToolCall、ToolResult 和最终
		// Assistant 消息，供本轮结束后写回会话历史。
		msgOpt, future := react.WithMessageFuture()

		stream, err := reactAgent.Stream(
			ctx,
			contextView.Messages,
			msgOpt,
			agent.WithComposeOptions(
				compose.WithCallbacks(newLifecycleCallback(output)),
			),
		)
		if err != nil {
			fmt.Fprintf(output, "error: %v\n", err)
			// 本轮还没有形成完整 Assistant 回复，撤回刚加入的 UserMessage，
			// 避免下轮 history 出现“只有提问、没有回答”的失败半轮。
			history = history[:len(history)-1]
			continue
		}

		fmt.Fprintf(output, "user: %s\n", line)
		fmt.Fprint(output, "assistant: ")

		chunks, err := writeAssistantStream(output, stream)
		if err != nil {
			fmt.Fprintf(output, "\nerror: %v\n", err)
			// 流中途失败时，用户可能已经看到部分文字，但它不是一条完整、
			// 可复用的 Assistant 消息，因此本轮 UserMessage 也不写入历史。
			history = history[:len(history)-1]
			continue
		}

		fmt.Fprintln(output)
		fmt.Fprintf(output, "chunks: %d\n", chunks)

		// 终端流只负责让用户尽快看到最终文本；完整历史必须从
		// MessageFuture 收集。一次工具闭环通常包含：Assistant ToolCall →
		// ToolResult → 最终 Assistant。每个消息自身也是增量流，需先用
		// ConcatMessages 合并，再按产生顺序写回 history。
		iter := future.GetMessageStreams()
		for {
			msgStream, hasNext, err := iter.Next()
			if err != nil {
				return fmt.Errorf("collect agent messages: %w", err)
			}
			if !hasNext {
				break
			}

			var roundMsgs []*schema.Message
			for {
				msg, err := msgStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return fmt.Errorf("read agent message stream: %w", err)
				}
				roundMsgs = append(roundMsgs, msg)
			}
			msgStream.Close()

			if len(roundMsgs) == 0 {
				continue
			}
			concated, err := schema.ConcatMessages(roundMsgs)
			if err != nil {
				return fmt.Errorf("concat agent message: %w", err)
			}
			history = append(history, concated)
		}
	}
}

// newLifecycleCallback 只观察 ChatModel 和 Tool 两类关键组件，帮助学习
// ReAct 的 Model → Tool → Model 顺序；Callback 不参与业务控制和消息保存。
func newLifecycleCallback(output io.Writer) callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			_ callbacks.CallbackInput,
		) context.Context {
			if shouldObserve(info) {
				fmt.Fprintf(output,
					"callback: start component=%s name=%s\n",
					info.Component,
					info.Name,
				)
			}
			return ctx
		}).
		OnEndFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			_ callbacks.CallbackOutput,
		) context.Context {
			if shouldObserve(info) {
				fmt.Fprintf(output,
					"callback: end component=%s name=%s\n",
					info.Component,
					info.Name,
				)
			}
			return ctx
		}).
		OnEndWithStreamOutputFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			streamOutput *schema.StreamReader[callbacks.CallbackOutput],
		) context.Context {
			defer streamOutput.Close()

			if shouldObserve(info) {
				fmt.Fprintf(output,
					"callback: stream_ready component=%s name=%s\n",
					info.Component,
					info.Name,
				)
			}
			return ctx
		}).
		OnErrorFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			err error,
		) context.Context {
			if shouldObserve(info) {
				fmt.Fprintf(output,
					"callback: error component=%s name=%s error=%v\n",
					info.Component,
					info.Name,
					err,
				)
			}
			return ctx
		}).
		Build()
}

func shouldObserve(info *callbacks.RunInfo) bool {
	if info == nil {
		return false
	}

	return info.Component == components.ComponentOfChatModel ||
		info.Component == components.ComponentOfTool
}
