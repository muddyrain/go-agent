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
- **先带写、再做迁移练习**：每个小节先围绕当前文件逐段解释用途、调用方、写法与原因，给出标有“现在请写入”的分段代码及完整对照；学习者写核心实现。讲解与运行后再安排小变体，不用无示例的独立任务替代授课。
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

- 当前阶段：**二轮第三编：指标入门与 Go 语法就地巩固（先在教学服务验证）**
- 当前路线决策：**旧一轮计划已关闭归档（`docs/archive/agenthub-plan-v1.md`）；二轮按缺口重新排课，R.1 起从标准库重走 HTTP。**
- 已掌握（二轮）：**R.1、R.2、R.3.1、R.3.2（当前教学范围）、R.4.1、R.4.2、R.5.1、R.5.2、R.6.1、R.6.2、R.7.1、R.7.2、R.7.3.1、R.7.3.2、R.7.4、R.8.1、R.8.2、R.8.3、R.9.1、R.9.2、R.10.1、R.10.2、R.11.1（各自限定教学范围）**
- 当前小节：**R.11.1 请求计数与 GET /metrics已验收；下一节 R.11.2 处理耗时 Histogram（未开始）。**

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

#### R.6.2 浏览器 EventSource 消费

- 状态：已掌握（当前固定事件与浏览器消费教学范围）；学习者页面、页面Handler与路由已保存，HTTP测试、真实浏览器逐事件显示/关闭/停止、重连对照和包检查通过。
- 用户可见目标：访问GET /events/demo，页面自动连接现有/events，三条message逐行出现；done显示发送完成并close，停止按钮可提前关闭，不继续等待或自动重连。
- 核心代码：同示例新增sse_demo.html与sse_page.go的eventsPageHandler；newMux只新增GET /events/demo。ServeFile使用固定路径examples/http-from-scratch/sse_demo.html，只返回该文件，不公开仓库目录；运行约定为项目根启动。
- 调用链：浏览器请求页面，Go返回HTML；页面JavaScript创建EventSource('/events')发起第二个请求；原eventsHandler写/刷新SSE，浏览器解析后按message/done事件名调用监听函数。两次请求各有自己的Handler/响应生命周期。
- 生命周期：message只追加event.data文本；done关闭并标完成；按钮关闭并标手动停止；error关闭并显示未正常完成，当前有限流选择不自动重试。close是客户端停止监听/重连，不等于服务端直接调用defer；服务端由请求取消或正常返回结束。
- 边界：无模型、Hertz、WebSocket、CORS配置、认证、续传、重试封装或前端构建；不新增Ticker、定时刷新页面或拼接不可信innerHTML；固定单行data无需JSON.parse。
- 关键概念：Go Handler可返回HTML，不在服务端执行JS；ServeFile没有error返回值，不能套if err写法；相对文件路径基于进程工作目录。EventSource构造发起连接，addEventListener登记回调而非立即执行，message和done职责分开；textContent安全显示文本。
- 辅助测试：新增sse_page_test.go，测试内用t.Chdir模拟项目根启动约定（不并行）；GET页面200且正文精确等于保存HTML，HEAD200无正文，POST405与Allow=GET, HEAD，子路径404不暴露目录。随机端口通过newMux和原中间件验证页面/事件两次独立请求、内容类型与done结尾，日志等待服务结束后核对。
- 浏览器验收：在既有8090服务上使用shot.py与真实Chromium，主页面四行输出与已完成/禁用按钮正常；临时同源iframe加载原HTML并仅包裹EventSource观测，四事件约2/1003/2003/3005毫秒到达，一次open和一次close，完成后继续观察3.5秒未重复连接；首条后点击停止，继续观察3.3秒仍仅首条，状态已手动停止、一次close，无额外消息。
- 重连对照：独立临时原生EventSource在done/error时不close，收到done后观察到第二次open，再主动清理。正式页面done/error都close，不能仅删done里的close就断言一定重连。未修改磁盘HTML、后端事件逻辑或用户服务进程。
- 验证边界：示例fmt、完整test、定向页面/SSE网络race、vet和临时路径build通过。桌面截图回读完整且无横向溢出/脚本错误；shot.py将四次主动关闭EventSource产生的requestfailed统一报network_error，验收脚本的close计数、输出不再变化和重连对照通过，用于解释此场景而非宣称报告零错误。缺少favicon仅为外观提示，保留原教学页面不顺带修改；未核验移动端实际交互。
- 理解验收：已确认路由返回HTML，由浏览器执行其中JS再请求/events；不是HTML Handler直接调用SSE Handler。message追加文本、done作为应用约定由监听回调close，服务端响应结束与客户端停止重连职责不同。采用实现解释和临时对照验收，不追加接口口试。
- 图示：docs/images/r6-browser-eventsource.svg，已渲染回读，与页面请求、事件请求和关闭关系一致。
- Git检查点：提交主题 `feat(http): 添加浏览器 SSE 消费与关闭验证`；仅本节页面、页面Handler/路由、测试、计划、示例说明及图示；.vscode/settings.json无变化，.DS_Store保留不提交，不push。实际哈希以Git历史为准。
- 下一步：本节验收后进入R.7 Hertz对照，把已建立的标准库HTTP心智模型迁移到框架；不在本节提前改生产代码。

### R.7 Hertz 对照（分阶段推进）

#### R.7.1 路由与 Handler 对照

- 状态：已掌握（当前路由与普通文本响应对照范围）；核心实现、测试、真实入口验证通过，方法配置与路由责任已集中校正。
- 用户可见目标：新增独立examples/hertz-from-scratch，监听127.0.0.1:8091；GET /ping返回pong换行，GET /hello?name=Go返回Hello, Go换行，缺少/空name400，POST hello405，未知路径404。保持原8090示例与生产代码不变。
- 核心代码：学习者只新建main.go；newServer构造server.New、注册两条GET，main调用Spin，两个Handler使用(_ context.Context, c *app.RequestContext)、c.Query与c.String；复用根go.mod已有Hertz v0.10.6，不新增模块或改依赖。
- 封装理由：newServer同时供启动与后续路由测试使用，避免两份注册表漂移；本节不建应用层/依赖注入框架。server.New不装默认中间件，日志/recovery对照留后续阶段；已有internal/httpapi/server.go使用Default，与此教学构造选择不同。
- Go/API：app.HandlerFunc是func(context.Context,*app.RequestContext)，不等于http.HandlerFunc；_是保留参数类型但不命名使用，c包含框架请求/响应操作，ctx用于取消/期限/向下游传递。c.String没有error返回，配置普通响应数据不等于客户端已收到，不能把它说成底层网络Write。WithHostPorts返回配置项，New接收可变数量配置项，构造不监听，Spin启动并等待结束。
- 预验证：本机缓存版本已查源码，临时内存PerformRequest检查Query与String字节；WithHandleMethodNotAllowed(false)时POST/HEAD hello404，开启后405，未知路径仍404。Hertz GET不自动包含HEAD，自动405不生成ServeMux同款Allow；本节明确这些默认差异，不假装完整HTTP响应契约完全等价。
- 方法配置：本节显式server.WithHandleMethodNotAllowed(true)用于延续405观察目标；不改现有AgentHub服务配置，不追加HEAD或NoMethod封装。
- 实现验证：助手新增main_test.go共用newServer，覆盖ping、Go/HTTP查询、缺少/空name、POST405、HEAD不自动注册、未知路径404与正文/内容类型。隔离实例开启/关闭配置对照，同一POST分别405/404，正确GET不变、未知路径始终404；直接调用helloHandler的POST仍200，证明GET约束由路由器执行。示例fmt/test/vet及临时路径build通过，未修改核心代码。
- 真实入口：保留既有8091服务，curl六组请求依次200/200/400/400/405/404且正文正确。没有停止用户进程；本节仅核验Hertz教学示例，不运行生产AgentHub或模型。
- 检查点：提交主题 `feat(http): 添加 Hertz 路由与 Handler 对照`，范围仅本节main.go、main_test.go、计划、示例说明与图示；.vscode/settings.json无变化，无关.DS_Store保留，不push，实际哈希以Git历史为准。
- 理解迁移：已集中校正GET注册而非函数本体执行方法限制；false不会把POST转成GET，true只识别同路径其他方法并返回405；关闭405识别不等于鉴权或隐藏所有访问。c.Query仍需业务空值判断，c.String负责普通响应构造。采用当前代码解释与隔离对照验收，不追加/chat或模型任务。
- 图示：docs/images/r7-hertz-handler-comparison.svg，标准库与Hertz对应职责及GET/HEAD默认差异，保存后渲染核对。
- 下一阶段：R.7.2对照JSON请求/响应，再按实际需要进入框架中间件与流式能力；一次只带写当前阶段。

#### R.7.2 JSON 请求与响应对照

