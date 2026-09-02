package agent

import (
	"context"
	"fmt"

	"agenthub/internal/llm"
	"agenthub/internal/memory"
	"agenthub/internal/tool"
)

// Agent 负责协调模型、记忆和工具系统，驱动完整的模型调用循环。
// 它只负责组件编排，不依赖具体模型厂商，也不包含具体工具的业务实现。
type Agent struct {
	model    llm.Model
	registry *tool.Registry
	executor *tool.Executor
	maxSteps int
	memory   memory.Memory
}

// RunResult 汇总一次 Agent 运行的完整结果。
// Messages 保留完整运行历史，Steps 记录模型调用次数，
// Usage 累加本次运行中每一轮模型请求的 Token 用量。
type RunResult struct {
	Messages     []llm.Message
	FinalMessage llm.Message
	Steps        int
	Usage        llm.Usage
}

// responseGenerator 抽象“一次模型响应”的获取方式。
// 同步调用可以直接传入 Model.Generate；流式调用则传入一个负责启动并消费 Stream 的闭包。
// 两种模式最终都返回完整 llm.Response，因此可以复用同一套 Agent Loop。
type responseGenerator func(
	ctx context.Context,
	request llm.Request,
) (llm.Response, error)

// New 校验并组装 Agent 运行所需的全部依赖。
// 构造阶段提前拒绝无效依赖，可以避免 Agent 运行到一半才因缺少组件而失败。
func New(
	model llm.Model,
	registry *tool.Registry,
	maxSteps int,
	mem memory.Memory,
) (*Agent, error) {
	if model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if registry == nil {
		return nil, fmt.Errorf("tool registry is required")
	}
	if maxSteps <= 0 {
		return nil, fmt.Errorf(
			"max steps must be greater than zero",
		)
	}
	if mem == nil {
		return nil, fmt.Errorf("memory is required")
	}
	// Registry 既向模型提供工具定义，也供 Executor 查找并执行本地工具。
	// 两者必须共享同一个 Registry，避免模型看到的工具与实际可执行工具不一致。
	executor, err := tool.NewExecutor(registry)
	if err != nil {
		return nil, fmt.Errorf(
			"create tool executor: %w",
			err,
		)
	}

	return &Agent{
		model:    model,
		registry: registry,
		executor: executor,
		maxSteps: maxSteps,
		memory:   mem,
	}, nil
}

// Run 是同步调用的薄入口。
// 这里传递的是 a.model.Generate 方法本身，而不是立即调用它；
// 共享循环会在每个模型步骤中使用该方法取得一次完整响应。
func (a *Agent) Run(
	ctx context.Context,
	messages []llm.Message,
) (RunResult, error) {
	return a.runWithGenerator(
		ctx,
		messages,
		a.model.Generate,
	)
}

