package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	message := flag.String("message", "/upper 你好 Go", "发送给服务端的消息")
	sessionID := flag.String("session", "", "可选的会话ID")
	flag.Parse()

	// 包含连接、响应头和整个响应体读取的总时限，适用于短消息教学。
	client := &http.Client{Timeout: 10 * time.Second}

	if err := sendChatStream(
		client,
		"http://127.0.0.1:8091/chat/stream",
		*message,
		*sessionID,
		os.Stdout,
	); err != nil {
		fmt.Fprintln(os.Stderr, "请求失败：", err)
		os.Exit(1)
	}
}