- 状态：已掌握（当前默认非流式JSON请求与固定字符串响应范围）；学习者实现与接线完成，前置校验职责已确认，JSON契约/边界测试、真实入口、静态检查与构建通过。
- 用户可见目标：Hertz教学示例新增POST /greet，接收name并返回message/name两个小写JSON键；缺/错Content-Type415、4KiB以上已接收正文413、非法JSON/类型不符/去空白后空name400，旧ping/hello不变。
- 核心代码：学习者新增json_handler.go（greetRequest/greetResponse/greetHandler），main.go的newServer只增加h.POST('/greet', greetHandler)；不改标准库示例、生产代码、go.mod或全局请求限制。
- 当前数据流：string(c.GetHeader('Content-Type'))交mime.ParseMediaType；c.Request.Body()取得当前默认非流式模式下框架已接收的[]byte；len检查后json.Unmarshal到请求结构体；TrimSpace业务校验；构造响应结构体再c.JSON(200,response)。成功JSON、错误c.String纯文本，沿用R.2主要契约。
- 大小边界：len(body)>4096是业务接收上限，不是流读取过程限内存，不能与MaxBytesReader等价；本机Hertz默认框架正文上限4MiB，超过框架限制可能在Handler前拒绝，本节不改变全局限制或实现流式请求。Body()不返回error，流式或失败读取场景可改BodyE/Body，但本节不声称覆盖所有读取失败。
- API选择：保留标准库Unmarshal以便直接对照完整JSON解析；可行替代c.BindJSON(&req)封装绑定，但当前默认单次Decode对尾随JSON行为不等价，本节不静默换解析语义。c.JSON接收Go对象，负责编码/Content-Type/响应数据，没有error返回；当前固定字符串响应无法触发正常请求编码失败，编码失败会经Render panic，不声称自动返回500，recovery下一阶段处理。
- 语法：导出字段与json标签；&req允许解码写入调用方变量，响应传值即可；[]byte转string只是头部类型转换，不能把Marshal后的[]byte再传c.JSON（会再次编码）。Query读取URL与Unmarshal读取JSON Body不同。
- 辅助测试：新增json_handler_test.go，共用newServer验证七种成功JSON（普通/去空白/引号反斜杠换行/中文/4096字节/未知字段/重复字段），重新解析map断言恰含小写message/name两键；十六种错误场景覆盖Content-Type、4097字节、校验顺序、非法/类型不符/尾随JSON、空body和缺失/空白/null name；另验证GET /greet405。保留全部R.7.1路由与方法配置测试。
- 真实入口：保留既有8091进程；curl九组greet请求验证成功、去空白、转义、非法JSON、错误字段类型、空白name、415及4096/4097字节边界，返回200/400/415/413符合契约；另GET /greet405、ping200、hello200。未读取用户终端日志，未停止或替换用户进程。
- 验证：示例fmt、完整包test、vet、临时路径build与diff检查通过；临时Go overlay仅将响应json标签改为大写Name，精确键名断言失败退出1，原始源码字节未变；正常源码重新PASS。当前不覆盖框架上限之前的读取失败、生产流式请求或无法由固定字符串正常触发的编码500。
- 理解验收：已确认成功响应前依次检查Content-Type、已接收正文字节数、JSON语法与字段类型、TrimSpace后的name；JSON解析成功不保证业务字段有效，JSON Body字段不是URL查询参数；len大小检查不是读取过程中限内存，成功响应由c.JSON编码。采用实现说明与反向契约验证，不追加接口口试。
- 图示：docs/images/r7-hertz-json-flow.svg，已渲染回读，顺序与职责对应当前实现。
- Git检查点：提交主题 `feat(http): 添加 Hertz JSON 请求与响应校验`；仅本节Handler、路由、JSON测试、计划、示例说明与图示；.vscode/settings.json无变化，无关.DS_Store保留，不push。实际哈希以Git历史为准。
- 下一阶段：R.7.3框架中间件对照，本节不新增recovery、日志状态、重试或模型能力。

#### R.7.3.1 请求日志中间件对照

- 状态：已掌握（当前同步处理链与请求日志教学范围）；用户核心实现与Use接线正确，计时与defer日志用途已说明，日志测试、真实入口、race、静态检查与构建通过。
- 可见目标：改动前只有框架启动/路由登记日志，没有自定义每请求耗时；完成后ping、hello、greet和当前404/405请求均打印一条method/path/duration，HTTP响应保持不变。
- 核心改动：学习者新增examples/hertz-from-scratch/middleware.go，直接定义loggingMiddleware(ctx context.Context,c *app.RequestContext)，不接收next、不返回Handler；main.go的newServer在server.New之后、全部路由注册之前h.Use(loggingMiddleware)。没有配置输入或第二个调用方，不新建返回闭包的工厂。
- 当前链：路由器先匹配并选出处理链；logging记录每请求start并登记defer；c.Next(ctx)在同一调用栈同步执行后续链；后续返回后中间件退出，defer计算time.Since(start)并log.Printf。相比标准库next.ServeHTTP，把后续处理链的保存/推进交给Hertz。
- 关键边界：Next不是网络请求、后台任务或取消通知；该版本框架外层本身也循环推进链，删Next/只return不会可靠地终止后续Handler，而会让defer日志先执行；真正中止链用Abort并return（只讲区别，不实现鉴权）。路由在注册时合并当时的中间件，临时实例实际验证Use放在GET后不能追溯覆盖已注册的GET；当前先Use再注册。
- 日志：c.Method/c.Path返回[]byte，当前fmt的%s/%q可直接格式化，不必转string；Path不包含query，不记录body/header或个人信息。耗时为本节同步处理链窗口，不保证客户端收到或覆盖网络读写/流式后台工作；defer能在当前栈panic展开时打印，但不是recover，不承诺500。
- 替代：可直接在Next之后log.Printf，正常返回可用但panic展开会跳过；选defer闭包将时间计算延迟到退出。可定义loggingMiddleware() app.HandlerFunc再返回闭包，但本节没有配置或共享依赖需捕获，使用直接函数及h.Use(loggingMiddleware)。
- 临时预验证：200/404/405各一条日志、不含query，响应状态不变；Next存在时顺序before→Handler→after→defer，删除Next时before→after→defer→Handler；Use晚于已注册GET时该GET无日志。均在助手工作区独立程序运行，非用户实现验收，无项目依赖变更。
- 辅助测试：新增middleware_test.go，临时接管标准log并在Cleanup恢复，不并行；共用newServer七种请求核对method/path/duration、一请求一条且不含query/body；隔离延迟Handler连续两次各等待25ms，断言日志在下游返回后、耗时包含等待、请求体/响应头/状态/正文不变、下游各执行一次；仅测试调用方捕获原值panic，确认defer仍打印但logging不recover。没有给服务注册故障路由或添加恢复器。
- 真实验证：既有8091服务六组curl回归200/200/400/405/404/200，未控制用户进程或读取其终端日志；额外Go overlay只把同一main入口监听改为随机端口，真实curl六请求逐项核对响应和独立进程日志，各一条且无query/body。隔离服务已停止，磁盘入口源码字节未变，未改项目端口或依赖。
- 检查：示例fmt、完整包test、定向日志race、vet、临时路径build和diff检查通过；反向Go overlay仅移除Next并保留ctx使用，原测试发现日志早于Handler且耗时未覆盖25ms而失败退出1；磁盘中间件未变，原码再运行完整包test通过。
- 理解范围：已说明每请求先保存当前时间，defer退出时打印方法、路径和耗时；duration是经过时间不是当前时间戳。实现中Next同步包住后续链，通过日志顺序和耗时反向测试核验，不将本节日志用途理解等同于已掌握recovery或异步/流式生命周期。
- 图示：docs/images/r7-hertz-middleware-next.svg，已渲染回读，登记与请求执行分离、Next同步调用和defer返回顺序对应当前实现；原稿入仓库，预览仅留工作区。
- Git检查点：提交主题 `feat(http): 添加 Hertz 请求日志中间件对照`；范围仅本节middleware.go、middleware_test.go、main.go接线、计划、示例说明与图示；.vscode/settings.json无变化，无关.DS_Store保留，不push；实际哈希以Git历史为准。
- 下一阶段：R.7.3.2异常恢复对照；本节不添加recovery、panic路由、status包装器、模型或流式实现。

#### R.7.3.2 普通响应异常恢复对照

