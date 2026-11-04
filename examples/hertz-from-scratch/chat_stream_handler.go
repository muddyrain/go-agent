package main

import (
	"context"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
)

type chatDelta struct {
	Content string `json:"content"`
}

func newChatStreamHandler(sessions *sessionStore) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 所有输入错误都在切换SSE模式前处理，仍可返回普通HTTP错误。
		mediaType, _, err := mime.ParseMediaType(string(c.GetHeader("Content-Type")))
		if err != nil || mediaType != "application/json" {
			c.String(http.StatusUnsupportedMediaType, "Content-Type must be application/json\n")
			return
		}

		body := c.Request.Body()
		// 正文已接收；这是业务大小检查，不是读取过程限内存。
		if len(body) > 4*1024 {
			c.String(http.StatusRequestEntityTooLarge, "request body too large\n")
			return
		}

		var req chatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			c.String(http.StatusBadRequest, "invalid JSON request\n")
			return
		}

		message := strings.TrimSpace(req.Message)
		if message == "" {
			c.String(http.StatusBadRequest, "message is required\n")
			return
		}
		sessionID := strings.TrimSpace(req.SessionID)

		answer, err := replyToMessage(message)
		if err != nil {
			// 当前只有可公开的指令输入错误；不直接套用于未来模型内部故障。
			c.String(http.StatusBadRequest, "%s\n", err.Error())
			return
		}

		response := chatResponse{Answer: answer}

		if sessionID != "" {
			// 与普通/chat共享成功处理计数；后续取消或写入失败不会回滚。
			response.SessionID = sessionID
			response.Turn = sessions.nextTurn(sessionID)
		}

		writer := sse.NewWriter(c)

		// 完整回答已算好；这里只模拟逐段发送，不是模型逐token生成。
		ticker := time.NewTicker(200 * time.Millisecond)

		defer ticker.Stop()

		for i, ch := range []rune(answer) {
			if i > 0 {
				select {
				case <-ctx.Done():
				case <-ticker.C:
				}
			}
			if err := ctx.Err(); err != nil {
				log.Printf("chat stream canceled: %v", err)
				return
			}
			data, err := json.Marshal(chatDelta{Content: string(ch)})
			if err != nil {
				log.Printf("encode chat delta: %v", err)
				return
			}

			if err := writer.WriteEvent("", "delta", data); err != nil {
				log.Printf("write chat delta: %v", err)
				return
			}
		}

		if err := ctx.Err(); err != nil {
			log.Printf("chat stream canceled: %v", err)
			return
		}

		data, err := json.Marshal(response)
		if err != nil {
			log.Printf("encode chat done: %v", err)
			return
		}

		if err := writer.WriteEvent("", "done", data); err != nil {
			log.Printf("write chat done: %v", err)
			return
		}
	}
}
