package llm

import (
	"errors"
	"fmt"
	"io"
)

// StreamHandler 接收模型产生的每个合法流事件。
// 具体事件如何输出由调用方决定，例如写入终端、SSE 或 WebSocket。
type StreamHandler func(event StreamEvent) error

// ConsumeStream 读取并校验一次完整模型流，将每个事件交给 Handler，
// 在收到 Done 后返回本轮完整 Response，并保证取得流所有权后最终关闭资源。
func ConsumeStream(
	stream Stream,
	handler StreamHandler,
) (response Response, err error) {
	if stream == nil {
		return Response{}, fmt.Errorf("stream is required")
	}
	if handler == nil {
		return Response{}, fmt.Errorf("stream handler is required")
	}

	// 命名返回值 err 允许 defer 在主流程成功时补充 Close 错误。
	// 如果接收或 Handler 已经失败，则保留更能解释失败原因的主错误。
	defer func() {
		closeErr := stream.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close stream: %w", closeErr)
		}
	}()
	for {
		// Recv 每次只读取下一个事件；一次模型响应通常需要多次读取。
		event, recvErr := stream.Recv()

		// EOF 只表示没有更多数据，不证明模型流已经按协议正常完成。
		// 只有显式 Done 事件才携带可交给 Agent Loop 的完整 Response。
		if errors.Is(recvErr, io.EOF) {
			return Response{}, fmt.Errorf(
				"stream ended before done event",
			)
		}
		if recvErr != nil {
			return Response{}, fmt.Errorf(
				"receive stream event: %w",
				recvErr,
			)
		}

		// 在事件进入外部 Handler 前完成协议校验，
		// 防止调用方处理当前 Runtime 尚未支持的事件类型。
		switch event.Type {
		case StreamEventDelta, StreamEventDone:
			// 合法事件
		default:
			return Response{}, fmt.Errorf(
				"unsupported stream event type %q",
				event.Type,
			)
		}

		// Handler 决定事件的输出方式；它返回错误通常表示下游无法继续接收，
		// 例如客户端断开连接或输出目标写入失败。
		if handleErr := handler(event); handleErr != nil {
			return Response{}, fmt.Errorf(
				"handle stream event: %w",
				handleErr,
			)
		}

		// Done 结束当前一次模型流并提供完整 Response。
		// Response 是否还包含 ToolCalls，由外层 Agent Loop 继续判断。
		if event.Type == StreamEventDone {
			return event.Response, nil
		}
	}
}
