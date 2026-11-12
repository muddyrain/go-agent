package main

import (
	"fmt"
	"sync"
	"time"
)

func simulateTask(name string, delay time.Duration) {
	fmt.Printf("%s：开始\n", name)
	// 固定等待仅用于观察并发，不是实际I/O，也不负责判断任务是否完成。
	time.Sleep(delay)
	fmt.Printf("%s：完成\n", name)
}

func runGoroutineDemo() {
	started := time.Now()
	var wg sync.WaitGroup

	// 两个固定任务，先登记再启动；不能把第一次Add放进新goroutine。
	wg.Add(2)
	go func() {
		defer wg.Done()
		simulateTask("任务A", time.Second)
	}()
	go func() {
		defer wg.Done()
		simulateTask("任务B", 2*time.Second)
	}()
	fmt.Println("main：已启动两个任务，等待完成")
	wg.Wait()
	fmt.Printf("main：全部完成，总耗时 %.2f 秒\n", time.Since(started).Seconds())
}

func main() {
	// 默认聚焦当前小节；历史实验保留，分别由测试验证。
	runTickerDemo()
}
