# AgentHub 二轮学习计划（缺口驱动）

## 1. 定位

一轮 A—I 已完整过一遍（存档：[`docs/archive/agenthub-plan-v1.md`](docs/archive/agenthub-plan-v1.md)，已关闭）。二轮不复述全部内容，按实际缺口重新排课，每条课程一个可运行、可验证的小目标。

二轮不是"重做一遍"，而是**把跟做过的能力变成能独立完成的能力**。

## 2. 二轮课程重点

- HTTP 筑基：标准库独立练习后对照 Hertz，重点理解路由分发、请求读取和响应写出。
- 独立复现：每节用小变体验证迁移能力，不以跟做或测试通过替代理解验收。
- Go 就地巩固：围绕当前代码解释接口、错误、并发、泛型等概念，不预设熟练程度。

## 3. 教学方式

沿用一轮原则（每节固定顺序：现象 → 问题 → 调用链 → 逐点解释 → 学习者写核心代码 → 运行 → 最小测试 → 回看抽象 → 理解验收 → Git 检查点；文件与方法解释模板；新增抽象门槛；四维度进度口径），并新增二轮强化：

- **HTTP 概念一律就地展开**：端点、Handler 签名、请求生命周期、状态码、JSON 约定、SSE 协议字节格式，每次出现都从"是什么、谁调用、不这样做会怎样"讲起。
- **每节末尾设"独立复现任务"**：不提供代码，学习者凭理解完成一个变体（重建 / 改造 / 解释），作为本节真正的验收项。
- **语法补丁课聚焦一轮出现过但未讲透的 Go 概念**：接口与错误链、并发原语、泛型与闭包。

## 4. 课程路线（三编，按课号顺序推进；语法补丁课可按需要提前插入）

### 第一编：HTTP 筑基（标准库从零，教学代码在 `examples/http-from-scratch/`）

目标：能独立写出并解释一个带 JSON、错误映射、中间件、SSE、测试的 HTTP 服务。

| 课 | 本节问题 | 学习者核心实现 | 涉及概念 |
|---|---|---|---|
| R.1 最小 HTTP 服务 | 框架用多了，HTTP 本质是什么 | 用 `net/http` 写 `/ping` 与 `/echo` | `http.Handler` 接口、`ServeMux`、请求行/头/体、状态码、`Content-Type` |
| R.2 JSON 与错误映射 | "返回 JSON"的约定是什么 | 手写 JSON 解析与 400/500 错误映射 | `json.Marshal/Unmarshal`、错误分层、日志 |
| R.3 中间件 | 日志与异常处理散落各处 | 手写 logging + recovery 中间件 | 函数式中间件、`defer/recover`、Handler 嵌套 |
| R.4 Context 与取消 | `request.Context()` 到底解决什么 | 慢 handler + 客户端断开观察 | `context`、取消传播、超时控制 |
| R.5 HTTP 测试 | 不会测 handler | `httptest` 单测 + 表驱动测试 | `httptest`、表驱动、集成测试 |
| R.6 SSE 从零写 | SSE 只见过 Hertz 封装 | 手写 `text/event-stream` 端点 | SSE 协议格式、流式写入、`io.EOF` |

### 第二编：对照 Hertz 与独立复现（对照 `internal/httpapi` 现有实现）

目标：把标准库理解迁移到现有 AgentHub，能独立重建三个核心端点。

| 课 | 本节问题 | 学习者核心实现 | 涉及概念 |
|---|---|---|---|
| R.7 Hertz 对照 | 标准库懂了，框架快在哪 | 把 R.1—R.3 迁到 Hertz，与 `internal/httpapi` 逐行对照 | 框架路由、请求上下文、SSE Writer |
| R.8 独立复现 `/chat` | 跟做一遍不等于会写 | 不参照现码重建最小 `/chat`（含 session 传参），逐行对照找遗漏 | SessionManager、结构体、map 并发 |
| R.9 独立复现 `/chat/stream` | SSE 协议理解是否落地 | 重建流式端点并对照 | channel、goroutine、`select`、EOF |
| R.10 独立复现 Workflow 端点 | 接线与类型流是否理解 | 重建简化 `/tech-proposal` 并对照 | 泛型、闭包、依赖注入、Compile |

### 第三编：生产化继续 + 语法补丁（沿一轮 I 主线）

目标：完成 I.6，补一轮未讲透的 Go 语法，以综合独立任务收口。

