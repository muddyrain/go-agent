package main

import (
	"strings"
	"sync"
	"testing"
)

const channelDemoWant = "channel：收到第 1 份结果 1000\nchannel：收到第 2 份结果 1000\nchannel：收到第 3 份结果 1000\nchannel：收到第 4 份结果 1000\nchannel汇总：期望 4000，实际 4000\n"

func TestChannelDemoResults(t *testing.T) {
	const runs = 10
	// 基础版本发送是最后一步，发送方不在交付后写stdout或共享状态。
	output := captureMutexDemoOutput(t, func() {
		for i := 0; i < runs; i++ {
			runChannelDemo()
		}
	})
	if output != strings.Repeat(channelDemoWant, runs) {
		t.Fatalf("all four results must be received once before summary: %q", output)
	}
}

func TestChannelDemoIndependentConcurrentCalls(t *testing.T) {
	const runs = 8
	output := captureMutexDemoOutput(t, func() {
		var wg sync.WaitGroup
		wg.Add(runs)
		for i := 0; i < runs; i++ {
			go func() {
				defer wg.Done()
				runChannelDemo()
			}()
		}
		wg.Wait()
	})
	// 各调用使用自己的通道与total；只核对结果，不依赖调用间的日志顺序。
	for _, line := range strings.Split(strings.TrimSuffix(channelDemoWant, "\n"), "\n") {
		if strings.Count(output, line+"\n") != runs {
			t.Fatalf("each invocation must receive its own four values: line=%q output=%q", line, output)
		}
	}
	if strings.Count(output, "\n") != runs*5 {
		t.Fatalf("unexpected output: %q", output)
	}
}
