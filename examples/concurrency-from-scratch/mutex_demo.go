package main

import (
	"fmt"
	"sync"
)

func runMutexDemo() {
	const workers = 4
	const increments = 1000

	var total int
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer wg.Done()
			for i := 0; i < increments; i++ {
				// 所有任务共用同一把锁，保护完整的读取、加一、写回。
				mu.Lock()
				total++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	// 所有写入都已结束，此处读取total不再与写入并发。
	fmt.Printf("共享计数：期望 %d，实际 %d\n", workers*increments, total)
}