| 课 | 本节问题 | 学习者核心实现 | 涉及概念 |
|---|---|---|---|
| R.11 Prometheus 指标（I.6） | 从"能看日志"到"能看指标" | `/metrics` 端点 + HTTP 请求计数器/直方图 | 中间件复用、Histogram 分桶、标签 cardinality |
| R.12 语法补丁：接口与错误链 | 隐式接口与 `%w` 错误链只会用不会讲 | 读现有代码指出接口实现点与错误包装点 | `interface`、`fmt.Errorf`、`errors.Is/As` |
| R.13 语法补丁：并发原语 | goroutine/channel/锁何时用 | 分析并改写一处并发点（SessionManager 或 MCP） | goroutine、channel、`sync.Mutex/Once`、`time.Ticker` |
| R.14 语法补丁：泛型与闭包 | Eino 泛型节点如何工作 | 手写一个泛型辅助函数并接线 | 泛型、闭包捕获 |
| R.15 综合独立任务 | 二轮是否真正独立 | 从用户需求中选一个真实小功能，独立完成全流程 | 全部 |

## 5. 二轮完成验收

1. 能独立写出带中间件、JSON、错误映射、SSE、测试的 HTTP 服务，并解释每个部分的调用者与原因。
2. 不看参考，能沿调用链解释 `/chat`、`/chat/stream`、`/tech-proposal` 中任一端点的文件与方法。
3. 通过全部语法补丁课的理解验收。
4. 每课满足一轮的完成条件：实现、真实入口运行、观察成功与失败路径、能解释、检查通过、更新计划、Git 检查点。

## 6. 每节课的计划记录模板

```markdown
### X.Y 课程名称

- 状态：未开始 / 进行中 / 已实现待串联 / 已串联待理解 / 已掌握 / 阻塞
- 用户可见目标：
- 当前现象：
- 本节只解决：
- 不在本节解决：
- 调用链：
- 学习者核心代码：
- 独立复现任务：
- 关键 Go 语法：
- 最小测试：
- 运行验证：
- 理解验收题：
- Git 提交：
- 下一步：
```

## 7. 当前学习位置

- 当前阶段：**二轮第一编：HTTP 筑基**
- 当前路线决策：**旧一轮计划已关闭归档（`docs/archive/agenthub-plan-v1.md`）；二轮按缺口重新排课，R.1 起从标准库重走 HTTP。**
- 已掌握（二轮）：**R.1、R.2、R.3.1、R.3.2（当前教学范围）、R.4.1、R.4.2、R.5.1、R.5.2、R.6.1（固定事件、逐段写出与刷新范围）**
- 当前小节：**R.6.1 已验收；下一节 R.6.2 浏览器 EventSource 消费（未开始）。**

### R.1 最小 HTTP 服务

- 状态：已掌握；核心实现、独立变体与集中理解验收完成；缺参退出缺陷已由学习者改为请求级 400，本课测试、静态检查与构建通过。
- 用户可见目标：独立启动本地 HTTP 服务；`GET /ping` 返回 `pong`，`POST /echo` 原样返回纯文本请求体；错误方法返回 405，未知路径返回 404。
- 范围：只创建 `examples/http-from-scratch/`，复用根目录 `go.mod`；保留原 `cmd/agenthub`、`internal`、配置与历史示例，不迁移或删除。
- 学习者核心代码：`examples/http-from-scratch/main.go`；显式创建 ServeMux、注册两个 Handler，并监听 `127.0.0.1:8090`。
- 关键概念：方法与路径、ServeMux 分发、Handler 函数值、Request 指针、ResponseWriter 接口、先响应头再状态码再正文、请求体读取与大小限制。
- `/echo` 边界：请求体上限为 4 KiB；读入完成后才开始写成功响应，超限返回 413，其他读取失败返回 400；不解析 JSON。
- 调用链图：[`docs/images/r1-native-http-request-flow.svg`](docs/images/r1-native-http-request-flow.svg)。
- 最小验证：`go fmt ./examples/http-from-scratch`、`go test ./examples/http-from-scratch -count=1`、`go vet ./examples/http-from-scratch` 与临时输出路径构建全部通过，仅检查本课包。
- 辅助测试：新增 `main_test.go`，保护 `/ping`、中文与空请求体回显、4096 字节成功、4097 字节返回 413，以及模拟读取失败返回 400；路由接线通过真实请求检查，测试未复制生产路由。
- 真实运行验证：对正在运行的 `127.0.0.1:8090` 发起 curl 请求，`GET /ping` 200、中文 `POST /echo` 200、`POST /ping` 405、未知路径 404、4096 字节回显 200、4097 字节 413 全部通过；编译另生成到临时路径，未启动替换或关闭学习者现有服务。
- 学习图：除调用链外，保留 [`r1-echo-request-response.svg`](docs/images/r1-echo-request-response.svg) 与 [`r1-body-read-limit.svg`](docs/images/r1-body-read-limit.svg)，均已解析并渲染检查。
- 调试备注：当前正文日志位于错误判断前，可能打印读取失败时的部分数据；仅限本地教学，调试收尾建议删除，不用于生产请求体记录。
- 理解验收：集中校正 ServeMux 只注册、匹配和分发路由，监听由 Server 完成；读取错误必须先处理，避免第一次写出默认 200 后无法改回错误响应，或把部分正文当作成功。
- 独立复现任务：已添加 `GET /hello`，通过 `r.URL.Query().Get("name")` 读取查询参数；`name=Go` 与 `name=HTTP` 成功测试通过。曾由隔离测试复现 `log.Fatalf` 退出；现已改为 `http.Error` 返回 400 + `return`，成功写出使用 `fmt.Fprintf` 并检查错误。隔离测试进程通过随机端口 HTTP 请求验证缺参 400 后同一服务器仍可返回正常 200。
- 运行版本边界：8090 的现有进程仍返回带感叹号的旧正文，与保存代码不同；没有关闭该进程或向它发送缺参请求。最新 Handler 通过隔离真实网络测试验证，保存代码另经构建；手动运行时需重启现有服务。
- Git 检查点：本课采用 `feat(http): 完成原生 HTTP 基础与查询参数练习` 提交主题，提交范围为教学示例、最小测试、计划、示例说明与 R.1 图示，不 push；实际提交信息以 Git 历史为准。
- 下一步：完成 R.1 后进入 R.2 JSON 请求与响应，不在本节引入中间件、Agent 或持久化。

