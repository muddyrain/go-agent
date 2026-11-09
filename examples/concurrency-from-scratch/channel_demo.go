package main

import "fmt"

func runChannelDemo() {
	const workers = 4
	const increments = 1000

	results := make(chan int)
	for worker := 0; worker < workers; worker++ {
		go func() {
			// 每个任务只修改自己的计数，不共同写调用方的total。
			var count int
			for i := 0; i < increments; i++ {
				count++
			}
			// 无缓冲发送等待接收方；结果交付是本任务的最后一步。
			results <- count
		}()
	}
	var total int
	for received := 0; received < workers; received++ {
		value := <-results
		// 只有当前接收方修改total，不需要Mutex保护共同写入。
		total += value
		fmt.Printf("channel：收到第 %d 份结果 %d\n", received+1, value)
	}
	fmt.Printf("channel汇总：期望 %d，实际 %d\n", workers*increments, total)
}
