# AgentHub 学习示例

`examples/` 保存课程中用于理解 Agent 原理和对照 Eino 行为的可运行示例。它们不是 AgentHub 的生产入口，也不应继续承载新的产品功能。

正式入口是 [`cmd/agenthub`](../cmd/agenthub)。后续真实模型、交互式 CLI、HTTP/SSE、持久化、RAG 等生产能力从该入口及其应用代码演进。

## 示例说明

| 目录 | 对应课程 | 用途 | 运行方式 |
|---|---|---|---|
| `selfbuilt-runtime` | Phase A | 观察自研 Message、Model、Registry、Executor、Memory 和 Agent Loop 如何组成工具调用闭环 | `go run ./examples/selfbuilt-runtime` |
| `eino-direct` | B.1 | 观察 Eino Chain、Compile、Runnable 和 ChatModel 的最小直接回答 | `go run ./examples/eino-direct` |
| `eino-tool` | B.2 | 观察 Eino Tool、ToolsNode 和 ReAct Agent 的同步工具调用闭环 | `go run ./examples/eino-tool` |

## 维护边界

- 示例允许为解释已有原理而修复错误，但不新增生产特性。
- 自研 Runtime 及其配置、测试和早期 MCP 协议实验已经收拢到 `selfbuilt-runtime/`；其中支持包位于 `selfbuilt-runtime/internal/`，Go 的 `internal` 导入规则会阻止目录树之外的生产代码依赖它们。
- 早期 `mcpclient` 绑定自研 `tool.Tool` 与 `Registry`，只保留为历史协议实验；Phase D 将围绕 Eino 与真实 MCP 连接重新实现，不从该包继续扩展。
- 新的生产能力统一从 `cmd/agenthub` 出发，并使用 Eino 承担通用 Agent Runtime 职责。