### R.2 JSON 请求与响应

- 状态：已掌握；核心实现、集中理解验收和 name 独立变体完成；JSON 导出字段与标签问题已修复，本课检查和最新程序真实 HTTP 验证通过。
- 用户可见目标：新增 `POST /greet`，接收 `{"name":"Go"}` 并返回 `{"message":"Hello, Go","name":"Go"}`；原 `/ping`、`/echo`、`/hello` 保持不变。
- 文件边界：在原示例目录新增 `json_handler.go`，包含 `greetRequest`、`greetResponse` 和 `greetHandler`；只在 `main.go` 增加路由注册，不新增包、全局状态或通用 JSON 封装。
- 关键概念：请求正文流与 JSON 数据不同；`ReadAll` 读取字节，`Unmarshal` 把 JSON 存入结构体，`Marshal` 把结构体转换成 JSON 字节，`Write` 才写回响应；导出字段、JSON tag、指针参数与局部变量作用域。
- 请求契约：`Content-Type` 必须为 `application/json`（允许合法参数，如 charset）；请求体上限 4 KiB；`name` 去除首尾空白后必填；成功返回 message 和小写 name 两个字段。
- 错误契约：缺少或错误 Content-Type 为 415；请求体超限为 413；读取失败、非法或不匹配类型的 JSON、空 name 为 400；响应编码失败为 500 并只记录服务端错误详情；已开始写出的失败只记日志，不能再改状态码。
- 当前简化边界：成功响应为 JSON，错误响应暂用 `http.Error` 的纯文本；采用 `ReadAll + Unmarshal`，未知字段按标准库默认忽略，重复字段按默认处理，不实现严格字段或重复键拒绝；只用固定 string 响应结构，正常请求无法触发编码失败，不人为增加故障注入参数。
- 调用链图：[`docs/images/r2-json-request-response.svg`](docs/images/r2-json-request-response.svg)，转换顺序已与当前实现核对。
- 最小验证：`go fmt ./examples/http-from-scratch`、`go test ./examples/http-from-scratch -count=1`、`go vet ./examples/http-from-scratch` 与临时输出路径构建通过，仅覆盖本课示例。
- 辅助测试：新增 `json_handler_test.go`，验证成功 JSON 能重新解析为恰含 message/name 的对象、首尾空白处理、引号/反斜杠/换行转义、非法 JSON、字段类型错误、空/缺少 name、null、缺少或错误 Content-Type、413 与模拟读取失败 400；固定字符串响应的 500 分支未人为触发。
- 真实请求：基线阶段 `/greet` 八组及三个旧接口回归通过；最终验收时 8090 无服务，临时启动最新构建的原入口，在真实 HTTP 上验证三组双字段成功（普通、去空白、引号/反斜杠）及非法 JSON、类型错误、空 name、415、413 五组错误，随后 `/ping` 仍可用；仅关闭自建临时进程，不遗留服务。
- 理解验收：已集中校正 Unmarshal 为 JSON 字节→Go 数据、Marshal 为 Go 数据→JSON 字节（非普通类型转换），先编码并检查错误再开始响应可保留返回 500 的机会；不再追加口试。
- 独立变体：已使用导出字段 `Name string` 并加 `json:"name"` 标签，直接赋值已清理的 name；双字段回归测试通过。标签决定 JSON 字段名，导出性决定字段是否参与标准库编解码。
- Git 检查点：本课提交主题 `feat(http): 添加 JSON 请求响应与字段校验`；范围为 JSON 示例、测试、入口接线、计划、示例说明和 R.2 图；不 push，实际提交哈希以 Git 历史为准。
- 下一步：R.3 中间件；本节不改生产 AgentHub、不接模型、数据库或 Hertz。

### R.3 中间件（分阶段推进）