- 状态：已掌握（当前普通缓冲响应、同栈panic教学范围）；学习者核心实现与接线正确，recover用途已说明，恢复器测试、真实请求、race、静态检查与构建通过。
- 可见目标：GET /panic 固定触发demo panic，经应用恢复器返回500与internal server error换行（text/plain; charset=utf-8）；服务端记录method/path/value/stack，随后GET /ping仍200。固定故障端点仅本地教学，不引入真实业务故障参数。
- 核心代码：学习者新增examples/hertz-from-scratch/recovery.go，含直接函数recoveryMiddleware(ctx,c)与固定panicHandler；newServer保留server.New，Use按loggingMiddleware、recoveryMiddleware顺序，全部路由前登记，新增GET /panic；不改标准库、生产代码、go.mod或添加第二套默认恢复。
- 恢复流程：defer内直接recover；nil不改正常响应；非nil记录服务端日志和debug.Stack；c.AbortWithMsg重置尚未发送的普通Response头/正文，设置500与通用正文并Abort剩余链；再SetContentType显式UTF-8纯文本；c.Next(ctx)同步包住后续链。恢复后当前中间件返回，不回到业务panic位置，也不继续执行原Next后的语句。
- 边界：仅普通缓冲响应、同一调用栈panic。c.String先构造普通Response，不等标准库ResponseWriter.WriteHeader已经确定网络响应；缓冲成功正文可清掉改500，不搬recoveryWriter.started或http.ErrAbortHandler（后者不是Hertz中断协议）。流式/接管/已经发送、独立goroutine、os.Exit、fatal或业务回滚均不在本节保证内。恢复器日志自身panic也不由同一defer兜底。
- Abort语义：停止尚未执行的Handler，不是取消context或关闭服务；AbortWithMsg会清掉整个旧Response头/正文，再构造500，参数顺序msg/code，与String的code/format不同；SetContentType在重置之后。单独String不能替代清理和Abort。
- 顺序：logging外层→recovery→业务；业务panic先被recovery处理，再回外层日志退出；当前recovery只覆盖其后的链，不保证覆盖外层日志自身错误。Use只登记函数，不调用业务；保留New以显示恢复来源。可选现成recovery.Recovery或server.Default，但不静默替换当前错误正文/头清理教学契约。
- 语法：value:=recover返回any，不仅error；if value==nil是普通请求分支；debug.Stack返回当前goroutine堆栈[]byte；日志%s方法/堆栈、%q路径、%v任意panic值，只在服务端打印，不写回客户端。panicHandler两个参数用_表示保留签名但不使用。
- 预验证：临时程序正常ping不变、panic500、缓冲后panic500且旧Content-Encoding/X-Stale清掉、剩余Handler不执行、后续ping正常；无恢复实例panic原值传到测试调用方。随机端口真实curl两种故障500与后续ping200，服务端两条panic日志，隔离进程已停止；这些是教学方案预验证，非最新用户代码验收。
- 辅助测试：原TestLoggingMiddlewareDoesNotRecoverPanic保持server.New仅Use(loggingMiddleware)隔离日志职责；新增recovery_test.go，核对正常201正文/头不变；string/error/数字panic均返回精确通用500，服务端一次故障日志含stack并先于外层请求日志，不记录query；panic后的业务语句不恢复、尚未执行的Handler不运行；普通缓冲成功正文和旧Content-Encoding/Content-Length/X-Stale清理；注册GET /panic500、后续ping200、POST /panic405。故障分支均为隔离测试实例，不为正式教学路由添加额外输入。
- 真实验收：既有8091的GET /panic、随后ping、缺参hello、POST /panic、未知路径依次500/200/400/405/404且正文正确，未控制用户进程或读取其终端日志；另用Go overlay仅在隔离实例改为随机端口、添加缓冲后panic测试路由，同一核心实现真实返回两次干净500与后续ping200；服务端两条panic日志含stack、三条请求日志，故障日志先于相应请求日志，query未入日志；隔离进程已停止，项目入口字节未变。
- 验证：示例fmt、完整包test、定向恢复/日志race、vet、临时路径build与diff检查通过；反向overlay把AbortWithMsg替换为只AbortWithStatus和SetBodyString（不清旧头），缓冲清理测试发现stale headers而失败退出1；项目恢复器字节未变，原码重新完整包test通过。不将此验证扩展为已发送流式响应/其他goroutine/副作用回滚保证。
- 理解范围：已说明recover接住panic；当前代码及多类型测试进一步区分panic值不一定是error，正常recover为nil，恢复后结束失败调用而非回到panic行。保留普通缓冲限定，不提前标记流式异常生命周期已掌握。
- 图示：docs/images/r7-hertz-recovery-buffered.svg，已渲染回读，对应正常不改响应、故障日志与缓冲重置、中止链和外层日志返回；原稿入仓库、预览不提交。
- Git检查点：提交主题 `feat(http): 添加 Hertz 普通响应异常恢复`；仅恢复器/故障Handler、main接线、recovery_test.go、原日志隔离测试调整、计划、示例说明与图示；.vscode/settings.json无变化，无关.DS_Store保留，不push；实际哈希以Git历史为准。
- 下一阶段：R.7.4 Hertz流式响应对照；不在本节提前实现流式恢复、模型、数据库或生产配置。

#### R.7.4 固定 SSE 流式响应对照

- 状态：已掌握（当前固定四事件、netpoll取消与HTTP/1流式中断教学范围）；用户核心实现与接线完成，发送循环、模式分支和取消通知职责已校正；测试、真实逐事件请求、断开取消、流式panic中断、静态检查与构建通过。
- 可见目标：GET /events与标准库同样推送三条message和一条done，首条立即，随后约1/2/3秒到达；客户端提前断开时停止等待；旧ping/hello/greet/panic响应不变。不接模型，不新建浏览器页面或通用SSE封装。
- 核心实现：学习者新建examples/hertz-from-scratch/sse_handler.go，以eventsHandler(ctx,c)保留固定四事件、Ticker/select/ctx.Err检查，采用sse.NewWriter(c)及writer.WriteEvent("",event.name,[]byte(event.data))；错误只记录并return。同一个请求中同步循环，不创建业务goroutine。main.go的New增加WithSenseClientDisconnection(true)，newServer增加GET /events。
- 职责迁移：Hertz sse.NewWriter设置no-cache与text/event-stream; charset=utf-8并接管Response写入；构造不立即发送。WriteEvent编码event/data/空行后写并Flush，返回编码/写/刷错误，不需要重复Fprintf或c.Flush。id空字符串表示本节不发id、不做续传，data字节不必JSON；不使用普通c.String拼SSE正文。不手设Content-Length/Transfer-Encoding或调用底层Finalize。
- 生命周期：循环控制总数，Ticker只是周期通知；首次不等，后续select同时等待tick/ctx.Done，随后统一检查Err；返回时Stop，不关闭ticker.C。done为应用事件，不关闭网络；Handler与中间件返回后Hertz对已登记writer Finalize结束响应。Writer.Close可提前完成响应，但本节交给框架统一收尾，不加defer Close，避免panic恢复前先正常结束流；WriteEvent本身已经Flush，不额外重复刷。日志的同步处理窗口含这段循环，不等客户端业务完成。
- 断开感知：WithSenseClientDisconnection默认false，当前选项仅netpoll适用，本地Mac默认netpoll下启用后连接断开取消传入ctx；不等服务端超时预算，不用Finished代替ctx.Done，不声明所有平台/运输实现相同。Windows标准网络等跨设备行为仍需重验，不修改全局网络实现。
- 必要恢复器改动：旧AbortWithMsg会Response.Reset并清掉已登记hijackWriter，流式panic不能靠重置生成正确500。学习者在recovery故障日志之后、普通500之前判断c.Response.GetHijackWriter()!=nil：Abort剩余链，GetConn().Close关闭当前连接并检查错误，然后return；否则保留原普通500逻辑。此判断是已切换写入模式的保守边界，不是已发送字节计数；创建writer尚未发首条时发生panic也中断，不补造500。仅当前HTTP/1教学，非HTTP/2多路复用通用方案；不重新panic http.ErrAbortHandler、不关闭整个服务。
- 已有对照：internal/httpapi/server.go的streamChatHandler同样使用sse.NewWriter与WriteEvent，但当前包含模型/会话/额外并发；本节只提取框架写入机制，不代搬生产实现或修复历史生产分支。
- 方案预验证：隔离真实HTTP四事件字节一致，约0.001/1.002/2.001/3.002秒到达且干净EOF；curl0.35秒超时仅首条，启用断开感知后0.65秒内观测context canceled，默认false未出现该及时取消。流式写首条后panic，客户端保留200与partial事件、IncompleteRead，未混入500正文，后续ping正常；普通panic仍500。旧恢复器对照同样无法生成有效500，源码确认Reset丢hijackWriter；本节分支明确中断并保留写入模式。所有预验证程序与日志仅工作区，隔离进程已停止，不控制用户8091服务。
- 辅助测试：新增sse_handler_test.go，隔离ExtWriter记录精确四事件字节、四次Write/Flush、SSE头与Handler不主动Finalize；已取消零写入/首条Flush时取消仅一条、Write和Flush失败后退出且不追加普通错误或done；隔离连接核对创建Writer尚未发送/已经首条后panic均Abort并Close当前连接、保留Writer、不重写500，关闭失败才记录额外日志；POST /events405与断开感知配置。保留普通恢复器与JSON/路由全部回归测试。
- 真实入口：保留用户8091进程，完整流四事件约0.004/1.005/2.005/3.005秒到达，SSE类型正确、正常EOF；curl0.35秒退出28仅首条，随后ping200、POST events405、普通panic500，未控制用户进程或读取用户终端。额外隔离Go overlay运行同一项目核心，四事件约0.001/1.003/2.003/3.004秒到达，chunked且无Content-Length、SSE/no-cache正确；提前断开观测context canceled；首条后panic保留200/partial+IncompleteRead，创建Writer尚未发送panic为RemoteDisconnected无响应，后续ping正常，普通panic仍500；无故障正式路由新增。
- 反向验证：初次尝试覆盖模块缓存SSE源码被Go明确拒绝，未修改缓存；改为仅在项目Handler临时overlay包裹Writer，阻止逐事件Flush而保留Finalize，四事件全部约3.004秒才到达，证明刷新必要。项目main/Handler和缓存源码未改，隔离进程均已停止，临时材料不入仓库。
- 检查：示例fmt、完整包test、定向Events/Recovery/Logging的race、vet、临时路径build与diff检查通过；恢复器旧注释同步为普通500/HTTP1流式中断。没有验收浏览器消费、Windows运输实现、HTTP2、代理透传或模型生命周期。
- 理解验收：已确认切片控制总数、Ticker通知与select等待、WriteEvent编码并刷新；断开感知使连接断开关联ctx取消，代码仍要检查Err并return；false不保证完整循环，写入错误仍会结束。netpoll是通用网络底层，不是SSE专有协议；流式panic关闭当前连接不是关闭服务或直接取消上下文，Close失败才打印关闭错误。开关不能防止全部写入错误，保留取消与写入两层检查。
- 图示：docs/images/r7-hertz-sse-flow.svg，已渲染回读，NewWriter、WriteEvent格式/刷新、循环与取消、框架收尾及流式panic中断职责对应当前实现；原稿入仓库，临时预览不提交。
- Git检查点：提交主题 `feat(http): 添加 Hertz SSE 流式响应与取消对照`；仅sse_handler.go、sse_handler_test.go、恢复器分支、main接线、计划、示例说明与本节图示；.vscode/settings.json无变化，无关.DS_Store保留，不push。实际哈希以Git历史为准。
- 下一步：R.8最小/chat独立复现，先建立输入→处理→响应心智模型，不直接代写生产AgentHub或接模型。

