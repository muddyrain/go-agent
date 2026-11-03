package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/network"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
)

// 只记录应用事件字节和调用次数，不模拟网络分块编码；真实HTTP另行验证。
type eventTestWriter struct {
	bytes.Buffer
	writes, flushes, finalizes int
	writeErr, flushErr         error
	onFlush                    func()
}

func (w *eventTestWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	return w.Buffer.Write(p)
}

func (w *eventTestWriter) Flush() error {
	w.flushes++
	if w.onFlush != nil {
		w.onFlush()
	}
	return w.flushErr
}

func (w *eventTestWriter) Finalize() error {
	w.finalizes++
	return nil
}

func eventTestContext(w *eventTestWriter) *app.RequestContext {
	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/events")
	c.Response.HijackWriter(w)
	return c
}

func TestEventsHandlerFormatAndFlush(t *testing.T) {
	w := &eventTestWriter{}
	c := eventTestContext(w)
	eventsHandler(context.Background(), c)
	want := "event: message\ndata: 第1条消息\n\n" +
		"event: message\ndata: 第2条消息\n\n" +
		"event: message\ndata: 第3条消息\n\n" +
		"event: done\ndata: 发送完成\n\n"
	if w.String() != want || w.writes != 4 || w.flushes != 4 {
		t.Fatalf("events=%q writes=%d flushes=%d", w.String(), w.writes, w.flushes)
	}
	if w.finalizes != 0 {
		t.Fatal("Handler must leave response finalization to Hertz")
	}
	if string(c.Response.Header.Peek("Content-Type")) != "text/event-stream; charset=utf-8" || string(c.Response.Header.Peek("Cache-Control")) != "no-cache" {
		t.Fatal("missing SSE headers")
	}
}

func TestEventsHandlerCancellation(t *testing.T) {
	for _, alreadyCanceled := range []bool{true, false} {
		output := captureHertzRequestLog(t)
		ctx, cancel := context.WithCancel(context.Background())
		w := &eventTestWriter{onFlush: cancel}
		if alreadyCanceled {
			cancel()
		}
		eventsHandler(ctx, eventTestContext(w))
		cancel()
		wantWrites := 1
		if alreadyCanceled {
			wantWrites = 0
		}
		if w.writes != wantWrites || w.flushes != wantWrites || !strings.Contains(output.String(), "SSE request canceled: context canceled") {
			t.Fatalf("alreadyCanceled=%v writes=%d flushes=%d log=%q", alreadyCanceled, w.writes, w.flushes, output.String())
		}
	}
}

func TestEventsHandlerWriteAndFlushErrors(t *testing.T) {
	for _, where := range []string{"write", "flush"} {
		t.Run(where, func(t *testing.T) {
			output := captureHertzRequestLog(t)
			w := &eventTestWriter{}
			failure := errors.New("test " + where + " failed")
			if where == "write" {
				w.writeErr = failure
			} else {
				w.flushErr = failure
			}
			eventsHandler(context.Background(), eventTestContext(w))
			if w.writes != 1 || !strings.Contains(output.String(), "write SSE event: "+failure.Error()) {
				t.Fatal("Handler must stop after a failed event")
			}
			if where == "write" && w.flushes != 0 || where == "flush" && w.flushes != 1 {
				t.Fatalf("unexpected flushes=%d", w.flushes)
			}
			if strings.Contains(w.String(), "internal server error") || strings.Contains(w.String(), "done") {
				t.Fatal("failed stream must not append ordinary error or done")
			}
		})
	}
}

type streamTestConn struct {
	network.Conn // 本测试只调用Close，其他网络方法不使用。
	closed       int
	closeErr     error
}

func (c *streamTestConn) Close() error {
	c.closed++
	return c.closeErr
}

func TestRecoveryMiddlewareAbortsStreamingMode(t *testing.T) {
	for _, sendFirst := range []bool{false, true} {
		output := captureHertzRequestLog(t)
		w := &eventTestWriter{}
		c := eventTestContext(w)
		conn := &streamTestConn{}
		c.SetConn(conn)
		pending := 0
		c.SetHandlers(app.HandlersChain{recoveryMiddleware, func(_ context.Context, c *app.RequestContext) {
			writer := sse.NewWriter(c)
			if sendFirst {
				if err := writer.WriteEvent("", "message", []byte("partial")); err != nil {
					t.Fatal(err)
				}
			}
			panic("stream panic")
		}, func(_ context.Context, _ *app.RequestContext) { pending++ }})
		c.SetIndex(-1)
		c.Next(context.Background())
		if conn.closed != 1 || !c.IsAborted() || pending != 0 || c.Response.GetHijackWriter() != w {
			t.Fatal("stream recovery must close current connection, abort chain, and preserve writer")
		}
		if c.Response.StatusCode() != 200 || strings.Contains(w.String(), "internal server error") || !strings.Contains(output.String(), "value=stream panic") {
			t.Fatal("stream panic must not become a second 500 response")
		}
	}
}

func TestRecoveryStreamingCloseFailureLogged(t *testing.T) {
	output := captureHertzRequestLog(t)
	c := eventTestContext(&eventTestWriter{})
	c.SetConn(&streamTestConn{closeErr: errors.New("close failure")})
	c.SetHandlers(app.HandlersChain{recoveryMiddleware, func(_ context.Context, _ *app.RequestContext) { panic("stream panic") }})
	c.SetIndex(-1)
	c.Next(context.Background())
	if !strings.Contains(output.String(), "close SSE connection after panic: close failure") {
		t.Fatal("failed Close must be logged")
	}
}

func TestEventsRouteMethodAndDisconnectionOption(t *testing.T) {
	h := newServer()
	if !h.GetOptions().SenseClientDisconnection {
		t.Fatal("current netpoll example requires client disconnection sensing")
	}
	r := ut.PerformRequest(h.Engine, "POST", "/events", nil)
	if r.Code != 405 || r.Body.String() != "Method Not Allowed" {
		t.Fatal("POST must not enter the streaming Handler")
	}
}
