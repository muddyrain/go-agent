package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func runTickerTask(ctx context.Context, interval, workDelay time.Duration) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("run ticker task: %w", err)
	}
	if interval <= 0 {
		return 0, errors.New("ticker interval must be positive")
	}

	started := time.Now()
	ticker := time.NewTicker(interval)

	defer ticker.Stop()
	completed := 0
	for {
		select {
		case <-ctx.Done():
			return completed, fmt.Errorf("run ticker task: %w", ctx.Err())
		case tick := <-ticker.C:
			// 取消与tick可能同时就绪；开始下一次处理前再检查已知取消。
			if err := ctx.Err(); err != nil {
				return completed, fmt.Errorf("run ticker task: %w", err)
			}
			fmt.Printf("ticker：开始第 %d 次处理，通知=%dms，开始=%dms\n",
				completed+1, tick.Sub(started).Milliseconds(), time.Since(started).Milliseconds())

			// 只模拟本次工作的耗时；仍在当前调用内顺序处理，不另开goroutine。
			timer := time.NewTimer(workDelay)

			select {
			case <-ctx.Done():
				timer.Stop()
				return completed, fmt.Errorf("run ticker task: %w", ctx.Err())
			case <-timer.C:
				timer.Stop()
			}
			if err := ctx.Err(); err != nil {
				return completed, fmt.Errorf("run ticker task: %w", err)
			}
			// 只计已完成工作；取消中断的当前这一份不计入。
			completed++
		}
	}
}

func runTickerDemo() {
	fmt.Println("ticker：快速处理（周期100ms，工作20ms）")
	ctx, cancel := context.WithTimeout(context.Background(), 650*time.Millisecond)
	completed, err := runTickerTask(ctx, 100*time.Millisecond, 20*time.Millisecond)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		fmt.Println("ticker：执行结果异常：", err)
		return
	}
	fmt.Printf("ticker：超时停止，已完成 %d 次\n", completed)

	fmt.Println("ticker：慢速处理（周期100ms，工作250ms）")
	ctx, cancel = context.WithTimeout(context.Background(), 650*time.Millisecond)
	completed, err = runTickerTask(ctx, 100*time.Millisecond, 250*time.Millisecond)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		fmt.Println("ticker：执行结果异常：", err)
		return
	}
	fmt.Printf("ticker：超时停止，已完成 %d 次\n", completed)
}
