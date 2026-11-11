package main

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

const selectDemoWant = "select：收到结果 42\nselect：等待超时，后台任务已退出\n"

func TestSelectDemoOutput(t *testing.T) {
	// 使用虚拟时间验证本节两次调用，不依赖真实机器的调度耗时上限。
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		output := captureMutexDemoOutput(t, runSelectDemo)
		if output != selectDemoWant || time.Since(start) != 200*time.Millisecond {
			t.Fatalf("output=%q elapsed=%v", output, time.Since(start))
		}
	})
}

func TestWaitForNumberSuccess(t *testing.T) {
	for _, delay := range []time.Duration{0, -time.Second, 100 * time.Millisecond} {
		t.Run(delay.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				value, err := waitForNumber(context.Background(), delay)
				wantDelay := delay
				if wantDelay < 0 {
					wantDelay = 0
				}
				if err != nil || value != 42 || time.Since(start) != wantDelay {
					t.Fatalf("value=%d error=%v elapsed=%v want=%v", value, err, time.Since(start), wantDelay)
				}
			})
		})
	}
}

func TestWaitForNumberDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		start := time.Now()
		value, err := waitForNumber(ctx, time.Second)
		if value != 0 || !errors.Is(err, context.DeadlineExceeded) || time.Since(start) != 100*time.Millisecond {
			t.Fatalf("value=%d error=%v elapsed=%v", value, err, time.Since(start))
		}
		if err == context.DeadlineExceeded || !strings.HasPrefix(err.Error(), "wait for number: ") {
			t.Fatalf("expected a wrapped stage error: %v", err)
		}
	})
}

func TestWaitForNumberManualCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var canceler sync.WaitGroup
		canceler.Add(1)
		go func() {
			defer canceler.Done()
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		start := time.Now()
		value, err := waitForNumber(ctx, time.Second)
		canceler.Wait()
		if value != 0 || !errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || time.Since(start) != 50*time.Millisecond {
			t.Fatalf("value=%d error=%v elapsed=%v", value, err, time.Since(start))
		}
	})
}

func TestWaitForNumberAlreadyCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		start := time.Now()
		value, err := waitForNumber(ctx, 0)
		if value != 0 || !errors.Is(err, context.Canceled) || time.Since(start) != 0 {
			t.Fatalf("value=%d error=%v elapsed=%v", value, err, time.Since(start))
		}
	})
}

func TestWaitForNumberDoesNotCancelParentOnSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		value, err := waitForNumber(ctx, time.Millisecond)
		if err != nil || value != 42 || ctx.Err() != nil {
			t.Fatalf("value=%d error=%v parent=%v", value, err, ctx.Err())
		}
	})
}

func TestWaitForNumberIndependentCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(8)
		for i := 0; i < 8; i++ {
			// 每个调用拥有自己的ctx和channel；取消一份不影响其他调用。
			go func(index int) {
				defer wg.Done()
				if index%2 == 0 {
					value, err := waitForNumber(context.Background(), 10*time.Millisecond)
					if err != nil || value != 42 {
						t.Errorf("success invocation: value=%d error=%v", value, err)
					}
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
				defer cancel()
				value, err := waitForNumber(ctx, time.Second)
				if value != 0 || !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("canceled invocation: value=%d error=%v", value, err)
				}
			}(i)
		}
		wg.Wait()
	})
}
