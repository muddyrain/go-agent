# AgentHub 历史学习记录

本文件归档路线校准前已经完成的课程验收记录。它回答“过去实现并验证了什么”；当前教学顺序、状态口径和下一步以仓库根目录的 [`AGENTHUB_PLAN.md`](../AGENTHUB_PLAN.md) 为准。

> 说明：这里的“已完成”沿用旧口径，主要表示组件实现与测试已完成，不自动代表已接入应用入口或学习者已经掌握。


### 0.1 项目结构与 Go Module

- 状态：✅ 已完成
- 产出：项目目录、`go.mod`、`cmd/agenthub/main.go`
- 验证：`go run ./cmd/agenthub` 输出 `AgentHub starting...`

### 0.2 配置管理

- 状态：✅ 已完成
- 产出：YAML 配置、Viper 配置加载器、默认值、环境变量覆盖和配置校验
- 验证：默认配置、端口覆盖、多字段覆盖和非法端口四个场景均符合预期
- 关键概念：struct tag、指针传参、配置来源优先级、使用 `%w` 保留错误链

### 0.3 结构化日志

- 状态：✅ 已完成
- 产出：基于 `log/slog` 的统一日志组件、文本/JSON Handler、日志级别过滤和结构化启动日志
- 验证：文本、JSON、ERROR 级别过滤和非法日志级别四个场景均符合预期
- 关键概念：Logger 与 Handler 的职责、接口承接不同实现、日志级别门槛、组件低耦合

### 0.4 错误处理

- 状态：✅ 已完成
- 产出：应用错误类型、错误码、`New`/`Wrap`、错误链提取和统一启动错误出口
- 验证：配置错误可提取 `CONFIG_ERROR`；正常与失败进程退出码分别为 0 和 1；格式化、测试、静态检查和构建均通过
- 关键概念：`error` 接口、指针接收者、`Unwrap`、`errors.Is`/`errors.As`、`main` 与 `run` 的职责

### 0.5 工程命令

- 状态：✅ 已完成
- 产出：统一的 Makefile，支持帮助、运行、格式化、测试、静态检查、构建、组合检查和清理
- 验证：`make run`、`make check`、`make clean` 和命令行变量覆盖均符合预期
- 关键概念：Make 目标与命令、Tab 语法、`.PHONY`、目标依赖、退出码传播和变量覆盖

### 0.6 测试基础

- 状态：✅ 已完成
- 产出：`apperr` 与 `config` 单元测试、表驱动测试、临时配置和环境变量隔离，以及 `make coverage`
- 验证：`make check` 与 `make coverage` 通过；`apperr` 覆盖率 100%，`config` 覆盖率 92.1%，当前项目总覆盖率 58.0%
- 关键概念：`testing.T`、子测试、表驱动测试、测试辅助函数、`t.TempDir`、`t.Setenv`、测试缓存和覆盖率边界

### 1.1 Model 与 Message 抽象

- 状态：✅ 已完成
- 产出：统一消息角色与构造函数、模型请求响应、Token 用量结构和 `Model` 接口
- 验证：消息构造、模型成功响应、错误传播和 Context 取消测试全部通过；`internal/llm` 覆盖率 100%
- 关键概念：自定义字符串类型、可比较结构体、隐式接口实现、编译期接口断言、指针方法集和 `context.Context`

### 1.2 Tool 系统

- 状态：✅ 已完成
- 产出：Tool 定义与调用协议、函数式工具、并发安全 Registry、注册期 JSON Schema 编译、参数校验、Executor 错误分层，以及 Model/Message 的工具调用字段
- 验证：`internal/tool` 与 `internal/llm` 单元测试、数据竞争检查和 `make check` 全部通过
- 关键概念：接口扩展、函数类型与闭包、`sync.RWMutex`、防御性复制、JSON 语法与 Schema 校验、Context 错误传播、可变参数和深度比较

### 1.3 Agent Loop

