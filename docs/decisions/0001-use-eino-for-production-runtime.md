# ADR-0001：生产 Agent Runtime 采用 Eino

- 状态：已接受
- 决策日期：2026-09-08

## 背景

AgentHub 在 Phase A 实现了自研的最小 Agent Runtime，包括 Model、
Message、Registry、Executor、Memory、Agent Loop 和流式协议。

Phase B 使用 Eino 复现了直接回答、工具调用闭环、流式输出和 Callback
观察。两条链路已经产生了等价的用户可见行为。

继续同时扩展两套 Runtime，会产生重复实现、重复测试和双轨维护成本。

## 决策

AgentHub 后续生产功能统一使用 Eino 作为 Agent Runtime。现有自研 Runtime 完成了帮助理解 Agent Loop、工具协议和流式处理的教学任务，后续只保留为学习对照，不再承载生产功能。

同一个生产入口只能使用 Eino，不允许同时运行自研 Agent Loop 和 Eino ReAct Loop。AgentHub 服务本身继续承载产品与应用职责，并通过 Eino 完成底层 Agent 执行。

## Eino 负责

Eino 负责与具体 Agent 产品无关的通用执行机制：

- 使用 `schema.Message` 表达 User、Assistant、ToolCall 和 ToolMessage。
- 通过 ChatModel 接口统一普通生成与流式生成。
- 向模型绑定工具名称、描述和参数 Schema。
- 通过 ToolsNode 解析工具参数、定位工具并执行调用。
- 通过 ReAct Agent 推进 Model → Tool → Model 循环、消息历史和结束判断。
- 通过 StreamReader 传递增量输出并处理流的结束与错误。
- 通过 Callback 观察 ChatModel、Tool 等组件的执行生命周期。
- 通过 Chain、Graph 和 Compose 编排后续更复杂的 Agent 流程。

AgentHub 不再自行扩展与以上职责重复的通用 Runtime 实现。

## AgentHub 负责

AgentHub 负责面向最终产品和具体业务的应用职责：

- 读取并校验应用、模型、工具和部署环境配置。
- 选择并组装具体模型、工具、Prompt、知识库和 Agent 用例。
- 提供交互式 CLI、HTTP/SSE、Web Playground 等产品入口。
- 管理 Agent、用户、会话及其持久化和恢复。
- 管理工具目录、MCP 连接以及工具的启停和业务配置。
- 实施身份认证、权限校验、安全策略、调用限额和审计。
- 将 Eino 或模型错误映射为稳定的产品错误和接口响应。
- 记录产品级日志、指标、Trace、成本和评测结果。

这些职责可以调用 Eino，但不应由 Eino 决定 AgentHub 的产品模型。具体代码边界等到真实 CLI、HTTP 或持久化调用方出现后再按需提取，不在本节预先创建通用 Runtime 接口。

## 为什么这样选择

1. 自研 Runtime 已帮助理解 Agent Loop、工具协议和流式消费，教学目标已经达到。
2. B.1～B.3 证明 Eino 可以承担直接回答、ReAct 工具循环、流式输出和 Callback。
3. 继续自研通用 Runtime 不会直接增加 AgentHub 的产品能力，反而会继续增加维护和测试成本。
4. 后续精力应投入真实模型、CLI、HTTP/SSE、持久化、RAG、MCP 和产品可观测性。

## 后果

### 正面后果

- 生产代码只有一套 Agent Loop。
- 可以直接使用 Eino 的 Tool、Stream、Callback、Graph 和后续 RAG 能力。
- AgentHub 可以集中开发用户可见的产品功能。

### 负面后果

- AgentHub 会依赖 Eino 的接口与版本变化。
- 应用层需要控制 Eino 类型的扩散范围。
- 现有自研 Runtime 需要在后续课程中明确标记为学习实现或进行清理。

## 约束

1. 后续生产功能默认基于 Eino 实现。
2. 不继续向自研 Runtime 添加生产特性。
3. 同一个生产入口不得同时运行自研 Agent Loop 和 Eino ReAct Loop。
4. 尚未出现真实重复或变化来源之前，不提前新增 Runtime 抽象。
5. 自研 Runtime 的保留和清理范围在 B.5 单独决定。