- 阶段：R.3.1 请求日志；R.3.2 异常恢复。一次只带写当前阶段，不提前增加恢复、状态码捕获或通用中间件框架。

#### R.3.1 请求日志中间件

- 状态：已掌握；loggingMiddleware 与入口接线已实现，调用范围与执行顺序已确认；本课测试、定向竞态检查、静态检查和构建通过。R.3.2 已在当前普通 HTTP 教学范围完成，流式等边界仍按后续阶段处理。
- 用户可见目标：每个请求结束时统一记录方法、路径和 Handler 调用耗时，成功请求、400 等请求错误及 ServeMux 返回的 404/405 均可见；旧响应保持不变。
- 核心代码：新建 `examples/http-from-scratch/middleware.go`，实现 `loggingMiddleware(next http.Handler) http.Handler`；`main.go` 把 `loggingMiddleware(mux)` 传入 `ListenAndServe`。
- 封装必要性：已有四个路由，逐 Handler 添加相同请求日志会重复并漏掉未匹配路由；在 ServeMux 外包装一次覆盖统一请求边界。
- 数据与运行语义：启动时构造返回 Handler 并捕获 next；每次请求内单独记录 start，注册延迟日志函数，再同步执行 `next.ServeHTTP(w, r)`；不创建 goroutine，不读 Body、不写响应、不改变原 Handler。
- 关键语法：接口方法集、HandlerFunc 类型转换、匿名函数闭包、函数返回 Handler、defer 执行时机。`time.Since(start)` 放进延迟闭包内，不能在注册 defer 时提前计算。
- 日志边界：`method=%s path=%q duration=%s`；路径不含查询参数，%q 转义特殊字符；不记录正文、令牌或完整 URL。耗时表示下游 ServeHTTP 执行期间经过的时间，不是浏览器端到端耗时。
- 暂缓边界：不捕获 HTTP 状态码，不做 recovery；defer 在 panic 展开时可执行日志但不会恢复 panic，不能把记录日志宣称为正常成功。R.3.2 再建立异常行为基线。
- 学习图：[`docs/images/r3-request-logging-middleware.svg`](docs/images/r3-request-logging-middleware.svg)，教学目标示意。
- 验证：本例 fmt/test/vet/build 通过；新增 `middleware_test.go` 检查构造不执行请求、下游调用一次、同一请求/Writer 原样传递、请求体仍可读、响应不变、日志在下游返回后执行且不含 query/body；404/405 仍有日志。随机端口真实 HTTP 测试确认 `/ping` 正文与日志；定向 `go test -race` 通过。
- 理解确认：已统一校正启动时包装与请求时执行的区别；`next.ServeHTTP` 同步调用下游，defer 只记录日志、不恢复 panic。Express 的 app.use/next() 仅作框架机制类比，不假定 next() 返回意味着异步响应完成。
- 独立迁移验收：已确认仅对 `/greet` 加日志应包装该 Handler，而非整个 mux；当前仍保留全局日志，未改动运行范围。本阶段采用解释变体验收，不额外新增接口。
- Git 检查点：提交主题 `feat(http): 添加统一请求日志中间件`；仅包含本阶段中间件、测试、入口接线、计划、示例说明与学习图；不 push，实际提交哈希以 Git 历史为准。
- 下一阶段：R.3.2 异常恢复中间件，先观察 panic 请求行为，再决定恢复边界。

#### R.3.2 异常恢复中间件

