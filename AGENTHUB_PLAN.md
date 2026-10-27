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
- 已掌握（二轮）：**R.1**
- 下一小节：**R.2 JSON 与错误映射——把请求 JSON 转成 Go 数据，再构造 JSON 响应。**

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
