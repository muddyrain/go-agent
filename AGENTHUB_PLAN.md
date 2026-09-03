# AgentHub 学习路线

## 1. 项目目标

从零实现一个可部署的企业级 AI Agent 平台，逐步具备：

- 可扩展的 Agent Runtime
- 内置工具、自定义工具与 MCP 工具
- 会话与配置持久化
- HTTP API 与流式响应
- RAG 知识库
- 多 Agent 与工作流编排
- 日志、指标、链路追踪、测试和部署能力

## 2. 推进规则

每节课按以下闭环推进：

1. 明确本节目标和验收标准。
2. 讲解设计与当前需要的 Go 知识。
3. 学习者完成核心实现。
4. 运行最小有效验证。
5. 更新本计划的课程状态和验收记录。
6. 使用 `.agents/skills/agenthub-learning-checkpoint/SKILL.md` 检查并自动提交本节变更。

状态说明：`⬜ 待开始`、`🔄 进行中`、`✅ 已完成`、`⛔ 阻塞`。

## 3. 课程路线

### Phase 0：工程基础

| 课程 | 内容 | 交付物 | 状态 |
|---|---|---|---|
| 0.1 | 项目结构与 Go Module | 目录骨架、`go.mod`、可运行入口 | ✅ 已完成 |
| 0.2 | 配置管理 | YAML 配置、环境变量覆盖、配置加载器 | ✅ 已完成 |
| 0.3 | 结构化日志 | 统一日志组件、日志级别与字段 | ✅ 已完成 |
| 0.4 | 错误处理 | 错误包装、错误类型、错误码 | ✅ 已完成 |
| 0.5 | 工程命令 | Makefile、格式化、检查、构建、运行 | ✅ 已完成 |
| 0.6 | 测试基础 | 单元测试、表驱动测试、覆盖率 | ✅ 已完成 |

### Phase 1：Agent Runtime

| 课程 | 内容 | 交付物 | 状态 |
|---|---|---|---|
| 1.1 | Model 与 Message 抽象 | 模型接口和消息模型 | ✅ 已完成 |
| 1.2 | Tool 系统 | Tool 接口、Registry、参数校验 | ✅ 已完成 |
| 1.3 | Agent Loop | 模型—工具—结果闭环 | ✅ 已完成 |
| 1.4 | Memory | Token 预算、滑动窗口、摘要压缩 | ✅ 已完成 |
| 1.5 | Streaming | 统一同步/流式接口 | ✅ 已完成 |
| 1.6 | Agent Factory | 配置化创建不同 Agent | ✅ 已完成 |
| 1.7 | MCP | 多 Server 管理、超时、重连与工具适配 | ⬜ 待开始 |

### Phase 2：持久化

| 课程 | 内容 | 交付物 | 状态 |
|---|---|---|---|
| 2.1 | PostgreSQL | 连接池和健康检查 | ⬜ 待开始 |
| 2.2 | 数据库迁移 | 可版本化 Schema | ⬜ 待开始 |
| 2.3 | 数据访问层 | 类型安全查询与 Repository | ⬜ 待开始 |
| 2.4 | 会话持久化 | Conversation 与 Message 存储 | ⬜ 待开始 |
| 2.5 | Agent 配置存储 | Agent CRUD | ⬜ 待开始 |
| 2.6 | Redis | 缓存、会话状态与限流基础 | ⬜ 待开始 |

### Phase 3：API 服务

| 课程 | 内容 | 交付物 | 状态 |
|---|---|---|---|
| 3.1 | HTTP 服务 | 路由、参数绑定、健康检查 | ⬜ 待开始 |
| 3.2 | 中间件 | 请求日志、恢复、请求 ID、认证 | ⬜ 待开始 |
| 3.3 | REST API | Agent、会话、消息和工具接口 | ⬜ 待开始 |
| 3.4 | SSE | 流式对话接口 | ⬜ 待开始 |
| 3.5 | API 错误规范 | 统一响应和错误映射 | ⬜ 待开始 |
| 3.6 | OpenAPI | 可交互接口文档 | ⬜ 待开始 |

### Phase 4：RAG

| 课程 | 内容 | 交付物 | 状态 |
|---|---|---|---|
| 4.1 | Embedding | 文本向量化接口 | ⬜ 待开始 |
| 4.2 | pgvector | 向量存储与相似度查询 | ⬜ 待开始 |
| 4.3 | 文档处理 | 解析、切分、元数据 | ⬜ 待开始 |
| 4.4 | 检索 | 向量/关键词混合检索与重排 | ⬜ 待开始 |
| 4.5 | RAG Agent | 检索增强问答闭环 | ⬜ 待开始 |
| 4.6 | 知识库 API | 上传、索引、查询和删除 | ⬜ 待开始 |

### Phase 5：Workflow 与 Multi-Agent

| 课程 | 内容 | 交付物 | 状态 |
|---|---|---|---|
| 5.1 | Router | 请求分类与 Agent 路由 | ⬜ 待开始 |
| 5.2 | Orchestrator-Worker | 任务拆分、执行与聚合 | ⬜ 待开始 |
| 5.3 | Workflow | 节点、边、状态与条件分支 | ⬜ 待开始 |
| 5.4 | Human-in-the-loop | 暂停、确认和恢复 | ⬜ 待开始 |
| 5.5 | 并行执行 | 并发调度与结果聚合 | ⬜ 待开始 |
| 5.6 | 协作状态 | Agent 间消息与共享上下文 | ⬜ 待开始 |

### Phase 6：生产化

| 课程 | 内容 | 交付物 | 状态 |
|---|---|---|---|
| 6.1 | OpenTelemetry | 端到端 Trace | ⬜ 待开始 |
| 6.2 | Prometheus | QPS、延迟、错误率、Token 指标 | ⬜ 待开始 |
| 6.3 | 生命周期 | 健康检查与优雅退出 | ⬜ 待开始 |
| 6.4 | 稳定性 | 超时、重试、限流和熔断 | ⬜ 待开始 |
| 6.5 | 集成测试 | 数据库与 API 测试 | ⬜ 待开始 |
| 6.6 | 性能分析 | Benchmark 与 pprof | ⬜ 待开始 |

### Phase 7：交付与部署

| 课程 | 内容 | 交付物 | 状态 |
|---|---|---|---|
| 7.1 | Docker | 多阶段镜像 | ⬜ 待开始 |
| 7.2 | Compose | 应用、PostgreSQL、Redis 一键启动 | ⬜ 待开始 |
| 7.3 | 多环境配置 | 开发、测试、生产配置策略 | ⬜ 待开始 |
| 7.4 | CI | 格式化、测试、检查和构建流水线 | ⬜ 待开始 |

## 4. 当前进度

- 当前阶段：**Phase 1 Agent Runtime 已完成 1.1—1.6，准备进入 1.7 MCP**
- 当前课程：**1.7 MCP，待开始**
- 最近完成：**Agent Factory 的配置转换、Memory 策略选择、依赖注入与 Agent 组装**
- 下一验收目标：理解 MCP 协议和 Client/Server 边界，实现 MCP 工具发现与现有 Tool Registry 的适配

## 5. 验收记录

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