### R.8 最小 /chat 独立复现（分阶段）

#### R.8.1 输入校验与普通响应

- 状态：已掌握（当前输入校验与普通规则回复教学范围）；学习者按逐段讲解保存结构体、Handler与POST注册，处理链用途已说明；完整包测试、静态检查、构建与真实请求验证通过。本节为规则回复，不代表模型/Agent或完整R.8会话能力已完成。
- 可见目标：当前Hertz教学服务新增POST /chat，接收JSON message，去两端空白后返回仅answer一个小写JSON键，值为 `收到：` 加处理后的message。其余路由及中间件不变，无网络模型、数据库、会话、并发或配置输入。
- 文件与职责：学习者新建examples/hertz-from-scratch/chat_handler.go，包含chatRequest(Message string、json:message)、chatResponse(Answer string、json:answer)和chatHandler(_ context.Context,c *app.RequestContext)；main.go的newServer只新增POST /chat注册。不照抄internal/httpapi、不创建新模块/目录，不将教学功能移入生产代码。
- 本节契约：依次mime.ParseMediaType检查Content-Type为application/json（含合法参数）否则415；c.Request.Body取得字节后>4096返回413；json.Unmarshal非法或字段类型不匹配400；TrimSpace(message)后空值400；正常c.JSON200。错误正文依次为Content-Type must be application/json换行/request body too large换行/invalid JSON request换行/message is required换行，错误纯文本，成功JSON。沿用当前解析默认行为，不严格拒绝未知/重复键。
- 教学与迁移：先按当前文件连续讲解结构体、Content-Type、已接收大小、Unmarshal、业务空值与响应构造，提供每段“现在请写入”及最终完整文件与路由位置；用户编写核心。理解并运行后再做小变体，不把已学过API等同于能独立拼出函数，也不以背API名称为验收。
- 辅助测试：新增chat_handler_test.go，七种成功输入覆盖普通、去两端空白、保留中间空格、转义、4096字节、未知/重复字段默认行为；反解map检查恰含小写answer一个键、正文为收到前缀加处理后的message；十四种错误覆盖Content-Type、超限及顺序、非法/类型不符/尾随JSON、空缺/null message，错误仍精确纯文本；GET /chat405。保留其余路由、JSON、中间件与SSE测试。
- 验证：示例fmt、完整包test、vet与临时路径build通过；真实8091七组chat请求分别核对200规则回复、空白400、类型400、非法JSON400、媒体类型415、4096字节200、4097字节413，另GET chat405、ping/hello200回归；保留用户进程，不读取终端日志。完整原始测试和响应证据在工作区，临时二进制不提交；无模型或外部服务请求。
- 校验封装边界：Content-Type、共同大小策略与JSON解码可在规则和错误契约一致时复用辅助函数或适当路由组中间件；框架请求大小限制是接收层防护，不等价于Handler内接收后的业务len检查。name/message等字段规则留在各自Handler或类型校验函数；绑定API可减少代码，但须核对解析语义与错误映射，不静默替换当前Unmarshal行为。当前已出现实际重复，可作为后续按需教学对象，本节未创建封装或重构用户核心。
- 理解验收：已说明当前chat读取、前置校验、解析、message业务检查和answer响应链；授课采用先分段写入、再完整对照，不再用无示例的独立任务替代教学。后续小变体仍在讲解后安排，不把本节测试通过等同于完整Agent能力。
- Git检查点：提交主题 `feat(http): 添加最小 chat 输入校验与规则回复`；仅本节chat_handler.go、chat_handler_test.go、main注册、计划和示例说明；教学方式调整与本节带写一致。.vscode/settings.json无变化，无关.DS_Store保留，不push，实际哈希以Git历史为准。
- 下一阶段：R.8.2引入处理函数边界，在出现真实处理调用/失败需求时展开；会话与模型仍按后续小节，不本轮提前布置。

#### R.8.2 消息处理函数与错误返回

- 状态：已掌握（当前普通函数、固定指令与预期输入错误教学范围）；学习者保存chat_logic.go并接入Handler，已说明指令/普通消息分支并确认结果与错误分工。用户源码完整包测试、race、vet、build与真实8091验收通过；不是模型或完整会话能力验收。
- 用户可见目标：之前POST /chat只回声；本节新增message中的小写 `/upper` 指令，后跟ASCII空格和文本时返回大写文本，仅指令或空白文本返回400；普通消息仍回复 `收到：` 加原消息，JSON请求/响应字段与其余端点不变。指令只是规则演示，不是模型选择工具或Agent推理。
- 本节只解决：新增实际消息处理分支和输入失败路径，用一个普通函数说明输入→处理→结果/错误→HTTP映射。处理函数不接收Hertz上下文、不返回HTTP状态码，不引入interface、依赖注入、服务层/新包、会话、模型或公共JSON封装。
- 调用链：Hertz路由→chatHandler（Content-Type/正文大小/JSON/非空message检查）→replyToMessage（同步文本处理，成功返回string与nil，失败返回空string与error）→Handler将已知指令输入错误映射400或编码answer为200；普通panic仍归原恢复器。
- 学习者核心代码：新建examples/hertz-from-scratch/chat_logic.go，package main，导入errors/strings，逐段完成replyToMessage(message string) (string, error)；其入参由Handler保证已去两端空白且非空。chat_handler.go保留前置检查，在message非空检查后调用处理函数，error非nil时使用固定格式 `%s\n` 和err.Error返回400并return，成功Answer使用answer。main.go和其他Handler不改；同package无需显式导入本地函数。
- 指令边界：message等于 `/upper` 或以 `/upper ` 开头才进入分支；TrimPrefix移除一个指令前缀，TrimSpace处理文本，空文本errors.New("text is required after /upper")，成功strings.ToUpper；非命令回复收到前缀。`/uppercase go`、`/UPPER go`、`/upper` 后直接接Tab不是指令。空白规则与请求空白预检的关系需解释，避免误把HasPrefix("/upper")作为完整命令边界。
- 关键Go语法：普通函数参数/多返回值、一一接收、nil表示error不存在、return只结束当前函数、调用方显式检查错误、error.Error取说明；包与文件不同，同目录package main共享包内函数，不按文件名导入。errors.New生成错误值不panic，缺少指令文本是预期输入错误，不交恢复中间件；固定错误文本无需fmt.Errorf，未来包装内部失败才按错误分类和日志边界展开，不能把所有error永远400或原文公开。
- 教学形式：先分段讲文件、用途、调用方、原因、替代影响，给每段“现在请写入”；最后提供chat_logic.go完整对照与chatHandler完整对照。本节一次讲完整，用户编写核心，助手随后补项目单测；不以孤立TODO替代教学。
- 最小验收：新增chat_logic_test.go直接调用replyToMessage，九场景覆盖普通、内部空格、英文/中文指令、裸指令/空白参数、/uppercase前缀、大小写和Tab边界；chat_handler_test.go新增六场景覆盖指令路由、精确answer键/值、400纯文本及旧响应。最小测试与完整Hertz包test、Chat/Reply race、vet、临时路径build通过，gofmt -l无输出；真实8091九请求核对普通、upper、中文trim、裸/blank指令、前缀、405、ping和greet。用户核心源码哈希前后相同，未控制用户服务进程；暂存前检查与可读格式验证按检查点执行。工作区只overlay项目chat_logic的故意错误副本，被TestReplyToMessage/different_prefix发现/uppercase误变CASE GO，原码未改。
- 教材预验证：工作区overlay教材副本通过完整Hertz包test、vet与build，处理函数九场景及指令路由六场景，隔离18091真实HTTP十二场景通过；故意去掉命令边界的副本被测试识别（/uppercase误变CASE GO），项目main/chat源码哈希未改变，隔离进程已停止。此证据仅证明教材可运行，不替代用户保存后的验收；工作区脚本、二进制、日志均不提交。
- 理解验收题：谁调用replyToMessage，为什么它不需要c；两个返回值分别是什么；处理函数return后为什么Handler还能继续；为什么缺少文本返回error而不是panic，为什么当前400映射不能直接套到未来模型故障。
- Git检查点：提交主题 `feat(http): 分离 chat 消息处理并添加 upper 指令`；仅chat_logic.go、chat_logic_test.go、chat_handler.go、chat_handler_test.go、计划与示例说明。.vscode/settings.json无变化，不强造diff；无关.DS_Store保留，不push、不提交预验证材料，实际哈希以Git历史为准。
- 下一步：先验收本节，再按下一用户可见目标决定最小会话输入/状态教学；公共JSON封装仅在明确消除重复且符合下一目标时展开，不预建多层框架。