- 状态：已掌握（当前普通非流式 HTTP 教学范围）；核心实现与入口接线完成，关键分支已集中讲解，恢复边界测试和真实入口验证通过。
- 基线证据：`recovery_baseline_test.go` 在随机端口启动只有日志包装的 Handler；响应前 panic 导致客户端 EOF，已 Flush 部分响应后 panic 导致 HTTP 200 加残缺正文/ unexpected EOF；两种情况下后续 `/ping` 均成功，证明标准库已经隔离请求 panic，但不会自动生成完整 500。
- 用户可见目标：新增 `recoveryMiddleware`，普通请求行为不变；当前请求在写出前 panic 时返回通用 500 并记录堆栈，写出后 panic 则中断而不追加错误正文；日志包在 recovery 外，记录恢复流程总耗时。
- 核心代码：在同目录新建 `recovery.go`；用小型 `recoveryWriter` 包装 ResponseWriter，记录当前请求是否已开始最终响应；`main.go` 组装 `loggingMiddleware(recoveryMiddleware(mux))`。
- 封装必要性：只靠 defer/recover 无法知道是否已发送 200；包装仅追踪 Write/WriteHeader，防止把残缺成功响应追加成假成功，不为状态码指标或缓冲系统预建抽象。
- 范围边界：本节仅覆盖当前普通非流式 HTTP Handler；只对同一 goroutine 中的 panic 有效，不拦截 os.Exit、另起 goroutine 的 panic 或不可恢复的运行时故障。ResponseWriter 的可选接口不做完整代理，SSE/Flush/Hijack 必须在相应阶段补齐，不能把此版本称为通用生产恢复器。
- 恢复逻辑：recover 必须由延迟函数直接调用；已有 ErrAbortHandler（含包装）继续抛出；其余 panic 服务端记堆栈，未 started 时清理 Content-Length/Content-Encoding 后 http.Error 500，已 started 时 panic(http.ErrAbortHandler) 交给标准库中断响应，不重试、不继续 panic 后业务代码。
- 验证顺序：先新增仅本地教学 `/debug/panic` Handler，在接 recovery 前观察一次客户端失败，后接 recovery 观察 500 与后续 `/ping`；写后 panic 由隔离测试验证，不新增更多演示端点。核心代码仍由学习者编写，测试由辅助补充。
- 最小验证：本例 fmt/test/vet/build、定向 `go test -race` 全部通过；新增 `recovery_test.go` 验证响应前 panic 500、清理旧头且不泄露内部值、响应后 panic 原样保留正文并抛 ErrAbortHandler、主动中断（含包装）继续传播、普通接口与外层请求日志正常。随机端口真实 HTTP 测试确认完整 500、部分 200 后 unexpected EOF、后续 ping 正常；原入口临时启动验证 `/debug/panic` 500 后 `/ping` 200，随后已关闭自建进程。
- 理解验收：已按当前代码集中校正业务 panic 与中断信号的目的、recover 与再次 panic 的传播方向、响应前返回500和响应后中断是互斥分支；started 表示最终响应已确定，不保证客户端收到字节。采用解释分支作为迁移验收，不追加口试或新接口任务。
- 图示：[`docs/images/r3-panic-recovery-boundary.svg`](docs/images/r3-panic-recovery-boundary.svg)，已渲染检查，内容与当前分支一致。
- Git 检查点：提交主题 `feat(http): 添加请求级异常恢复与响应边界保护`；仅包含本阶段恢复示例、基线和回归测试、入口接线、计划、示例说明与图，不 push；实际提交哈希以 Git 历史为准。
- 下一步：R.4 Context 与取消，学习正常取消与 panic 的不同责任。

### R.4 Context 与取消（分阶段推进）

- 阶段：R.4.1 客户端断开时停止等待；R.4.2 服务端主动限制处理时间。一次只实现当前可运行目标。

#### R.4.1 慢请求与客户端取消

- 状态：已掌握（当前客户端取消教学范围）；slowHandler 与路由实现、辅助测试、真实 HTTP、静态检查、构建与定向竞态检查通过，集中理解验收完成。
- 用户可见目标：新增 `GET /slow`，模拟5秒工作，正常完成返回纯文本 `slow work completed`；客户端提前断开后停止等待，服务日志记录 canceled，不尝试写回不存在的客户端；其他请求仍正常。
- 核心代码：同目录新建 `slow_handler.go` 的 `slowHandler`；main 只增加路由，保留日志/恢复中间件与旧接口。
- 数据流：`r.Context()` 取得本次请求的生命周期信号；单次 Timer 模拟工作完成，select 同时等待 timer.C 和 ctx.Done；取消分支由 Handler 主动 return，context 不会强制杀死 goroutine。
- 当前选择：用 select 监听 ctx.Done，并在计时器就绪后再检查 ctx.Err；若多个 select case 同时就绪不存在固定优先级，额外检查只缩小竞争窗口，不保证写出前绝不取消；写入错误仍检查。
- 语法与生命周期：Context 方法、只接收 channel、select 阻塞不忙轮询、Timer 和 defer Stop。使用 time.NewTimer 可在提前返回时主动停止计时器，不声称新版Go必须Stop才能防内存泄漏；不自己新增 goroutine或锁。
- 边界：5秒是教学模拟时长，不是HTTP协议规定；请求context取消并不自动中断 Sleep/CPU运算或回滚已发生副作用；不将客户端curl超时值自动当成服务端deadline；本阶段取消分支不写500/499，不恢复panic。
- 真实验证：8090已有教学服务，原入口curl确认正常慢请求返回200与完整正文，`--max-time 1` 以退出码28结束且未收到HTTP状态，随后ping返回200；没有替换、关闭已有进程。运行服务的终端日志未读取，取消与日志语义由隔离真实HTTP测试核验。
- 最小测试：新增 `slow_handler_test.go`，已取消context快速返回且不写正文、等待中取消、随机端口真实正常请求约5秒返回200、客户端TCP断开后Handler及时结束且context.Canceled、后续ping仍正常；日志分别记录完成/取消，没有panic。示例 fmt/test/vet/build 和定向race全部通过，核心代码不改。
- 理解验收：已通过当前代码解释与集中校正：Sleep 只暂停当前goroutine，不监听取消；Timer配合select同时等待完成和取消。Done返回取消时关闭的通知通道，Err返回取消原因（未取消为nil），取消不等于程序异常；由Handler主动return停止工作。采用解释作为迁移验收，不追加接口任务。
- 图示：[`docs/images/r4-request-cancellation.svg`](docs/images/r4-request-cancellation.svg)，教学目标与实现一致。
- Git检查点：提交主题 `feat(http): 添加慢请求与客户端取消处理`；仅提交本节Handler、测试、路由接线、计划、示例说明与图示。根目录未跟踪的 `http-from-scratch` 构建文件保留不纳入提交，.vscode/settings.json当前无变更；不push，实际提交哈希以Git历史为准。
- 下一阶段：R.4.2 使用请求context派生处理超时，区分客户端断开与服务端deadline。

