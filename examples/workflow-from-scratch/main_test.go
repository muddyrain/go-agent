package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// 使用测试进程执行真正的main，允许验证os.Exit而不退出父测试进程。
func TestWorkflowCLIHelperProcess(t *testing.T) {
	if os.Getenv("AGENTHUB_TEST_WORKFLOW_CLI") != "1" {
		return
	}
	for index, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"workflow-cli-test"}, os.Args[index+1:]...)
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			main()
			os.Exit(0)
		}
	}
	os.Exit(2)
}

func TestWorkflowCLIArgumentsAndCancellation(t *testing.T) {
	normal := "任务：Go\n方案类型：规则模板（非模型生成）\n1. 明确输入与输出\n2. 实现最小逻辑\n3. 验证成功与错误路径\n"
	for _, tt := range []struct {
		name        string
		args        []string
		exit        int
		stdout      string
		stderr      string
		stderrExact bool
	}{
		{"normal", []string{"-task", "  Go  "}, 0, normal, "", true},
		{"cancel_false", []string{"-task", "Go", "-cancel-demo=false"}, 0, normal, "", true},
		{"cancel", []string{"-cancel-demo"}, 1, "", "任务已取消\n", true},
		{"cancel_true", []string{"-cancel-demo=true"}, 1, "", "任务已取消\n", true},
		{"cancel_before_task_validation", []string{"-cancel-demo", "-task", " "}, 1, "", "任务已取消\n", true},
		{"blank_task", []string{"-task", " "}, 1, "", "task is required", false},
		{"conflicting_modes", []string{"-serve", "-cancel-demo"}, 1, "", "参数错误：-cancel-demo 仅用于 CLI，不能与 -serve 同时使用\n", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			args := append([]string{"-test.run=^TestWorkflowCLIHelperProcess$", "--"}, tt.args...)
			command := exec.CommandContext(ctx, os.Args[0], args...)
			// 避免重复环境键；只对这一子进程启用教学入口。
			for _, value := range os.Environ() {
				if !strings.HasPrefix(value, "AGENTHUB_TEST_WORKFLOW_CLI=") {
					command.Env = append(command.Env, value)
				}
			}
			command.Env = append(command.Env, "AGENTHUB_TEST_WORKFLOW_CLI=1")
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			err := command.Run()
			if ctx.Err() != nil {
				t.Fatalf("CLI did not terminate: %v; stdout=%q stderr=%q", ctx.Err(), stdout.String(), stderr.String())
			}
			exit := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatal(err)
				}
				exit = exitErr.ExitCode()
			}
			if exit != tt.exit || stdout.String() != tt.stdout {
				t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
			}
			if tt.stderrExact {
				if stderr.String() != tt.stderr {
					t.Fatalf("stderr=%q want=%q", stderr.String(), tt.stderr)
				}
			} else if !strings.HasPrefix(stderr.String(), "执行失败： ") || !strings.Contains(stderr.String(), tt.stderr) {
				t.Fatalf("stderr=%q", stderr.String())
			}
		})
	}
}