#### R.8.3 最小会话计数：跨请求状态与互斥锁

- 状态：已掌握（当前进程内会话计数教学范围）；实现、测试、真实8091行为与共享store/Mutex临界区理解验收均通过。本节仅保存会话成功处理次数，不代表消息历史、模型记忆或完整生产会话能力。
- 用户可见目标：POST /chat可选session_id；去两端空白后非空时，成功响应在answer之外返回小写session_id与turn，同一ID成功处理次数从1递增，不同ID独立。不传、null或纯空白session_id按无会话处理，仍只有answer。错误请求不递增，计数表示服务端文本处理成功，不代表客户端收到响应。
- 本节只解决：让多次独立HTTP请求通过共同ID访问同一进程内状态；用普通map、Mutex和闭包说明状态归属、临界区和Handler构造。不是完整SessionManager，不保存消息/回答/个人资料、不做模型记忆、鉴权、随机ID生成、数据库、TTL、全局公共JSON封装或多层应用架构。
- 学习者核心代码：新建examples/hertz-from-scratch/session_store.go，sessionStore含mu sync.Mutex与turns map[string]int；newSessionStore返回已初始化map的指针；nextTurn使用指针接收者，Lock/defer Unlock包住读取、加一、写回和取本次结果。chat_handler.go新增SessionID请求字段、omitempty响应字段SessionID与Turn，将chatHandler改为newChatHandler(*sessionStore) app.HandlerFunc并返回原Handler闭包，在replyToMessage成功后按非空ID计数；main.go的newServer在注册/chat前创建一个store，再注册newChatHandler(sessions)。chat_logic.go不改。
- 生命周期与边界：每次newServer创建独立store，跨该server请求共享指针；禁止在请求函数里反复newSessionStore或通过值接收者复制Mutex。返回的闭包继续持有store，局部名字离开构造函数不等于对象失效；每请求局部req/message/answer/response不共享。锁只保护map短操作，不把文本处理、JSON写入、未来网络调用放入锁。并发turn保证不丢失/重复，不保证响应抵达顺序，也不是对整个HTTP会话排队或业务回滚。
- 关键Go语法：map[string]int的键值类型、make初始化、缺失键读0、map元素++；结构体&取指针、构造函数是普通自定义函数、指针方法接收者、Mutex零值可用和不可复制；func返回函数/闭包捕获、外层return交Handler给路由而内层return结束当前请求；omitempty在当前编码语义下省略空字符串/数字0，同package无需按文件import。
- 协议边界：session_id只作为客户端提供的教学分组键，不是身份凭证，知道同ID可影响同一计数；服务仅本机教学，不公开部署。新进程重启计数清空，不同进程不共享；无会话删除/过期/容量控制，新增不同ID会增长内存。不在本轮升级为生产会话或声称有模型历史。
- 教材预验证：工作区overlay通过完整Hertz包test、Session/SessionHandler race、vet和build；处理层并发64次与Handler并发32次结果不重复，新server重置、无会话旧契约、输入失败不计数和session_id类型错均检查。隔离18091八种串行请求及32次网络并发计数通过；每次请求新建store的反向副本被会话测试发现。源码hash未变，隔离进程已停止，工作区脚本/日志/二进制不提交；不替代用户实现后的验收。
- 教学方式：分段讲store状态/初始化、指针方法/锁、响应字段、Handler闭包与server注册；每段给“现在请写入”和完整文件/函数对照，一次讲完整，由用户写核心。先解释每个抽象消除的共享生命周期或并发问题，不只说框架需要。
- 代码与运行验收：新增session_store_test.go验证独立ID/独立store与64次并发递增，chat_handler_test.go新增17个串行场景、不同server隔离与32次并发请求；覆盖无/空/blank/null ID仅answer、精确小写answer/session_id/turn、ID去空白及大小写、原各类输入错误和指令失败不计数。gofmt检查、完整Hertz包test、Session/Handler race、vet与工作区build通过。真实8091十三种请求核对同ID1/2、指令/空白失败后3、另一ID1、无/blank/null ID、类型错400、GET405、ping与greet回归；另32次网络并发取得不重复的1—32。使用独立测试分组，不操作用户已有会话或服务进程；用户三个核心文件哈希前后不变。只overlay项目main的每请求新建store错误副本，被TestChatSessionServerIsolation发现第二次仍1，原码未改；原始证据、临时脚本/二进制留工作区不提交。
- 理解验收：已确认每次请求重新创建store会因map重置而始终返回turn=1；服务可并发处理请求，即使业务未主动启动goroutine，共享计数仍需同一把Mutex保护读取、加一、写回与取结果，防止数据竞争和丢失更新。锁保证计数操作完整，不是session_id唯一性或请求到达顺序；无ID保持仅answer，错误请求不计数。
- Git检查点：提交主题 `feat(http): 添加进程内会话计数与并发保护`；仅本节session_store.go、session_store_test.go、chat_handler.go、chat_handler_test.go、main.go、计划与示例说明；.vscode/settings.json照例核对且当前无diff，不强造修改；不push，无关.DS_Store保留，工作区验证脚本/二进制/日志不提交，实际哈希以Git历史为准。
- 下一步：先验收本节，再按下一可见目标展开最小历史或流式聊天对照，不在本轮同时布置后续核心实现。

### R.9 最小流式聊天（分阶段）

#### R.9.1 POST /chat/stream：输入校验后逐段发送规则回答

