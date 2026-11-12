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

// 精确时间只在虚拟时钟内断言；真实入口的时间受运行环境调度影响。
const tickerDemoWant = "ticker：快速处理（周期100ms，工作20ms）\n" +
	"ticker：开始第 1 次处理，通知=100ms，开始=100ms\n" +
	"ticker：开始第 2 次处理，通知=200ms，开始=200ms\n" +
	"ticker：开始第 3 次处理，通知=300ms，开始=300ms\n" +
	"ticker：开始第 4 次处理，通知=400ms，开始=400ms\n" +
	"ticker：开始第 5 次处理，通知=500ms，开始=500ms\n" +
	"ticker：开始第 6 次处理，通知=600ms，开始=600ms\n" +
	"ticker：超时停止，已完成 6 次\n" +
	"ticker：慢速处理（周期100ms，工作250ms）\n" +
	"ticker：开始第 1 次处理，通知=100ms，开始=100ms\n" +
	"ticker：开始第 2 次处理，通知=200ms，开始=350ms\n" +
	"ticker：开始第 3 次处理，通知=400ms，开始=600ms\n" +
	"ticker：超时停止，已完成 2 次\n"

func TestTickerDemoOutput(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		output := captureMutexDemoOutput(t, runTickerDemo)
		if output != tickerDemoWant || time.Since(start) != 1300*time.Millisecond {
			t.Fatalf("output=%q elapsed=%v", output, time.Since(start))
		}
	})
}

func TestMainRunsOnlyCurrentTickerLesson(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		output := captureMutexDemoOutput(t, main)
		if output != tickerDemoWant || time.Since(start) != 1300*time.Millisecond {
			t.Fatalf("default entry must only run the current Ticker lesson: elapsed=%v output=%q", time.Since(start), output)
		}
	})
}

func TestTickerTaskWorkCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()
		start := time.Now()
		output := captureMutexDemoOutput(t, func() {
			completed, err := runTickerTask(ctx, 100*time.Millisecond, time.Second)
			if completed != 0 || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("completed=%d error=%v", completed, err)
			}
			if err == context.DeadlineExceeded || !strings.HasPrefix(err.Error(), "run ticker task: ") {
				t.Fatalf("expected a wrapped stage error: %v", err)
			}
		})
		if time.Since(start) != 150*time.Millisecond || strings.Count(output, "开始第") != 1 {
			t.Fatalf("elapsed=%v output=%q", time.Since(start), output)
		}
	})
}

func TestTickerTaskCancellationBeforeFirstTick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		start := time.Now()
		output := captureMutexDemoOutput(t, func() {
			completed, err := runTickerTask(ctx, time.Second, 0)
			if completed != 0 || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("completed=%d error=%v", completed, err)
			}
		})
		if output != "" || time.Since(start) != 10*time.Millisecond {
			t.Fatalf("elapsed=%v output=%q", time.Since(start), output)
		}
	})
}

func TestTickerTaskManualCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var canceler sync.WaitGroup
		canceler.Add(1)
		go func() {
			defer canceler.Done()
			time.Sleep(150 * time.Millisecond)
			cancel()
		}()
		start := time.Now()
		output := captureMutexDemoOutput(t, func() {
			completed, err := runTickerTask(ctx, 100*time.Millisecond, time.Second)
			canceler.Wait()
			if completed != 0 || !errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("completed=%d error=%v", completed, err)
			}
		})
		if time.Since(start) != 150*time.Millisecond || strings.Count(output, "开始第") != 1 {
			t.Fatalf("elapsed=%v output=%q", time.Since(start), output)
		}
	})
}

func TestTickerTaskInputBoundaries(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		t.Run(interval.String(), func(t *testing.T) {
			completed, err := runTickerTask(context.Background(), interval, 0)
			if completed != 0 || err == nil || err.Error() != "ticker interval must be positive" {
				t.Fatalf("completed=%d error=%v", completed, err)
			}
		})
	}
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		start := time.Now()
		output := captureMutexDemoOutput(t, func() {
			// 已取消输入优先于非法周期，不启动周期任务。
			completed, err := runTickerTask(ctx, 0, 0)
			if completed != 0 || !errors.Is(err, context.Canceled) {
				t.Fatalf("completed=%d error=%v", completed, err)
			}
		})
		if output != "" || time.Since(start) != 0 {
			t.Fatalf("elapsed=%v output=%q", time.Since(start), output)
		}
	})
}

func TestTickerTaskZeroWork(t *testing.T) {
	for _, workDelay := range []time.Duration{0, -time.Second} {
		t.Run(workDelay.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
				defer cancel()
				start := time.Now()
				output := captureMutexDemoOutput(t, func() {
					completed, err := runTickerTask(ctx, 100*time.Millisecond, workDelay)
					if completed != 3 || !errors.Is(err, context.DeadlineExceeded) {
						t.Fatalf("completed=%d error=%v", completed, err)
					}
				})
				if time.Since(start) != 350*time.Millisecond || strings.Count(output, "开始第") != 3 {
					t.Fatalf("elapsed=%v output=%q", time.Since(start), output)
				}
			})
		})
	}
}

func TestTickerTaskIndependentCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// 捕获stdout的测试不t.Parallel；仅本夹具内并发调用，再等全部收尾。
		output := captureMutexDemoOutput(t, func() {
			var wg sync.WaitGroup
			wg.Add(8)
			for i := 0; i < 8; i++ {
				go func(index int) {
					defer wg.Done()
					deadline := 350 * time.Millisecond
					want := 3
					if index%2 == 1 {
						deadline = 50 * time.Millisecond
						want = 0
					}
					ctx, cancel := context.WithTimeout(context.Background(), deadline)
					defer cancel()
					completed, err := runTickerTask(ctx, 100*time.Millisecond, 0)
					if completed != want || !errors.Is(err, context.DeadlineExceeded) {
						t.Errorf("invocation %d: completed=%d want=%d error=%v", index, completed, want, err)
					}
				}(i)
			}
			wg.Wait()
		})
		if strings.Count(output, "开始第") != 12 {
			t.Fatalf("independent calls must not share progress: %q", output)
		}
	})
}

func TestStoppedTickerDoesNotCloseChannel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticker := time.NewTicker(time.Second)
		ticker.Stop()
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		select {
		case _, ok := <-ticker.C:
			t.Fatalf("stopped ticker must not deliver a tick or close: ok=%v", ok)
		case <-timer.C:
		}
	})
}