#### R.4.2 服务端处理超时

- 状态：已掌握（当前协作式、非流式处理超时教学范围）；核心实现、路由接线、集中理解校正、专项测试和真实入口验证通过。
- 用户可见目标：保留 `/slow` 的5秒成功/取消行为；新增 `GET /slow/timeout`，同样的5秒模拟工作限制为2秒，未提前断开时返回503与纯文本 `slow work timed out`；客户端先断开则取消等待，不追加503。
- 核心代码：在现有 `slow_handler.go` 新增薄入口 `slowTimeoutHandler`，用 `context.WithTimeout(r.Context(), 2*time.Second)` 派生ctx，defer cancel，再调用 `slowHandler(w, r.WithContext(ctx))`；main只注册新路由。
- 封装必要性：两条真实路由复用同一等待/响应逻辑，超时入口只承担处理预算，不复制5秒工作或添加通用中间件。
- 当前实现调整：slowHandler的select只等待Done或Timer；出select后统一检查ctx.Err，避免两个分支重复取消/超时映射。DeadlineExceeded返回503，其他取消只记日志并return；无取消则写成功正文。
- 关键语义：WithTimeout不等待2秒才返回；父ctx取消传给子ctx，子ctx到期不取消父ctx，父deadline更早时仍以更早者为限。WithContext返回请求浅拷贝，不发网络请求，也不原地修改原请求；必须传递返回的请求才能让slowHandler监听子ctx。
- 生命周期：defer cancel在下游同步返回后释放派生ctx资源，不是立即取消；定时器Stop与ctx cancel分别结束模拟工作Timer和处理预算。超时通知由Handler协作处理，不强制终止CPU/Sleep/外部操作。
- 响应约定：本教学端点把处理预算耗尽映射为503，不把所有超时机械映射为504，也不把ctx自动取消等同于自动HTTP错误响应；未开始写响应时才能选择状态码，仍检查成功写出错误与写出竞争边界。
- 验证：现有8090教学服务curl确认新端点约2秒完整503、客户端--max-time 1以退出码28结束且无HTTP状态、旧slow约5秒200、ping200；没有替换或关闭用户进程。用户服务终端日志未读取，取消/超时日志由随机端口网络测试核验。
- 辅助测试：在 `slow_handler_test.go` 保留R.4.1回归并新增超时测试，验证已取消父ctx快速返回且不写错误正文、父更早deadline生效、真实2秒503且外层请求父ctx仍未取消、TCP断开后子处理及时结束、后续ping正常；日志区分超时/取消，没有panic。示例fmt/test/vet/build及全部慢请求定向race通过，核心代码未代改。
- 理解验收：已集中厘清空case仍执行接收、无default的select等待可进行的通道操作、选中后继续统一Err判断；select不是if，也不必须两个分支。DeadlineExceeded表示截止时间耗尽取消，不是任务正常完成。采用解释作为迁移验收，不追加任务。
- 图示：[`docs/images/r4-server-timeout.svg`](docs/images/r4-server-timeout.svg)，已渲染检查，内容与当前实现一致。
- Git检查点：提交主题 `feat(http): 添加请求级处理超时与错误响应`；仅含本节Handler调整、路由、测试、计划、示例说明与图示，.vscode/settings.json无变化；不push。未跟踪构建文件 `http-from-scratch` 保留不提交，实际哈希以Git历史为准。
- 下一阶段：R.5 HTTP测试，学习如何自己构造请求和断言这些行为。

### R.5 HTTP 测试（分阶段推进）

- 阶段：R.5.1 直接测试Handler与表驱动断言；R.5.2 再通过ServeMux/测试服务器验证路由分发。每次只推进一个阶段。

#### R.5.1 Handler 表驱动测试