- 状态：已掌握（当前规则回答分段传输教学范围）；实现、测试、真实8091行为与输入错误/SSE边界、done/Handler收尾理解验收通过。本节不是模型逐token生成或完整Agent流验收。
- 用户可见目标：新增POST /chat/stream，使用与/chat相同的message和可选session_id，先完成当前规则回答，再按Unicode码点发delta事件，每个data为仅content字段的JSON字符串；首条立即、后续由200ms Ticker等待，最后done事件data为完整chatResponse。输入如/upper go得到G、O两个delta和完整GO的done。这是已生成字符串的分段传输模拟，不是模型token流或完整Agent。
- 本节只解决：把已学JSON输入、普通错误、replyToMessage、会话计数与SSE输出连起来；保持/chat、/events及中间件行为，不加模型、历史、浏览器页面、channel、新goroutine、流式Reader/EOF、通用绑定或公共JSON封装。
- 学习者核心代码：新建examples/hertz-from-scratch/chat_stream_handler.go，chatDelta含Content string与小写json:content；newChatStreamHandler(*sessionStore) app.HandlerFunc返回请求闭包。main的newServer在现有/chat注册后新增POST /chat/stream并传同一个sessions，不创建第二store；旧Handler与store/处理函数不改。
- 输入与状态边界：Content-Type415、接收后len超过4096返回413、Unmarshal400、TrimSpace message非空400、replyToMessage缺指令文本400，全部在sse.NewWriter之前处理；session_id去空白，无ID不计数。规则处理成功后调用nextTurn一次，含ID的done与/chat共用序列；之后客户端取消或写入失败不会回滚，这是服务端处理次数不是送达次数。
- SSE协议：sse.NewWriter切换写入模式并设置text/event-stream/no-cache；for range []rune(answer)逐码点，避免按UTF8字节拆开中文，不宣称按用户可见字符簇或模型token。json.Marshal(chatDelta{Content:string(ch)})负责字段/转义，WriteEvent空id、delta/data负责SSE外层event/data/空行并刷新。循环后ctx.Err检查、Marshal完整response、WriteEvent done；Handler返回后Hertz收尾，不defer Writer.Close，不调用c.JSON/c.String追加第二种响应。流式取消/编码/写入失败日志+return且不发done，正常EOF无done不能作业务完成凭证；原HTTP1恢复器已支持流式panic中断，本节不用修改recovery。
- 时间与取消：200ms Ticker仅用于观察分段，首条不等，后续select在Done或Ticker可读时结束等待，再检查Err；defer Stop不结束Handler，循环/return才结束。等待检查与WriteEvent错误互补，开启的WithSenseClientDisconnection仍沿用。长字符串按码点数延长教学发送时长，不作为生产吞吐方案。
- 教学：分段讲输入错误/SSE边界、响应计数、rune与string转换、Ticker/select、JSON内层与SSE外层、done与框架收尾、共用store注册；每段有现在请写入与位置，最后完整文件和newServer对照。说明每个log前缀归属与%v，不引入无必要错误包装，不用无代码TODO代替教学。
- 教材预验证：工作区overlay完整Hertz包test、StreamLesson race、vet/build通过；精确中文delta/JSON转义/done完整内容、无session done只answer、预取消与首条取消、Write/Flush失败不追加done、输入普通错误及GET405均检查。隔离18091四delta实际约0/0.2/0.4/0.6秒到达、无Content-Length、chunked、共享/chat计数1→stream2→chat3通过；curl提前0.1秒退出只首delta、ctx取消日志和计数不回滚通过。误改byte切分的工作区副本被中文格式测试发现。首次测试夹具使用不存在RequestHeader.SetContentType导致编译失败，已仅在工作区修为Header.Set后重跑通过；用户源码哈希不变，自建进程已停止，预验材料不提交。
- 代码与运行验收：新增chat_stream_handler_test.go，精确核对中文两个delta与完整done、每事件Write/Flush调用、Handler不Finalize；六个内容/无会话场景覆盖回声、blank/null ID、JSON转义、组合码点和4096字节；十七个输入错误场景在直接Handler和路由测试中核对普通400/413/415且未进入SSE模式，GET/HEAD405。七个取消/写入失败场景覆盖预取消、首delta/末delta取消、delta与done的Write/Flush失败，取消和delta失败不追加done、不改写普通500，规则处理计数不回滚；done Flush失败时字节可能已写出，不能声称客户端一定未见done。gofmt、最小测试、完整Hertz包test、Stream/Session race、vet与临时build通过。
- 真实入口验收：8091的/upper 你好Go四delta约0.001/0.202/0.402/0.602秒，done约0.602秒；text/event-stream/no-cache、chunked、无Content-Length、干净收尾、content小写键与完整answer核对通过；普通/chat→stream→chat共用同ID计数1/2/3。另无会话、指令/JSON/空白错误、媒体415、4097字节413、GET405、ping/hello回归通过；curl0.1秒退出28只见首delta、无done且已增加计数不回滚。仅overlay当前项目main监听地址的隔离18091进程确认ctx取消日志，已停止自建进程，不控制用户8091；用户核心文件hash未变。只overlay项目stream错误副本按byte拆中文，被正式格式测试发现。完整证据与临时材料留工作区不提交。
- 理解验收：已确认输入校验必须在进入SSE模式前完成，以便错误仍返回普通HTTP400/413/415；响应字节已发送后不能改成另一份普通错误响应。完整answer先算好，循环发送JSON delta，done只是应用事件名称；正常发送done后因Handler逻辑结束而返回，由Hertz收尾本次响应，不等于必然关闭底层TCP连接。客户端断开由当前断开感知关联ctx取消，代码通过Done/Err或WriteEvent错误显式return，不继续发送delta/done；WithSenseClientDisconnection不负责正常done收尾。共享turn在规则处理成功后增加，发送失败不回滚。
- Git检查点：提交主题 `feat(http): 添加规则回答的 SSE 流式聊天接口`；仅本节chat_stream_handler.go、chat_stream_handler_test.go、main接线、计划与示例说明；.vscode/settings.json照例核对且当前无diff，不强造改动；不push，无关.DS_Store保留，工作区脚本/日志/二进制不提交，实际哈希以Git历史为准。
- 下一步：先验收本节；只有接入真实异步或模型流输入时再就地展开channel/goroutine/Reader与EOF，当前不预建这些能力。

#### R.9.2 Go SSE 客户端：边读边显示与完成判定

- 状态：已掌握（当前单行JSON delta/done客户端教学范围）；学习者保存核心实现，正式测试、race、vet/build、真实go run与完成判定理解验收通过。不是通用SSE解析器或完整模型流客户端。
- 用户可见目标：不再只用curl看原始event/data行；独立Go程序向本机POST /chat/stream发送message与可选session_id，逐delta解码并直接输出文字，完整done.answer与累计内容核对，读取正文正常结束后显示完成及可选会话次数。HTTP400等普通错误显示状态和正文；没有done或中途读取失败不显示完成。
- 文件与职责：学习者新建examples/hertz-chat-client/main.go与client.go，复用根go.mod、不新建模块。main只定义-message/-session参数、10秒总时限HTTP Client、调用sendChatStream与退出码；client.go定义协议结构体、sendChatStream和readChatStream。不同目录package main是不同包，客户端仅按HTTP契约定义DTO，不导入服务端main或生产internal。当前server Handler/路由/会话/依赖保持不变。
- 发送链：json.Marshal请求结构体→bytes.NewReader→http.NewRequest POST→Content-Type application/json、Accept text/event-stream→Client.Do→defer Body.Close。Do得到响应头不等正文读完；非2xx不会自动成为Do error，先主动检查StatusCode，非200只读取最多4096字节错误正文片段；200解析Content-Type须为text/event-stream，再调用readChatStream。不是把整个SSE ReadAll后才显示。
- 读取核心：bufio.NewScanner(body)默认按行，临时eventType/eventData记录当前事件，在空行时才分派delta/done；data为JSON，Unmarshal解码后用strings.Builder累计及fmt.Fprint(out)立即输出。done保存完整结果并与累计文本核对，标记done=true后继续读至正文EOF，拒绝done后的额外事件。扫描停止先检查scanner.Err（EOF被Scanner隐藏为nil，传输/超限错误仍非nil），再检查未结束事件与未见done；EOF只代表输入结束，不代表业务完成或TCP断开。
- 输出与生命周期：readChatStream返回streamResult/error；sendChatStream核对done的session_id为请求去两端空白后的ID、计数有效性后才打印完成。失败时可保留已输出的部分回答，但不补成功提示；Body.Close在sendChatStream返回前执行，main收到error后打印stderr并Exit(1)不跳过该清理。默认10秒时限包含连接与整个正文读取，只用于短消息教学，长流需重新定义策略；不新增重试或后台goroutine。
- 关键Go语法：flag.String返回*string、Parse后解引用；io.Reader/io.Writer接口分别表示读入/写出能力，HTTP Body、strings.Reader、bytes.Buffer及os.Stdout可按能力作为参数；strings.Builder.WriteString是内存累计，不等网络写出。fmt.Errorf前缀标具体失败步骤，%w保留已有错误链、%d用于HTTP数值状态、%s插文本、%q用于引号与转义；没有底层错误对象的校验用errors.New或格式化消息，不乱用%w。switch/continue与多个赋值就地解释，核心函数分段后给完整对照。
- 简版解析边界：只消费当前本机服务的event:空格、单行data:空格、JSON delta/done格式；支持LF/CRLF，不是完整SSE规范实现。未实现多data行、id/retry/注释/BOM/重连与通用字段策略；Scanner默认单行大小限制仍保留，超长行报错。10秒总时限不是总体内存配额，累计回答无独立总量限；不公开作为生产SDK或任意服务通用解析器。
- 教材预验证：工作区main/client/test副本test、race、vet/build通过；12个解析场景覆盖正常/LF与CRLF/空行、EOF无done/未结束事件、坏JSON/空delta/空done/内容不符、done后事件、未知字段；额外验证读取/输出错误%w保持错误链，5个HTTP场景包括400/错误媒体/会话不符/非法turn，管道测试证明EOF前已输出delta。Go客户端实际连接8091，短消息逐段输出首段至末段约相差0.8秒，同会话连续两次1/2，/upper缺参显示HTTP400并exit1；移除done必需检查的工作区反向副本被无done测试发现。教材副本/测试/临时二进制与证据不提交；用户服务端源码未改、服务进程不操作。
- 代码与运行验收：助手新增正式examples/hertz-chat-client/client_test.go，覆盖LF/CRLF/JSON转义、逐字节Reader跨UTF-8和行边界、空流/缺done/半帧/错误JSON/空content/回答不符/额外事件/未知行及Scanner超长行；HTTP请求方法、头和JSON字段、session去空白/无ID、400/500、错误媒体类型、非法turn及4096字节错误正文片段；读取/输出/transport错误包装、成功及失败路径Body.Close。测试专用io.Pipe控制delta先到、done已到但正文未结束，验证即时显示及继续读到EOF，不给业务新增goroutine。正式包test、race、vet与工作区路径build通过；当前实际工具链go1.26.0 darwin/arm64，不修改go.mod或依赖。
- 真实入口：以go run ./examples/hertz-chat-client连接现有8091，/upper 你好 Go逐段显示，首delta到末delta间隔约0.8秒，stdout最终为回答及完成、stderr空、exit0；独立测试会话连续两次turn1/2。/upper缺参和空白message均HTTP400、stdout空、不显示完成、exit1，go run另打印exit status 1。只overlay当前项目client.go的错误副本删除必需done检查，正式无done测试发现error为nil而失败；用户客户端与服务端关键文件hash前后未变，原始证据/临时二进制/反向副本只留工作区，未操作用户服务进程。
- 理解验收：确认event名称由服务端WriteEvent第二参数给出，经event行保存到eventType再由switch匹配；Scanner创建不发请求，Scan按需读取Body并按行拆分，Text取最近一行；Builder仅在内存累计，fmt.Fprint(out)才写到本次传入的os.Stdout。完成判定要求合法done与正文正常读取结束：EOF表明正文已结束，不保证完整回答已交付，缺done返回SSE ended without done，不显示完成。保留WriteString而去掉Fprint仍累计但不显示delta。
- Git检查点：提交主题 `feat(http): 添加 Go SSE 聊天客户端`；范围仅用户main.go/client.go、client_test.go、根计划与示例说明；.vscode/settings.json已核对无diff，不强造修改；不push、不提交临时材料，实际哈希以Git历史为准。
- 下一步：R.10按简化Workflow目标分小节，先离线两节点Workflow验证类型与调用链，再接POST /tech-proposal；不把规则输出冒充模型生成，不预建channel、重试或生产封装。