// runWithGenerator 实现同步和流式调用共享的 Agent Loop。
// generate 只决定如何取得一次模型响应，其余 Memory、ToolCall、
// Usage、历史记录和最大步骤控制都在这里统一处理。
func (a *Agent) runWithGenerator(
	ctx context.Context,
	messages []llm.Message,
	generate responseGenerator,
) (RunResult, error) {
	if generate == nil {
		return RunResult{}, fmt.Errorf(
			"response generator is required",
		)
	}

	if len(messages) == 0 {
		return RunResult{}, fmt.Errorf(
			"messages are required",
		)
	}

	// 创建独立的外层切片保存完整运行历史，避免后续 append
	// 直接改变调用方传入的切片结构。
	// 这里只复制 Message 元素，不是对 Message 内部引用数据的完整深拷贝。
	history := make(
		[]llm.Message,
		len(messages),
	)
	copy(history, messages)

	// Usage 从零值开始，随后累加每一轮模型响应的 Token 用量。
	// step 表示模型调用次数；同一个模型步骤中可能执行零个、一个或多个工具。
	usage := llm.Usage{}
	for step := 1; step <= a.maxSteps; step++ {
		// history 始终保留完整运行记录，Memory 只生成当前一轮模型可见的消息视图。
		// 工具调用会持续追加新的消息，因此每次请求模型前都必须重新应用 Memory。
		croppedMessages, err := a.memory.Apply(history)
		if err != nil {
			return RunResult{}, fmt.Errorf("memory apply: %w", err)
		}

		// 每轮重新读取工具定义，使模型请求与 Registry 当前可用的工具保持一致。
		// 发给模型的只有工具定义，不包含本地 Tool 实现和执行逻辑。
		request := llm.Request{
			Messages: croppedMessages,
			Tools:    a.registry.Definitions(),
		}
		// generate 屏蔽同步与流式响应获取方式的差异。
		// 对共享循环来说，两种方式最终都会返回一个完整的 llm.Response。
		response, err := generate(
			ctx,
			request,
		)
		if err != nil {
			return RunResult{}, fmt.Errorf(
				"generate model response at step %d: %w",
				step,
				err,
			)
		}

		// 一次 Agent 运行可能多次请求模型，因此最终 Usage 必须按轮次累加，
		// 不能只保留最后一次模型响应的用量。
		usage.InputTokens += response.Usage.InputTokens
		usage.OutputTokens += response.Usage.OutputTokens
		usage.TotalTokens += response.Usage.TotalTokens

		// 模型只能返回 Assistant 消息：普通回答和工具调用都属于 Assistant 行为。
		// User 和 Tool 消息应分别由调用方和本地工具执行流程创建。
		if response.Message.Role != llm.RoleAssistant {
			return RunResult{}, fmt.Errorf(
				"model response role at step %d must be %q",
				step,
				llm.RoleAssistant,
			)
		}

		// 必须先记录包含 ToolCalls 的 Assistant 消息，再追加对应的 ToolMessage，
		// 这样下一轮模型才能正确理解每个工具结果是由哪次调用产生的。
		history = append(
			history,
			response.Message,
		)

		// 没有 ToolCalls 表示模型已经给出最终回答，Agent Loop 可以正常结束。
		// 继续请求模型只会额外消耗资源，并可能改变已经完成的答案。
		if len(response.Message.ToolCalls) == 0 {
			return RunResult{
				Messages:     history,
				FinalMessage: response.Message,
				Steps:        step,
				Usage:        usage,
			}, nil
		}

		// 最后一个允许的模型步骤仍然请求工具时，不再执行这些工具。
		// 因为执行后的结果已经没有下一次模型调用可以处理，
		// 继续执行还可能产生数据库写入、消息发送等无意义副作用。
		if step == a.maxSteps {
			return RunResult{}, fmt.Errorf(
				"agent exceeded maximum steps: %d",
				a.maxSteps,
			)
		}

		// 同一轮模型可能返回多个 ToolCall。
		// 当前按模型返回顺序串行执行，以保持 ToolMessage 顺序稳定，
		// 并避免并发工具之间出现未定义的数据依赖。
		for _, call := range response.Message.ToolCalls {
			toolResult, err := a.executor.Execute(
				ctx,
				call,
			)

			// Executor 会把参数 Schema 错误和普通工具业务错误转换为 Tool Result，
			// 交给下一轮模型理解和修正。
			// 这里收到的 error 表示执行流程本身无法继续，应立即向上传播。
			if err != nil {
				return RunResult{}, fmt.Errorf(
					"execute tool call %q at step %d: %w",
					call.ID,
					step,
					err,
				)
			}
			// ToolMessage 使用 CallID 将执行结果关联回模型发出的具体 ToolCall。
			// Name 标识工具类型，CallID 标识本轮中的某一次具体调用。
			toolMessage := llm.ToolMessage(
				toolResult.Name,
				toolResult.CallID,
				toolResult.Content,
			)

			// 工具结果进入完整 history 后，下一轮 Memory.Apply 会基于最新历史
			// 重新生成模型视图，使模型能够根据工具结果继续推理或给出最终回答。
			history = append(
				history,
				toolMessage,
			)
		}

	}

	return RunResult{}, fmt.Errorf(
		"agent exceeded maximum steps: %d",
		a.maxSteps,
	)
}

// RunStream 是流式调用入口。
// 流式模型每轮产生多个事件，但 ConsumeStream 会将一轮流最终收敛为完整 Response，
// 再交给 runWithGenerator 复用同步模式下的 Agent Loop。
func (a *Agent) RunStream(
	ctx context.Context,
	messages []llm.Message,
	handler llm.StreamHandler,
) (RunResult, error) {

	if handler == nil {
		return RunResult{}, fmt.Errorf(
			"stream handler is required",
		)
	}

	// Agent 只要求基础模型满足 llm.Model；流式能力是可选扩展。
	// 运行时通过类型断言判断当前模型的具体实现是否还满足 StreamingModel。
	streamingModel, ok := a.model.(llm.StreamingModel)

	if !ok {
		return RunResult{}, fmt.Errorf(
			"model does not support streaming",
		)
	}

	// 将“启动并消费一次模型流”包装成 responseGenerator，
	// 使共享循环不需要知道 Response 来自同步调用还是流式事件。
	return a.runWithGenerator(
		ctx,
		messages,
		func(
			ctx context.Context,
			request llm.Request,
		) (llm.Response, error) {
			stream, err := streamingModel.Stream(
				ctx,
				request,
			)
			if err != nil {
				return llm.Response{}, fmt.Errorf(
					"start model stream: %w",
					err,
				)
			}
			// ConsumeStream 负责逐个转发 Delta/Done 事件、关闭 Stream，
			// 并从 Done 事件中取得本轮完整响应。
			// Done 只代表当前模型流结束，不一定代表整个 Agent Loop 结束。
			response, err := llm.ConsumeStream(
				stream,
				handler,
			)

			if err != nil {
				return llm.Response{}, fmt.Errorf(
					"consume model stream: %w",
					err,
				)
			}

			return response, nil
		},
	)
}
