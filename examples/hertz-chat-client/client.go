package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

type streamRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

type streamDelta struct {
	Content string `json:"content"`
}

type streamResult struct {
	Answer    string `json:"answer"`
	SessionID string `json:"session_id"`
	Turn      int    `json:"turn"`
}

func sendChatStream(client *http.Client, url, message, sessionID string, out io.Writer) error {
	body, err := json.Marshal(streamRequest{
		Message:   message,
		SessionID: sessionID,
	})
	if err != nil {
		return fmt.Errorf("encode chat request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send chat request: %w", err)
	}
	defer resp.Body.Close()
	// HTTP 400/500不会自动成为Do的error，必须主动检查状态码。
	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if err != nil {
			return fmt.Errorf("read HTTP error body: %w", err)
		}
		return fmt.Errorf(
			"HTTP %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(data)),
		)
	}

	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || mediaType != "text/event-stream" {
		return fmt.Errorf(
			"unexpected Content-Type: %q",
			resp.Header.Get("Content-Type"),
		)
	}
	if _, err := fmt.Fprint(out, "回答："); err != nil {
		return fmt.Errorf("write answer prefix: %w", err)
	}

	result, err := readChatStream(resp.Body, out)
	if err != nil {
		return err
	}

	if result.SessionID != strings.TrimSpace(sessionID) {
		return errors.New("done session_id does not match request")
	}

	if result.SessionID == "" && result.Turn != 0 ||
		result.SessionID != "" && result.Turn < 1 {
		return errors.New("invalid session turn in done")
	}

	if _, err := fmt.Fprint(out, "\n完成\n"); err != nil {
		return fmt.Errorf("write completion: %w", err)
	}

	if result.SessionID != "" {
		if _, err := fmt.Fprintf(
			out,
			"会话：%s，第 %d 次\n",
			result.SessionID,
			result.Turn,
		); err != nil {
			return fmt.Errorf("write session result: %w", err)
		}
	}

	return nil
}

// readChatStream只接收当前教学接口的单行JSON data；不是通用SSE解析器。
func readChatStream(body io.Reader, out io.Writer) (streamResult, error) {
	scanner := bufio.NewScanner(body)

	var eventType, eventData string
	var answer strings.Builder
	var result streamResult
	done := false

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if eventType == "" && eventData == "" {
				continue
			}
			if done {
				return streamResult{}, errors.New("received event after done")
			}

			switch eventType {
			case "delta":
				var delta streamDelta
				if err := json.Unmarshal([]byte(eventData), &delta); err != nil {
					return streamResult{}, fmt.Errorf("decode delta: %w", err)
				}
				if delta.Content == "" {
					return streamResult{}, errors.New("delta content is empty")
				}

				answer.WriteString(delta.Content)
				if _, err := fmt.Fprint(out, delta.Content); err != nil {
					return streamResult{}, fmt.Errorf("write delta: %w", err)
				}

			case "done":
				if err := json.Unmarshal([]byte(eventData), &result); err != nil {
					return streamResult{}, fmt.Errorf("decode done: %w", err)
				}
				if result.Answer == "" || result.Answer != answer.String() {
					return streamResult{}, errors.New("done answer does not match deltas")
				}
				done = true

			default:
				return streamResult{}, fmt.Errorf("unexpected event: %q", eventType)
			}

			eventType, eventData = "", ""
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			eventData = strings.TrimPrefix(line, "data: ")
		} else {
			return streamResult{}, fmt.Errorf("unexpected SSE line: %q", line)
		}
	}

	// Scanner遇到EOF返回false，Err为nil；传输错误则Err非nil。
	if err := scanner.Err(); err != nil {
		return streamResult{}, fmt.Errorf("read SSE: %w", err)
	}
	if eventType != "" || eventData != "" {
		return streamResult{}, errors.New("SSE ended before event delimiter")
	}
	if !done {
		return streamResult{}, errors.New("SSE ended without done")
	}

	return result, nil
}