- 状态：已掌握（直接Handler测试范围）；学习者完成四用例表驱动改写，状态/头/正文断言、反向验证、隔离缺参回归与包检查通过。
- 用户可见目标：改写已有 `main_test.go` 的 `TestHelloHandler`，一次命令分别显示Go/HTTP成功、缺少name、显式空name四个子测试；断言状态、响应头和正文。故意改错一个预期正文并观察失败，恢复后再通过，以验证断言确实能捕获差异。
- 核心代码：只替换 `TestHelloHandler`；既有import足够，不新建测试文件、不改helloHandler或main接线；保留 `TestHelloHandlerMissingNameDoesNotExit` 的隔离回归。
- 数据流：用例切片提供请求目标与独立预期；t.Run每次创建Request和Recorder，直接同步调用helloHandler；调用完成后Result取得响应快照，读取正文并按契约断言。不会启动服务、占用8090或发网络请求。
- 关键语法：`[]struct`匿名结构体切片、命名字段复合字面量、range的索引丢弃与当前用例、子测试匿名函数；Test签名与文件名约定、t.Fatalf/t.Errorf区别。t.Fatalf终止当前测试而非退出整个服务，不与log.Fatalf混淆；本节不t.Parallel。
- API范围：NewRequest构造服务端请求对象不发送；NewRecorder实现ResponseWriter并记录写出；Result必须在Handler完成后调用，Header为写出时快照，Body需关闭。预期结果按接口契约写，不从实际输出反推。
- 边界：直接调用Handler不经过ServeMux与中间件，不据此宣称路由方法限制或404/405已验证；普通内存Recorder不能代替真实网络取消/流式中断测试。
- 最小验证：四个具名子测试通过；使用临时Go overlay仅把go用例预期改为 `Hello, Wrong`，退出码1并定位正文断言失败，用户源码保持不变；正常源码四子项重新PASS。hello缺参独立进程回归、示例包test与vet通过，格式化与diff检查通过。仅测试变化，无需启动8090服务。
- 理解验收：已解释以用例数据组织四种查询参数场景，由同一测试流程直接调用Handler并比较独立预期；保留直接调用不验证ServeMux的范围边界。采用用例改写及错误预期验证作为迁移验收，不新增接口或追加任务。
- 图示：[`docs/images/r5-handler-test-flow.svg`](docs/images/r5-handler-test-flow.svg)，已渲染检查，与当前测试流程一致。
- Git检查点：提交主题 `test(http): 完善 hello 表驱动测试与响应断言`；仅本节测试、计划、示例说明及图示，.vscode/settings.json无变化；未跟踪的.DS_Store保留不提交，不push，实际哈希以Git历史为准。
- 下一阶段：R.5.2 在同样的请求/响应心智模型上加入真实路由分发。

#### R.5.2 路由分发测试

- 状态：已掌握（当前路由分发测试范围）；newMux提取、TestRoutes、真实HTTP回归及反向验证通过，路由与Handler责任边界已集中校正。
- 用户可见目标：运行TestRoutes自动核对ping200、hello成功200、hello缺参400、POST hello405（Allow包含GET/HEAD）、未知路径404；失败可定位到路由与方法约束，而非直接绕过mux调用业务。
- 核心代码：在现有main.go把全部七条路由原样提取为 `newMux() *http.ServeMux`，main调用newMux再接原日志/恢复包装；main_test.go新增TestRoutes并调用newMux，不复制路由注册，保留原测试。
- 封装必要性：新增真实测试调用方；main与测试需要同一份路由配置，提取消除重复注册与测试/启动漂移。newMux只构造路由不监听，不引入通用Server/配置层，也不改Handler。
- 数据流：NewRequest构造method与target，Recorder收集响应；newMux.ServeHTTP按模式匹配再调用Handler，或生成404/405。Result后断言状态、正文、Content-Type与Allow，五用例为表驱动测试。
- 关键语法：返回具体指针*ServeMux用于表达路由构造职责；return mux不会销毁返回对象；ServeHTTP是同接口下的入口但本次目标为路由器。GET模式也匹配HEAD，因此405 Allow应包含GET与HEAD；HTTP方法与查询参数含义不同。
- 范围边界：本节核心测试仍在内存里，不是发起网络请求；共用newMux能验证实际路由配置，但不因此宣称Listener/端口/日志/恢复整体通过。验收时辅助用同一mux进行随机端口真实HTTP回归并检查原入口。
- 验证：TestRoutes五子测试通过；临时Go overlay仅去掉GET方法约束，POST用例实际200/成功正文/缺Allow，三项断言失败退出码1，用户源文件未改；正常源码回归通过。示例fmt/test/vet/build通过。
- 辅助真实HTTP：新增TestRoutesRealHTTP，共用newMux及原日志/恢复顺序，在随机本地端口验证200/400/405/404、正文和响应头。8090无服务时临时启动最新构建原入口，curl五组结果通过，随后只关闭自建进程，无遗留服务。
- 理解验收：已解释缺参400由helloHandler字段检查返回，405由ServeMux按注册方法约束返回；直接调用helloHandler绕过ServeMux，注册的GET约束不会自动附到函数上。临时overlay对同一POST请求实测直接Handler200、经mux405，用户源码未改；采用解释及边界对照作为迁移验收，不追加任务。
- 图示：[`docs/images/r5-routing-test-flow.svg`](docs/images/r5-routing-test-flow.svg)，本节路由测试目标与实现一致。
- Git检查点：提交主题 `test(http): 共用路由注册并验证方法与路径分发`；仅本节路由提取、测试、计划、示例说明与图，.vscode/settings.json无变化；.DS_Store等无关文件保留不提交，不push，实际哈希以Git历史为准。
- 下一阶段：R.6 SSE，从完整响应转向逐段发送并刷新，届时再处理流式Writer能力边界。

