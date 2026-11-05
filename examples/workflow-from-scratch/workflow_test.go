package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/compose"
)

func TestAnalyzeTask(t *testing.T) {
	for _, tt := range []struct {
		name, task, want string
		wantError        bool
	}{
		{"normal", "给聊天接口补一个测试", "给聊天接口补一个测试", false},
		{"trim", " \t Go \n", "Go", false},
		{"inner_space", "test  /chat", "test  /chat", false},
		{"empty", "", "", true},
		{"blank", " \t\n ", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			analysis, err := analyzeTask(context.Background(), proposalRequest{Task: tt.task})
			if tt.wantError {
				if err == nil || err.Error() != "task is required" || analysis.Task != "" || analysis.Steps != nil {
					t.Fatalf("analysis=%v error=%v", analysis, err)
				}
				return
			}
			if err != nil || analysis.Task != tt.want || !reflect.DeepEqual(analysis.Steps, []string{"明确输入与输出", "实现最小逻辑", "验证成功与错误路径"}) {
				t.Fatalf("analysis=%v error=%v", analysis, err)
			}
		})
	}
}

func TestBuildProposalUsesAnalysis(t *testing.T) {
	// 直接传另一组步骤，确认第二节点使用上游数据，而不是重新生成固定步骤。
	analysis := taskAnalysis{Task: "Go %s", Steps: []string{"先验证", "再改写"}}
	result, err := buildProposal(context.Background(), analysis)
	want := "任务：Go %s\n方案类型：规则模板（非模型生成）\n1. 先验证\n2. 再改写\n"
	if err != nil || result.Content != want {
		t.Fatalf("result=%q error=%v", result.Content, err)
	}
}

func TestProposalWorkflow(t *testing.T) {
	workflow, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// 同一个Runnable可以执行不同任务，每次结果独立，不累积上次Content。
	for _, task := range []string{"给聊天接口补一个测试", "  test /chat  ", "A\nB"} {
		result, err := workflow.Invoke(context.Background(), proposalRequest{Task: task})
		want := "任务：" + strings.TrimSpace(task) + "\n方案类型：规则模板（非模型生成）\n1. 明确输入与输出\n2. 实现最小逻辑\n3. 验证成功与错误路径\n"
		if err != nil || result.Content != want {
			t.Fatalf("task=%q result=%q error=%v", task, result.Content, err)
		}
	}
	result, err := workflow.Invoke(context.Background(), proposalRequest{Task: " \t "})
	if err == nil || !strings.Contains(err.Error(), "task is required") || result != (proposalResult{}) {
		t.Fatalf("empty task result=%v error=%v", result, err)
	}
}

func TestProposalCancellation(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer stop()
	workflow, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		ctx  context.Context
		want error
	}{
		{"canceled", canceled, context.Canceled},
		{"deadline", expired, context.DeadlineExceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			analysis, err := analyzeTask(tt.ctx, proposalRequest{Task: "Go"})
			if !errors.Is(err, tt.want) || !strings.HasPrefix(err.Error(), "analyze task: ") || analysis.Task != "" {
				t.Fatalf("analyze result=%v error=%v", analysis, err)
			}
			result, err := buildProposal(tt.ctx, taskAnalysis{Task: "Go"})
			if !errors.Is(err, tt.want) || !strings.HasPrefix(err.Error(), "build proposal: ") || result.Content != "" {
				t.Fatalf("build result=%v error=%v", result, err)
			}
			result, err = workflow.Invoke(tt.ctx, proposalRequest{Task: "Go"})
			if !errors.Is(err, tt.want) || result.Content != "" {
				t.Fatalf("workflow result=%v error=%v", result, err)
			}
		})
	}
}

func TestCompileAndInvokePhases(t *testing.T) {
	// 测试包装函数只记录调用顺序，不改动正式节点实现。
	var order []string
	wf := compose.NewWorkflow[proposalRequest, proposalResult]()
	wf.AddLambdaNode("analyze", compose.InvokableLambda(func(ctx context.Context, input proposalRequest) (taskAnalysis, error) {
		order = append(order, "analyze")
		return analyzeTask(ctx, input)
	})).AddInput(compose.START)
	wf.AddLambdaNode("build", compose.InvokableLambda(func(ctx context.Context, input taskAnalysis) (proposalResult, error) {
		order = append(order, "build")
		return buildProposal(ctx, input)
	})).AddInput("analyze")
	wf.End().AddInput("build")
	runnable, err := wf.Compile(context.Background())
	if err != nil || len(order) != 0 {
		t.Fatalf("compile error=%v executed=%v", err, order)
	}
	if _, err := runnable.Invoke(context.Background(), proposalRequest{Task: "Go"}); err != nil || !reflect.DeepEqual(order, []string{"analyze", "build"}) {
		t.Fatalf("invoke error=%v order=%v", err, order)
	}
	order = nil
	if _, err := runnable.Invoke(context.Background(), proposalRequest{}); err == nil || !reflect.DeepEqual(order, []string{"analyze"}) {
		t.Fatalf("continued after input error: error=%v order=%v", err, order)
	}
}

func TestWorkflowRejectsTypeMismatch(t *testing.T) {
	wf := compose.NewWorkflow[proposalRequest, proposalResult]()
	// START输出proposalRequest，而buildProposal需要taskAnalysis；错误应在Compile发现。
	wf.AddLambdaNode("build", compose.InvokableLambda(buildProposal)).AddInput(compose.START)
	wf.End().AddInput("build")
	if _, err := wf.Compile(context.Background()); err == nil {
		t.Fatal("proposalRequest incorrectly accepted as taskAnalysis")
	}
}