- 状态：✅ 已完成
- 产出：Agent 构造与运行结果、直接回答、单工具调用、同轮多工具调用、连续多轮工具调用、Usage 累加和最大步数保护
- 验证：直接回答、工具闭环、多工具与连续调用、参数校验、工具业务错误回传、Context 取消、后续模型错误和最大步数测试均通过；数据竞争检查与 `make check` 通过
- 关键概念：Agent Loop、消息历史顺序、ToolCall ID 关联、模型步数与工具次数、业务错误和流程错误分层、`%w` 错误链、循环终止与副作用边界

### 1.4 Memory

- 状态：✅ 已完成
- 产出：Memory 接口、按消息条数裁剪的 SimpleSliding、Tokenizer 抽象、按 Token 预算裁剪的 TokenBudgetMemory，以及 Agent Loop 的逐轮 Memory 接入
- 验证：滑动窗口、System 消息保护、Token 预算裁剪与超预算错误测试通过；Agent、Memory、全项目测试、数据竞争检查、静态检查和构建通过
- 关键概念：全量 history 与模型视图分离、保留 System 消息、从旧到新裁剪、按条数与按 Token 裁剪的差异、依赖接口和逐轮重新裁剪
- 当前边界：Tokenizer 仍为课程用固定消息计数实现；摘要压缩将在后续接入真实模型能力时扩展

### 1.5 Streaming

- 状态：✅ 已完成
- 产出：Stream 与 StreamingModel 接口、Delta/Done 事件协议、统一流消费器、同步与流式共用的 Agent Loop，以及 `RunStream` 工具调用闭环
- 验证：流式直接回答、流式工具调用、参数与能力校验、EOF/未知事件、启动与接收错误、Handler/Close 错误、Context 取消和数据竞争测试均通过；`make test-pretty` 共 104 个测试通过，`make check` 通过
- 关键概念：方法值与函数类型、接口嵌入与类型断言、Done 与 EOF 的区别、流资源关闭、主错误优先、错误链传播，以及同步/流式逻辑复用
- 当前边界：流事件目前仅支持文本 Delta 与最终 Done；真实模型适配、工具参数增量和 SSE 输出将在后续课程扩展
- 下一步：暂停进入 1.6，先按调用链复习 1.1—1.5，并只添加说明设计意图而非复述语法的必要注释

### 复习检查点：Agent Loop

- 状态：✅ 已完成
- 产出：为 `Agent` 组装、共享响应生成器、完整历史、逐轮 Memory、Usage 累加、最大步数、工具执行与流式入口补充设计注释；保存 Agent Loop 流程图
- 验证：`internal/agent` 专项测试和数据竞争检查通过；全项目格式化、测试、静态检查和构建通过
- 关键概念：接口方法的具体实现由运行时对象提供、方法值与方法调用、模型步数与工具次数、Assistant ToolCall 与 ToolMessage 顺序、最后步骤的副作用边界，以及单次流 Done 与整个 Agent 完成的区别
- 下一步：复习 Streaming 的事件协议、资源关闭和错误优先级

### 复习检查点：Streaming

- 状态：✅ 已完成
- 产出：为 Stream、StreamingModel、StreamEvent、StreamHandler 和 ConsumeStream 补充设计注释
- 验证：`internal/llm` 专项测试和数据竞争检查通过；全项目格式化、测试、静态检查和构建通过
- 关键概念：同步与流式响应、接口嵌入、事件消费、命名返回值与 defer、Done 与 EOF、Handler 职责、主错误和 Close 错误优先级，以及单次模型流与 Agent Loop 的边界
- 下一步：开始 1.6 Agent Factory，通过配置统一组装 Agent 依赖

### 1.6 Agent Factory