### R.10 简化技术方案 Workflow（分阶段）

#### R.10.1 离线两节点：整理任务 → 固定练习方案

- 状态：已掌握（当前离线固定两节点编排教学范围）；用户核心、正式测试、实际CLI、类型/构建与执行理解验收通过。未接模型或HTTP，不代表完整技术方案生成能力。
- 用户可见目标：go run接收-task，在终端显示去空白后的任务与固定三步练习方案；空任务报错并非零退出。输出明确标注规则模板（非模型生成），不声称完成真实技术分析或自动执行任务。
- 文件与范围：新建examples/workflow-from-scratch/workflow.go及main.go，复用根go.mod与当前Eino v0.9.19。workflow.go包含proposalRequest、taskAnalysis、proposalResult、analyzeTask、buildProposal与newProposalWorkflow；main仅解析参数、创建上下文、构建与Invoke后输出。现有HTTP/SSE、生产internal/techproposal及依赖不改，不读项目证据、访问模型或新增业务goroutine。
- 调用链：START(proposalRequest)→analyze_task(analyzeTask，输出taskAnalysis)→build_proposal(buildProposal，输出proposalResult)→END。普通函数可直接调用，Eino封装仅用于对照现有Workflow的类型与编排边界，不为了两个固定步骤声称必须上框架；不预建节点接口、服务层或配置层。
- 核心教学：先定义输入/中间/输出结构，再逐段写两个普通节点；ctx.Err在节点入口检查协作取消，TrimSpace与空任务错误；Builder和Fprintf(&content)仅内存构建，main的fmt.Print才写终端。newProposalWorkflow用NewWorkflow[I,O]声明总体输入输出、InvokableLambda适配普通函数、AddLambdaNode指定唯一节点键、AddInput指定上游并传整个输出、End接最终结果；最后Compile构建Runnable，Invoke才执行一次任务。强调Eino Compile不是Go编译器，也不替代节点业务校验。
- 语法与错误：方括号为泛型类型参数，不是切片内容；InvokableLambda从函数签名推断I/O，传函数名不是立即调用；节点名字符串与函数标识符不同。AddLambdaNode返回WorkflowNode，链式AddInput等价于分两句；Runnable[I,O]为执行接口，构建失败返回nil/error，节点失败返回空结构/error。fmt.Errorf的compile/analyze/build前缀表示失败阶段，%w保留底层错误，固定输入错误用errors.New；Fprintf的%s插入文本、%d编号，&content为Builder指针，替代直接函数串联和普通字符串累积就地讲明影响。
- 教材预验证：工作区main/workflow/test副本test、race、vet/build及go run通过；检查精确规则输出、去空白、空任务、直接节点/整体取消、构建阶段零节点调用、Invoke顺序及输入错误停止下游。错误接线将build_proposal直接接START，CLI在初始化时报proposalRequest与taskAnalysis类型不匹配，非业务执行错误；原教材未改，临时文件/二进制不提交。
- 代码与运行验收：用户修正main初始化错误检查前多余Invoke，入口先检查构建结果、只调用一次Invoke；正式workflow_test.go覆盖五种任务整理、输入步骤真实传递到文本节点、同Runnable多次结果独立、节点与整体取消/DeadlineExceeded错误链、Compile零节点执行、Invoke顺序及输入错误不执行下游、接线类型错在构建阶段发现。gofmt、实际包test/race/vet及工作区build通过。真实go run默认/去空白任务精确规则文本、空任务stderr错误且stdout空/exit1；仅工作区overlay记录节点入口，确认analyze_task/build_proposal各一次；构建错误及强制初始化失败先错误退出不调用nil Runnable。重加Invoke的错误副本虽stdout仍正常，入口观测发现每节点执行两次。用户main/workflow与go.mod/go.sum哈希前后不变，未修改核心或依赖、未请求网络模型，临时证据/二进制/overlay不提交。
- 理解验收：确认Compile只构建接线，Invoke才处理本次任务；build_proposal通过AddInput(analyze_task)接收analyzeTask产生的taskAnalysis，不是直接拿入口proposalRequest。stdout正常结果与stderr故障说明分开，Fprintln第一参数指定输出目标；写stderr不会自动结束程序，os.Exit(1)才退出。不把可编译视为重复调用正确，也不要求机械背框架写法。
- Git检查点：提交主题 `feat(workflow): 添加离线两节点 Eino 编排示例`；仅本节main.go、workflow.go、workflow_test.go、根计划和examples/README.md；.vscode/settings.json已核对无diff、不强造修改，不push，不混入临时材料，实际哈希以Git历史为准。
- 下一步：R.10.2把同一已验证Runnable接到最小POST /tech-proposal，届时展开构建一次与请求多次Invoke的生命周期；当前不同时布置HTTP实现。

#### R.10.2 同一Workflow的HTTP入口：构建一次，每请求Invoke

- 状态：已掌握（当前固定Workflow的HTTP适配教学范围）；用户核心、正式测试、race、vet/build、真实8092与CLI回归及请求上下文理解验收通过。未接真实模型或SSE，不代表完整生产Workflow故障策略。
- 用户可见目标：保留go run ./examples/workflow-from-scratch -task的离线输出；增加-serve在127.0.0.1:8092提供POST /tech-proposal，JSON task输入，成功仅小写result字段，其值是原固定规则方案；普通输入错误返回纯文本415/413/400，不调用Workflow。输出仍标明非模型生成，非SSE/Agent自动规划。
- 改动范围：workflow.go/workflow_test.go及原8091HTTP/SSE、生产internal均保持；http.go只定义proposalHTTPRequest(Task json:task)、proposalHTTPResponse(Result json:result)、newProposalHandler(Runnable)闭包与newProposalServer。main增加flag.Bool serve，在已构建Runnable后分支Spin/return，CLI原Invoke保留。复用当前根模块与Eino/Hertz，不新建模块、不搬代码到生产、不加公共JSON层或重试。
- 生命周期与类型：main在模式分支前newProposalWorkflow一次，构建错误先退出；HTTP注册newProposalHandler(workflow)仅生成并登记函数，不执行任务。请求进入闭包，局部req/task/result独立，捕获的Runnable沿服务生命周期复用；Invoke(ctx,proposalRequest{Task:task})使用当前请求context而非启动Background。本节节点无共享可变业务状态，无需会话Mutex，不把所有未来Runnable组件都断言为自动并发安全。
- HTTP检查：ParseMediaType→Hertz接收后的body长度4096字节上限→Unmarshal→TrimSpace task非空→Invoke→c.JSON。错误依次为Content-Type must be application/json换行、request body too large换行、invalid JSON request换行、task is required换行；未知/重复字段保留Unmarshal默认行为。HTTP结构体与Workflow内部类型显式转换，只解释传输边界，不提前创建应用接口；节点空任务检查保留保护CLI/直接调用，Handler预检承担准确400映射，不通过err.Error文本判断业务错误。
- 失败归属：已通过输入校验后的Workflow error记录proposal workflow failed前缀和%v，当前返回通用500 workflow execution failed换行，随后return；不在Handler调用os.Exit或把所有内部error写400，不公开内部节点诊断。取消/时限详细HTTP策略后续按实失败展开，当前原样传ctx、可合作返回错误，不声称生产完整故障分类。c.JSON直接接Go对象，Content与result字段映射，不把预Marshal字节再次JSON编码。
- 服务构造：独立8092使用server.Default自带recovery，避免重复复制8091自定义中间件；此前server.New显示自写恢复器，本节明确Default=New再Use内置recovery，普通error分支仍由Handler处理，panic兜底不保证同样正文。WithHandleMethodNotAllowed(true)保留方法分发，WithSenseClientDisconnection(true)用于当前Mac/netpoll请求取消通知，与是否SSE无关；不保证Windows或其他运输实现相同。Spin运行到停止后return避免落入CLI，再执行一次任务；serve参数为*bool需解引用。
- 教材预验证：工作区main/http与当前workflow原码/旧测试组合，HTTP与旧Workflow完整test/race/vet/build通过；13项输入错误与方法/未知路由未Invoke，6项成功包含去空白/转义/默认未知重复字段/4096边界，响应恰有result键，Invoke每请求一次；请求ctx传递与500通用正文/服务端日志，8项并发请求各自任务无串值。真实临时8092三成功、415/413/400、GET405、未知404 Not Found及8并发任务通过，CLI原行为保留，临时实例已停止、用户8091未控制。移除HTTP task预检的工作区错误副本被测试发现空task走到Invoke并500；源码/模块哈希不变，临时测试/日志/脚本/二进制不提交。
- 代码与运行验收：助手新增正式http_test.go，十五项输入校验与方法/未知路径核对未Invoke，六项成功核对精确小写result键、输入去空白/转义、Unmarshal未知重复字段默认行为与4096边界；注册不执行任务、服务选项、每请求一次Invoke、本次ctx原样传入、取消上下文错误、内部失败精确通用500与日志、下一请求仍成功，八项并发结果独立。实际包旧Workflow+HTTP test/race/vet/build及gofmt通过。原go run正常和空任务CLI回归；现有用户8092仅请求不控制进程，三成功、415/413/400、4096边界、GET405、未知404、八并发网络请求通过。只overlay项目文件的隔离18092人为节点错误返回通用500、后续200，诊断仅服务端日志；自建进程已停止。删HTTP task预检的工作区反向副本被正式测试发现空task误入Invoke/500，原码重跑通过；main/http/workflow与模块hash不变，临时文件不提交。
- 理解验收：确认main启动时newProposalWorkflow/Compile一次，两个正常HTTP请求各Invoke一次；闭包持有已构建Runnable，本次req/task/result独立。Handler中的ctx由Hertz随本次请求传入，不是main构建时的Background；Invoke将本次取消关系传给框架与节点，节点通过Err或支持取消的下游协作退出。当前Mac/netpoll断开感知启用后客户端断开可取消请求ctx，ctx不是强杀器或整个服务开关；使用新的Background会切断本次请求关系，而从请求ctx派生子上下文可以保留关系。
- Git检查点：提交主题 `feat(workflow): 添加技术方案 Workflow HTTP 入口`；仅main.go、http.go、http_test.go、根计划与examples/README.md；.vscode/settings.json已核对无diff，不强造修改，不push、不提交临时证据或生产文件，实际哈希以Git历史为准。
- 下一步：本节收尾后按根路线和具体缺口选择下一小节，不立即加入模型或完整生产工作流。

