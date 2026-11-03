package main

import (
	"errors"
	"strings"
)

// replyToMessage 接收已去两端空白且非空的消息，只处理文本，不读写HTTP。
func replyToMessage(message string) (string, error) {
	if message == "/upper" || strings.HasPrefix(message, "/upper ") {
		text := strings.TrimSpace(strings.TrimPrefix(message, "/upper"))
		if text == "" {
			return "", errors.New("text is required after /upper")
		}

		return strings.ToUpper(text), nil
	}

	return "收到：" + message, nil
}