- 状态：✅ 已完成
- 产出：Agent Factory 配置与依赖定义、Sliding/TokenBudget Memory 策略构造、统一 `Build` 入口、应用配置到 Factory 配置的转换、YAML/环境变量配置接入，以及配置边界、条件依赖和组装职责的设计注释
- 验证：配置合法与非法分支、YAML 解析、环境变量覆盖、两种 Memory 构造、依赖缺失、Factory 组装后真实运行均通过；`internal/config` 与 `internal/agentfactory` 数据竞争检查和 `make check` 通过
- 关键概念：值配置与运行时依赖、按需依赖校验、接口返回不同具体实现、工厂组装职责、分层校验、单向依赖和边界配置转换
- 当前边界：Factory 接受已经创建好的 Model、Registry 与 Tokenizer；真实模型适配和应用入口运行时组装尚未实现
- 下一步：进入 1.7 MCP，先学习协议边界、传输层与工具发现，再适配现有 Tool Registry

### 1.7.1 MCP Session 边界

- 状态：✅ 已完成
- 产出：MCP 远程工具定义、内容块与调用结果模型，以及支持工具发现、远程调用和资源关闭的 `Session` 接口
- 验证：工具发现、工具调用、Context 取消与会话关闭测试通过；`internal/mcpclient` 数据竞争检查和 `make check` 通过
- 关键概念：Host/Client/Session/Server 职责、MCP 协议模型与内部 Tool 模型隔离、一次调用的多内容块、工具业务错误与流程错误分层，以及由 SDK Adapter 隐藏生命周期和传输差异
- 当前边界：只定义稳定的客户端会话抽象和测试替身，尚未实现 MCP Tool Adapter、真实 SDK、传输或网络连接
- 下一步：实现 MCP Tool Adapter，将远程工具注册到现有 Registry 并通过 Executor 调用

### 1.7.2 MCP Tool Adapter

- 状态：✅ 已完成
- 产出：将 MCP `ToolDefinition` 转换为内部 `tool.Definition` 的 `ToolAdapter`，通过 `Session.CallTool` 执行远程工具，合并文本内容块，并接入现有 Registry 与 Executor
- 验证：定义转换、名称与参数转发、多文本块合并、MCP 业务失败转换、Context 取消、构造参数校验和 Executor 集成测试通过；`internal/mcpclient` 数据竞争检查与 `make check` 通过
- 关键概念：适配器模式、远程名称与模型可见名称分离、JSON 语法校验与 Schema 编译职责分层、`json.RawMessage` 防御性复制、`%w` 保留错误链、编译期接口断言，以及 MCP 业务错误到 Tool Result 的转换
- 当前边界：仅支持文本内容块；普通网络和协议错误仍按普通工具错误进入模型，精细错误分类、超时和重连留待后续实现
- 下一步：实现多 MCP Server 的 Session 管理、工具批量发现、命名空间和名称冲突处理

### 1.7.3 多 MCP Server 管理与名称冲突

- 状态：✅ 已完成
- 产出：Registry 原子批量注册、ToolAdapter 本地模型名与远程工具名分离、Manager 多 Session 管理、工具批量发现、`server__tool` 命名空间和同名 Server 拒绝策略
- 验证：批次成功、批次内重名、已有名称冲突、后置非法 Schema 不产生部分写入、两个 Server 的同名远程工具注册与执行、构造参数校验、重复 Server、注册失败后重试和 ListTools 错误链测试通过；`internal/mcpclient` 与 `internal/tool` 数据竞争检查及 `make check` 通过
- 关键概念：本地名称与远程名称分离、注册阶段与运行阶段分离、批量操作全成功或全失败、锁外准备与锁内提交、检查与写入的同一临界区、接口切片与具体类型切片不兼容，以及 Manager、Adapter、Registry 的职责边界
- 当前边界：Manager 在添加 Server 时串行持锁执行工具发现与注册；尚未负责关闭全部 Session、连接状态、超时分类、断线检测或重连
- 学习图示：`internal/mcpclient/mcp-multi-server-manager-flow.svg`
- 下一步：进入 1.7.4，设计 MCP Session 的超时、关闭、断线与可测试重连策略