### R.11 指标入门（分阶段，先在教学服务验证）

#### R.11.1 按响应状态计数与 GET /metrics

- 状态：已掌握（当前请求计数教学范围）；用户实现、正式测试、race、vet/build、隔离真实HTTP与CLI回归、指标分组与累加职责理解验收通过。不代表监控平台、Histogram或生产可观测性已完成。
- 用户可见目标：8092的GET /metrics返回Prometheus文本格式；POST /tech-proposal处理后按status分组累加，先发正常和空任务请求即可看到200/400计数；重复查询metrics不加业务计数，不再只靠单条日志猜总体请求情况。
- 范围：新增examples/workflow-from-scratch/metrics.go，含requestMetrics(registry/CounterVec)、newRequestMetrics、middleware与handler；http.go仅改newProposalServer，创建一份metrics，server.New后Use(metrics.middleware,recovery.Recovery())，保留原POST并增加GET metrics。main/Workflow/业务Handler不改，原CLI保持；本节不加Histogram/PromQL/监控平台/模型，不改生产入口，R.11整体及旧I.6尚未完成。
- 依赖：固定github.com/prometheus/client_golang v1.23.2与github.com/prometheus/common v0.66.1，分别使用prometheus和expfmt包；根go.mod按实际导入归为直接依赖，其余相关库保留间接依赖。既有版本和go1.25.11声明未升级，未宽范围tidy旧依赖。下载使用命令级代理并保留GOSUMDB校验，未修改全局环境。
- 指标与生命周期：NewRegistry是每服务独立的空收集注册表，NewCounterVec定义agenthub_proposal_requests_total和status单标签，MustRegister只在新Registry登记固定合法指标，配置错误panic而非请求输入错误。创建CounterVec不等创建每个标签组合；第一次WithLabelValues访问对应组合时创建Counter，未Inc可保持0，Inc才加一。当前中间件在选择后立即Inc；无请求且未预初始化时metrics正文可为空。registry和requests被同一requestMetrics指针持有，服务构造一次，不每请求重置。
- 请求计数：middleware指针接收者，经c.Next同步执行下游，返回后用c.Response.StatusCode/strconv.Itoa取最终普通响应状态，再WithLabelValues(status).Inc。方法必须POST且FullPath等于已匹配模板/tech-proposal；metrics查询、未知路由及405不在本节范围。标签只有有限状态码，不用task/session/query/raw URL等无界值。Counter是进程内累计增量，不持久化，重启清空；计数表示响应被Handler处理，不代表送达/Workflow成功/实时在途请求或跨进程总量。
- 恢复顺序：此前Default隐式先登记recovery，本节切New并显式Use(metrics,recovery)，保留同一内置恢复器、让其在计数内层；panic由下游恢复并设置500后返回，计数再读状态，不在异常展开时提前记200。指标方法自身panic不在内层恢复保证内，固定标签避免数量错；仅当前普通响应，不套SSE/已发送最终状态语义。不额外复制8091自定义恢复器。
- 输出：handler Gather采集注册指标族，出错log+通用500；bytes.Buffer在内存接收expfmt.MetricFamilyToText编码，全部成功后使用expfmt.NewFormat(TypeTextPlain)给Content-Type，c.Data200写字节。官方编码器负责HELP/TYPE/标签转义，不手拼指标、不JSON包字符串、不在每个family后写一份响应。单一固定text0.0.4，不支持OpenMetrics协商、额外Go/process collectors或SDK完整抓取策略；Gather编码失败不返回部分成功指标。
- Go就地解释：结构体指针与接收者、m.middleware/m.handler方法值捕获同一对象；CounterVec是按标签组合的一组Counter而非单个整数；strconv.Itoa是整数转十进制字符串而非string(int)；Inc调用累加、不启动后台任务。MustRegister与Register(error)替代及作用范围，Gather结果切片/空切片、_忽略写入字节数但检查error，Buffer与之前Builder相同的内存/终端区别、c.Data与c.JSON/c.String区别；错误日志前缀和%v用途就地展开。
- 教材预验：工作区main/http/workflow/metrics与原测试在modfile副本中test/race/vet/build通过，旧HTTP输入与CLI回归、官方文本解析器核对计数、重复查询/路由排除、独立server注册表、Workflow500、普通panic恢复后的500、32并发累加与8并发抓取、故障Collector触发Gather500均通过。隔离18093真实文本响应正常，起始空；两成功/两400/一413/一415计数准确，重复抓取和非目标方法路径不变，16并发成功后200计数18；原CLI正常，自建进程已停止，用户8092/8091未控制。工作区提前读取状态码的错误副本被测试发现六请求全记200，原教材未改；核心哈希保持不变，临时材料不提交。
- 代码与运行验收：正式metrics_test.go覆盖初始无序列、六响应精确计数、查询及405/404排除、两个server独立；单独选择标签得到0、Inc后累加且Gather不重置；Workflow500后下一请求200、普通缓冲响应panic恢复500后计数、32并发请求与8并发抓取、故障Collector的Gather通用500/日志。实际包完整test/race/vet/build和gofmt检查通过。原go run CLI成功/空任务行为保留；仅overlay项目http端口的隔离18111真实HTTP，六请求200=2/400=2/413=1/415=1，重复查询与错误方法路径不变，16并发成功后200=18；自建实例已停止，用户8092/8091未控制。提前读取状态码的错误副本被正式测试发现六请求误记200=6；原用户核心hash不变，临时脚本/日志/二进制/overlay不提交。
- 理解验收：区分创建/注册、按标签选择、Inc加一、Gather收集和expfmt编码；指标不是HTTP业务必需或自动上传，Prometheus客户端库不等监控服务器。标签名与值数量/顺序对应，不同标签组合属于同名指标族，families是族切片而非请求明细。Next返回只表示下游执行返回，不保证客户端收到；外层计数等内层恢复后读取普通500，错误响应也计数。当前只有POST，未增加无区分价值的method标签。
- Git检查点：提交主题 `feat(metrics): 添加 Workflow HTTP 请求计数`；仅metrics.go、metrics_test.go、http.go、go.mod/go.sum、根计划与examples/README.md；.vscode/settings.json已核对无diff、不强造修改，不push，提交哈希以Git历史为准。
- 下一步：R.11.2在同一指标对象增加处理耗时Histogram，先解释Observe与累计bucket；不同时引入监控平台或模型。
