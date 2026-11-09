package main

import (
	"io"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// 不使用t.Parallel：本夹具短暂捕获进程stdout；所有任务收尾后才恢复。
// 虚拟时钟验证等待关系，不用真实机器上的固定耗时上限断言并发。
func TestGoroutineLessonWaitsForBothTasks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		original := os.Stdout
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		defer writer.Close()
		os.Stdout = writer
		defer func() { os.Stdout = original }()

		started := time.Now()
		main()
		elapsed := time.Since(started)
		// 漏Wait的反证也先让两个固定任务收尾，再恢复stdout；不泄漏goroutine。
		time.Sleep(3 * time.Second)
		os.Stdout = original
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		output := string(data)
		if elapsed != 2*time.Second {
			t.Fatalf("main elapsed=%v; want overlapping waits totaling 2s; output=%q", elapsed, output)
		}
		for _, line := range []string{"任务A：开始\n", "任务A：完成\n", "任务B：开始\n", "任务B：完成\n", "main：已启动两个任务，等待完成\n"} {
			if strings.Count(output, line) != 1 {
				t.Fatalf("line %q must occur once; output=%q", line, output)
			}
		}
		for _, name := range []string{"任务A", "任务B"} {
			if strings.Index(output, name+"：开始") >= strings.Index(output, name+"：完成") {
				t.Fatalf("task completion preceded start: %q", output)
			}
		}
		summary := "main：全部完成，总耗时 2.00 秒\n"
		count := "共享计数：期望 4000，实际 4000\n"
		if !strings.HasSuffix(output, summary+count+channelDemoWant) || strings.Count(output, "\n") != 12 {
			t.Fatalf("previous demonstrations must precede the channel results: %q", output)
		}
	})
}

func TestSimulateTaskDuration(t *testing.T) {
	for _, delay := range []time.Duration{0, -time.Second, 250 * time.Millisecond} {
		t.Run(delay.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				original := os.Stdout
				reader, writer, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				defer writer.Close()
				os.Stdout = writer
				defer func() { os.Stdout = original }()

				started := time.Now()
				simulateTask("测试任务", delay)
				elapsed := time.Since(started)
				os.Stdout = original
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
				wantDelay := delay
				if wantDelay < 0 {
					wantDelay = 0
				}
				if elapsed != wantDelay || string(data) != "测试任务：开始\n测试任务：完成\n" {
					t.Fatalf("elapsed=%v want=%v output=%q", elapsed, wantDelay, data)
				}
			})
		})
	}
}
