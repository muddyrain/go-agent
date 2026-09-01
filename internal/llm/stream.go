package llm

import (
	"errors"
	"fmt"
	"io"
)

type StreamHandler func(event StreamEvent) error

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

	// 退出函数前关闭 stream
	defer func() {
		closeErr := stream.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close stream: %w", closeErr)
		}
	}()
	for {
		// 调用 stream.Recv()
		event, recvErr := stream.Recv()

		// 区分 io.EOF 和真正的接收错误
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

		// 校验事件类型
		switch event.Type {
		case StreamEventDelta, StreamEventDone:
			// 合法事件
		default:
			return Response{}, fmt.Errorf(
				"unsupported stream event type %q",
				event.Type,
			)
		}

		// 把事件交给 handler
		if handleErr := handler(event); handleErr != nil {
			return Response{}, fmt.Errorf(
				"handle stream event: %w",
				handleErr,
			)
		}

		// 遇到 done 返回最终 Response
		if event.Type == StreamEventDone {
			return event.Response, nil
		}
	}
}
