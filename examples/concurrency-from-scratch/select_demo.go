package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

func waitForNumber(ctx context.Context, delay time.Duration) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("wait for number: %w", err)
	}
	results := make(chan int)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.NewTimer(delay)
		defer timer.Stop()
		// 模拟工作等待也响应取消，不用不可取消的Sleep拖延收尾。
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		// 调用方可能已取消、不再接收；发送本身也需要退出路径。
		select {
		case <-ctx.Done():
			return
		case results <- 42:
		}
	}()
	// 返回前确认任务和Timer收尾；通知取消本身不等于任务已退出。
	defer wg.Wait()
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("wait for number: %w", ctx.Err())
	case value := <-results:
		return value, nil
	}
}

func runSelectDemo() {
	value, err := waitForNumber(context.Background(), 100*time.Millisecond)
	if err != nil {
		fmt.Println("select：执行失败：", err)
		return
	}
	fmt.Printf("select：收到结果 %d\n", value)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	value, err = waitForNumber(ctx, time.Second)
	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Println("select：等待超时，后台任务已退出")
		return
	}
	if err != nil {
		fmt.Println("select：执行失败：", err)
		return
	}
	fmt.Printf("select：慢任务收到结果 %d\n", value)
}
