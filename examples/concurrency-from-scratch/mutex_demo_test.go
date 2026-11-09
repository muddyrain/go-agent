package main

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

// 不使用t.Parallel：只在所有调用及其任务收尾后恢复进程stdout。
func captureMutexDemoOutput(t *testing.T, run func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	os.Stdout = writer
	defer func() { os.Stdout = original }()

	run()
	os.Stdout = original
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestMutexDemoCount(t *testing.T) {
	const runs = 10
	output := captureMutexDemoOutput(t, func() {
		for i := 0; i < runs; i++ {
			runMutexDemo()
		}
	})
	want := strings.Repeat("共享计数：期望 4000，实际 4000\n", runs)
	if output != want {
		t.Fatalf("shared count must be complete and start fresh on each call: output=%q", output)
	}
}

func TestMutexDemoIndependentConcurrentCalls(t *testing.T) {
	const runs = 8
	output := captureMutexDemoOutput(t, func() {
		var wg sync.WaitGroup
		wg.Add(runs)
		for i := 0; i < runs; i++ {
			go func() {
				defer wg.Done()
				runMutexDemo()
			}()
		}
		wg.Wait()
	})
	// 各调用的total和Mutex是局部的；不依赖调用结束顺序。
	want := strings.Repeat("共享计数：期望 4000，实际 4000\n", runs)
	if output != want {
		t.Fatalf("independent calls must each produce one complete count: output=%q", output)
	}
}
