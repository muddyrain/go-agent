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
- 自研 Runtime 的支持包目前仍位于 `internal/agent`、`internal/agentfactory`、`internal/llm`、`internal/memory`、`internal/tokenizer` 和 `internal/tool`；保留它们是为了维持 Phase A 的可运行对照，不表示生产入口继续使用它们。
- `internal/mcpclient` 仍属于早期自研协议实验，尚未进入正式入口；Phase D 接入真实 MCP 时重新评估，不直接当作现成生产实现。
- 新的生产能力统一从 `cmd/agenthub` 出发，并使用 Eino 承担通用 Agent Runtime 职责。