### R.6 SSE（分阶段推进）

#### R.6.1 SSE 逐段响应

- 状态：已掌握（固定事件、逐段写出与刷新教学范围）；学习者实现事件循环、路由与FlushError，专项测试、真实逐事件接收、取消、竞态检查、静态检查及构建通过。
- 用户可见目标：新增GET /events，固定UTF-8单行数据；正常响应立即发送首条message，随后约每秒发送第二/三条message与done，共四个事件后结束；客户端提前取消时停止等待，不追加500或done。curl -N观察逐段而非结束后一次输出。
- 核心代码：学习者在recovery.go新增FlushError方法，先记录最终响应已开始再委托底层ResponseController.Flush；完成sse_handler.go的eventsHandler（固定列表、Ticker与Context等待、逐事件Write与Flush），newMux已注册GET /events。助手只补测试和课程记录，未代改核心逻辑。
- 刷新边界：现包装仅嵌入ResponseWriter接口，不会自动暴露底层可选刷新方法；ResponseController识别FlushError并由包装转发。不能只Unwrap绕开started追踪；本节不实现完整Flusher/Hijacker代理或声称通用生产恢复器。
- 数据语义：事件使用event与data字段、空行结束；固定message/done约定，done不是HTTP结束信号，Handler返回才结束本响应。EventSource自动重连与收到done后close放下一阶段。不接模型、不接用户输入，不预建通用SSE编码器。
- 资源与并发：同请求goroutine写响应，不另起生产goroutine；Ticker每请求创建并defer Stop，首事件立即写，其余select等ticker或取消，select后再次检查ctx.Err。取消与写出存在竞争窗口，不承诺任意时间取消都绝不写；写/刷新错误仅记录并return，已开始响应不再http.Error。
- 运行约定：Content-Type=text/event-stream，Cache-Control=no-cache；不设Content-Length/Connection/Transfer-Encoding，不加未使用代理配置。Flush只提交服务端缓冲，不保证穿过所有代理或等于客户端业务处理完成。
- 辅助测试：新增sse_handler_test.go，按空行读取完整事件，断言四事件字节、头/状态、首条提前到达、后续等待、done后干净EOF；覆盖已取消、首条刷新后取消、真实客户端关闭、写入/刷新失败、底层刷新不支持。recovery_test.go新增无正文刷新标started、错误转发、刷新后panic中断以及真实SSE部分响应后unexpected EOF与ping回归。
- 真实入口：保留既有8090服务，curl -N -i观察四事件分别约0.254/1.255/2.255/3.255秒到达（本次客户端实测，含启动和连接开销）；正常退出0，0.35秒客户端超时退出28且只收到首条，随后ping200、POST /events405且Allow=GET, HEAD。用户服务终端日志未读取，服务端取消原因与及时返回由随机端口真实HTTP测试核验。
- 最小验证：示例fmt/test/vet及临时路径build通过；SSE/刷新边界定向race通过。随机端口首事件约0.00025秒到达、其余约1/2/3秒；首事件后关闭Body，Handler返回并观察context.Canceled。时间断言使用宽松窗口，不承诺精确一秒或所有网络环境的延迟。
- 反向验证：临时Go overlay去掉Handler刷新，不修改磁盘源码；真实流测试失败退出1，发现缓冲到末尾形成固定Content-Length，证明测试能捕获漏Flush。原始源码字节核对未变，正常源码回归通过。
- 理解与迁移验收：集中厘清Ticker只按周期通知、for range决定事件总数、函数返回才执行defer Stop；NewTimer为单次通知，AfterFunc为延迟回调，通知式便于select同时等取消。不把done当HTTP关闭命令，Write与Flush职责不同；采用现代码解释及漏刷新反向验证，不追加接口口试。
- 图示：[`docs/images/r6-sse-write-flush.svg`](docs/images/r6-sse-write-flush.svg)，已渲染回读，与固定事件顺序和刷新调用链一致。
- Git检查点：提交主题 `feat(http): 添加 SSE 逐段响应与刷新边界验证`；范围仅本节Handler、路由、恢复刷新方法、相关测试、计划、示例说明和图示；.vscode/settings.json无变更，.DS_Store保留不提交，不push。实际提交哈希以Git历史为准。
- 下一阶段：R.6.2 使用浏览器EventSource消费事件并区分业务done、HTTP流结束与重连。
