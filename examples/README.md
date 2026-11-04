# AgentHub 学习示例

`examples/` 保存用于理解 Go/HTTP 基础、Agent 原理和对照 Eino 行为的教学示例。它们不是 AgentHub 的生产入口，也不应继续承载新的产品功能。

正式入口是 [`cmd/agenthub`](../cmd/agenthub)。后续真实模型、交互式 CLI、HTTP/SSE、持久化、RAG 等生产能力从该入口及其应用代码演进。

## 示例说明

| 目录 | 对应课程 | 用途 | 运行方式 |
|---|---|---|---|
| `selfbuilt-runtime` | Phase A | 观察自研 Message、Model、Registry、Executor、Memory 和 Agent Loop 如何组成工具调用闭环 | `go run ./examples/selfbuilt-runtime` |
| `eino-direct` | B.1 | 观察 Eino Chain、Compile、Runnable 和 ChatModel 的最小直接回答 | `go run ./examples/eino-direct` |
| `eino-tool` | B.2 | 观察 Eino Tool、ToolsNode 和 ReAct Agent 的同步工具调用闭环 | `go run ./examples/eino-tool` |
| `http-from-scratch` | 二轮 R.1—R.6 | 标准库 HTTP 教学练习；纯文本、JSON、全局日志、请求级 panic 恢复、R.4 请求取消/处理时限、R.5 表驱动/路由测试已验收；R.6.1 `/events` 固定四事件逐段刷新已验收，包装器补齐 ResponseController 刷新但未完整代理可选接口；R.6.2 `/events/demo` 浏览器逐事件显示、done关闭和手动停止已验收（从项目根启动） | `go run ./examples/http-from-scratch` |
| `hertz-from-scratch` | 二轮 R.7—R.8.3 | Hertz 对照练习；R.7.1 两条GET路由、查询参数、纯文本响应与404/405分发已验收；R.7.2 POST /greet 的JSON请求/响应及前置校验已验收（大小检查在正文接收后）；R.7.3.1 Use/Next请求日志已验收，方法/路径/同步处理耗时统一记录（不含query/body，不承担异常恢复）；R.7.3.2普通缓冲响应恢复已验收，GET /panic返回通用500、故障详情仅写服务端日志；R.7.4 GET /events固定四事件逐次刷新与取消已验收，当前netpoll开启断开感知、HTTP/1流式panic中断连接（不重写500，不保证HTTP2/其他运输实现）；R.8.1 POST /chat输入校验与普通规则回复已验收（收到前缀，不接模型或会话）；R.8.2独立消息处理函数与/upper指令已验收，回答/预期输入错误由函数返回、HTTP映射保留在Handler；R.8.3可选session_id的进程内成功处理计数已验收，无ID保持仅answer，不保存消息，重启清空，非鉴权或模型记忆；本节状态以根学习计划为准；监听127.0.0.1:8091，保留标准库示例作对照 | `go run ./examples/hertz-from-scratch` |

## 维护边界

- 示例允许为解释已有原理而修复错误，但不新增生产特性。
- 自研 Runtime 及其配置、测试和早期 MCP 协议实验已经收拢到 `selfbuilt-runtime/`；其中支持包位于 `selfbuilt-runtime/internal/`，Go 的 `internal` 导入规则会阻止目录树之外的生产代码依赖它们。
- 早期 `mcpclient` 绑定自研 `tool.Tool` 与 `Registry`，只保留为历史协议实验；Phase D 将围绕 Eino 与真实 MCP 连接重新实现，不从该包继续扩展。
- 新的生产能力统一从 `cmd/agenthub` 出发，并使用 Eino 承担通用 Agent Runtime 职责。
