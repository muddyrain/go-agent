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
| `hertz-from-scratch` | 二轮 R.7—R.9.1 | Hertz 对照练习；R.7.1 两条GET路由、查询参数、纯文本响应与404/405分发已验收；R.7.2 POST /greet 的JSON请求/响应及前置校验已验收（大小检查在正文接收后）；R.7.3.1 Use/Next请求日志已验收，方法/路径/同步处理耗时统一记录（不含query/body，不承担异常恢复）；R.7.3.2普通缓冲响应恢复已验收，GET /panic返回通用500、故障详情仅写服务端日志；R.7.4 GET /events固定四事件逐次刷新与取消已验收，当前netpoll开启断开感知、HTTP/1流式panic中断连接（不重写500，不保证HTTP2/其他运输实现）；R.8.1 POST /chat输入校验与普通规则回复已验收（收到前缀，不接模型或会话）；R.8.2独立消息处理函数与/upper指令已验收，回答/预期输入错误由函数返回、HTTP映射保留在Handler；R.8.3可选session_id的进程内成功处理计数已验收，无ID保持仅answer，不保存消息，重启清空，非鉴权或模型记忆；R.9.1 POST /chat/stream已验收，规则回答先生成再按码点发送JSON delta、最后done，输入错误在SSE前返回；共用会话计数，发送取消不回滚计数，尚未接模型流；课程状态以根学习计划为准；监听127.0.0.1:8091，保留标准库示例作对照 | `go run ./examples/hertz-from-scratch` |
| `hertz-chat-client` | 二轮 R.9.2 | 标准库Go客户端消费8091的POST /chat/stream；逐行读取当前单行JSON事件协议，空行分派delta/done；Builder累计、终端立即显示，核对完整done与正文正常结束后输出完成和可选会话次数。HTTP错误、缺done、读取/输出失败与Body关闭已测试，真实go run逐段显示和计数递增通过；课程完成状态以根计划为准。不是通用SSE解析器，不自动重连；10秒总超时仅用于短消息教学 | `go run ./examples/hertz-chat-client -message '/upper 你好 Go'` |
| `workflow-from-scratch` | 二轮 R.10.1—R.12.2 | 两节点 Eino Workflow：任务整理后生成固定规则模板，非模型生成。保留CLI；`-serve`启动8092的POST /tech-proposal，JSON task输入、result输出，启动构建一次，每请求用自身ctx调用Invoke；输入415/413/400、内部通用500、后续请求与并发结果独立已验收。R.11.1 GET /metrics展示目标POST按status分组的进程内累计计数，查询/405/404不计入；R.11.2增加以秒计的下游处理耗时Histogram，累计bucket/sum/count与恢复500及并发观察已验收；指标重启清空。R.12.1 CLI的-cancel-demo在执行前取消本次ctx，errors.Is识别取消并输出友好消息/exit1；与-serve冲突时启动前拒绝，普通任务和HTTP保持。R.12.2空task返回携带Field/Reason的输入错误，CLI用errors.As穿过包装链读取字段并输出友好提示/exit1；取消仍优先，HTTP原400/500策略不变；类型提取、包装与同文断链反证已验收。未接监控平台、SSE，不读项目证据或调用外部模型 | CLI：`go run ./examples/workflow-from-scratch -task '给聊天接口补一个测试'`；输入错误：`go run ./examples/workflow-from-scratch -task ' '`；取消：`go run ./examples/workflow-from-scratch -cancel-demo`；HTTP：`go run ./examples/workflow-from-scratch -serve`；指标：`curl -sS http://127.0.0.1:8092/metrics` |
| `concurrency-from-scratch` | 二轮 R.13.1—R.13.5 | 标准库并发入门：两个固定模拟任务通过goroutine重叠执行，WaitGroup先登记、各任务退出时Done、调用方等待后再总结。Sleep仅模拟耗时，不是实际I/O或取消机制；任务启动次序不保证，WaitGroup不承担共享数据锁或错误收集。R.13.2四任务共享计数，同一Mutex保护完整读改写，等待后输出4000；重复/独立调用、race及去锁/独立锁/漏解锁反证已验收。R.13.3无缓冲channel传递四份1000，单接收方汇总4000，不共同修改total；交接不等同发送方后续收尾。R.13.4单任务select成功/超时对照，工作等待、发送和接收均响应同一ctx取消，正常100毫秒取得42；defer Wait协调后台收尾，没有内置超时或严格取消优先。R.13.5 Ticker周期通知与快慢处理对照已验收：周期100毫秒，工作20/250毫秒，各以650毫秒期限停止，显示通知时间与实际开始时间，慢接收可迟到/丢tick；未完成工作不计数。默认main仅运行当前Ticker实验，不混入旧输出；旧两任务完整保留为runGoroutineDemo，Mutex/channel/select均保留独立测试。正式测试、race/vet/build、真实入口及取消/循环/Stop/时间语义反证通过；真实时间与次数受调度影响，精确时间只在虚拟时钟内断言。不改已有HTTP、会话存储或Workflow | 当前实验：`go run ./examples/concurrency-from-scratch`；竞态检查：`go run -race ./examples/concurrency-from-scratch`；全部实验测试：`go test ./examples/concurrency-from-scratch -count=1` |
| `generics-from-scratch` | 二轮 R.14.1 | 泛型与闭包入门：同一filter[T any]同步遍历整数或字符串切片，调用方以func(T)bool提供规则，输入/回调/输出保持同一元素类型，按顺序收集结果而不改写输入槽位。整数最低值3改6后复用同一闭包，第二次读取当前minimum；字符串HasPrefix区分大小写。独立入口仅三行，不运行旧实验。正式测试覆盖顺序/重复、回调次数、空输入、不同类型、浅拷贝及显式/推导一致，test/race/vet/build与真实入口通过；类型不匹配、any直接比较、漏接append返回、反向规则、冻结阈值、复用输入存储反证已验收。不构建通用集合库、不接模型、不改已有Workflow或依赖 | `go run ./examples/generics-from-scratch`；测试：`go test ./examples/generics-from-scratch -count=1` |

## 维护边界

- 示例允许为解释已有原理而修复错误，但不新增生产特性。
- 自研 Runtime 及其配置、测试和早期 MCP 协议实验已经收拢到 `selfbuilt-runtime/`；其中支持包位于 `selfbuilt-runtime/internal/`，Go 的 `internal` 导入规则会阻止目录树之外的生产代码依赖它们。
- 早期 `mcpclient` 绑定自研 `tool.Tool` 与 `Registry`，只保留为历史协议实验；Phase D 将围绕 Eino 与真实 MCP 连接重新实现，不从该包继续扩展。
- 新的生产能力统一从 `cmd/agenthub` 出发，并使用 Eino 承担通用 Agent Runtime 职责。
