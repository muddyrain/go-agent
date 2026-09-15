# AgentHub 学习路线

## 1. 学习目标

通过持续实现 AgentHub，学习企业级 Go 服务与 AI Agent 系统设计。

本项目最终仍会覆盖 Agent Runtime、工具与 MCP、持久化、HTTP/SSE、RAG、多 Agent、可观测性和部署，但课程不再按“先造完所有底层组件”推进，而改为：

> 先做出一个能运行、能观察、能解释的纵向功能，再围绕真实问题补充抽象、测试和生产能力。

最终产品形态、系统边界、Eino 采用策略和各阶段详细交付见 [`docs/product-blueprint.md`](docs/product-blueprint.md)，总览图见 [`docs/images/agenthub-product-roadmap.svg`](docs/images/agenthub-product-roadmap.svg)。最终目标不是自研 Eino，而是实现一个**基于 Go 与 Eino 的可部署 Agent 应用平台**。

## 2. 本次路线校准结论

### 2.1 当前真实状态

代码已经积累了较完整的 Runtime 零件：

- Model、Message 与流事件协议
- Tool、Registry、Executor 与 JSON Schema 校验
- Agent Loop、Memory、Streaming 与 Agent Factory
- MCP Session、Tool Adapter、多 Server Manager 与命名空间
- 配置、日志、错误处理、工程命令和大量单元测试

路线校准前，`go run ./cmd/agenthub` 只会加载配置、初始化日志并输出启动信息。现在经过 A.1—A.2，入口已经能运行一条演示 Model—Tool—Model 纵向链路：

```text
application starting app=AgentHub env=development address=127.0.0.1:8848
tool_call: id=call-weather-001 name=get_weather arguments={"city":"杭州"}
tool_result: id=call-weather-001 name=get_weather content=杭州今天晴，25°C。
assistant: 根据天气工具的查询结果：杭州今天晴，25°C。
steps: 2
usage: input=0 output=0 total=0
```

当前已经具备：

- 从入口创建并运行一个 Agent；
- 一条用户可观察的本地工具调用链；
- 演示 Model 根据消息历史完成 `ToolCall → ToolMessage → FinalAnswer`；
- Tool Definition、Registry、Executor、Memory、Factory 和 Agent Loop 的首次纵向串联；
- 正式入口使用真实模型、真实工具调用与流式回答；
- 同一终端会话内循环输入、显式退出和多轮消息历史。

当前仍没有：

- SSE 流式 HTTP 输出或 Web Playground；
- MCP 自动重连、断线恢复和产品化诊断；
- 持久化的会话与多用户隔离；
- RAG、Web Playground 或 Multi-Agent。

因此，原计划中的“Phase 1 接近完成”只表示**内部组件实现进度**，不代表**可运行产品进度**；A.1—A.2 已开始把这些零件转化为可运行、可观察、可解释的纵向能力。

### 2.2 教学断点

当前问题不是代码没有价值，而是教学顺序失衡：

1. **组件先于场景**：先实现接口、工厂、适配器和生命周期，之后才准备连接真实入口。
2. **测试先于行为理解**：现有 `internal/` 下有 86 个顶层测试函数，测试代码约 5429 行，生产代码约 1976 行；测试约为生产代码的 2.75 倍。这个比例本身不代表质量问题，但说明课程把大量注意力放在边界覆盖，而不是先建立完整运行心智模型。
3. **“完成”的定义过窄**：组件测试通过就被标为完成，但没有检查学习者能否运行、观察和解释它在整条链路中的位置。
4. **主入口长期没有增长**：核心能力停留在 `internal/` 与测试替身中，学习者无法从程序表面看到项目正在变成什么。
5. **抽象缺少问题来源**：没有先经历具体重复、失败或变化，就难以理解为什么需要 Registry、Factory、Adapter、Memory 等边界。

### 2.3 调整策略

- 暂停原定的 **1.7.4 MCP 超时、关闭、断线与重连**。
- 不推倒现有实现；把它们视为待接入、待解释、待验证的零件库。
- 先补齐“终端输入 → Agent → Model → Tool → Model → 最终回答”的纵向切片。
- 后续每出现一个真实问题，再回到对应组件解释其设计，并按需补测试。
- 持久化、HTTP、RAG、Multi-Agent 等阶段全部改为纵向功能驱动，不再先做基础设施大全。
- 路线校准图：[`docs/images/learning-route-reset.svg`](docs/images/learning-route-reset.svg)。

## 3. 新的进度口径

以后每项能力同时记录四个维度，不能再用单个“已完成”概括：

| 维度 | 含义 | 验收问题 |
|---|---|---|
| 实现 | 代码是否存在并通过适用检查 | 这个组件能独立工作吗？ |
| 串联 | 是否进入从入口开始的真实调用链 | `main` 或真实请求会调用它吗？ |
| 可见 | 学习者是否能从终端/API 观察到效果 | 改动前后有什么可见差异？ |
| 理解 | 学习者是否能解释职责和取舍 | 为什么存在、谁调用、不这样做会怎样？ |

状态值统一使用：`未开始`、`已实现待串联`、`已串联待理解`、`已掌握`、`阻塞`。

“课程完成”必须同时满足：

1. 代码已实现。
2. 功能已经从真实入口运行。
3. 学习者观察过成功路径，并至少观察一个本节关心的失败路径。
4. 学习者能沿调用链解释本节涉及的文件和方法。
5. 适用的格式化、测试、静态检查或构建通过。
6. 本计划已更新验收记录与下一步。
7. 已执行 `.agents/skills/agenthub-learning-checkpoint/SKILL.md` 的 Git 检查点流程。

## 4. 教学方式

### 4.1 每节课的固定顺序

1. **先看现象**：先运行当前程序，记录它能做什么、不能做什么。
2. **提出一个问题**：本节只解决一个用户可感知的问题。
3. **画出调用链**：只列本节会经过的文件、方法和数据。
4. **逐点解释**：解释每个文件/方法的调用者、输入、输出、副作用和失败方式。
5. **学习者写核心代码**：除非明确要求，不直接代写核心实现。
6. **先运行功能**：先确认真实入口产生预期效果。
7. **再写最小测试**：只固定本节新增行为和关键失败边界。
8. **回看抽象**：用刚出现的问题解释为什么需要或不需要封装。
9. **一轮理解验收**：每节只集中提问一轮；学习者回答后，直接指出正确部分、统一修正偏差并给出结论，不因非阻塞性偏差继续追加口试。
10. **Git 检查点**：只有会阻碍下一节的关键误解才需先澄清，否则完成总结后进入检查点。

### 4.2 文件与方法解释模板

每次引入或修改一个关键文件/方法，教学必须回答：

- 谁调用它？
- 它接收什么，返回什么？
- 它修改了什么状态或产生了什么副作用？
- 错误交给谁处理？
- 为什么放在这个包，而不是调用者中？
- 如果删除这层封装，当前会发生什么具体问题？
- 它是当前必需，还是为未来预留？若是后者，默认暂缓。

### 4.3 新增抽象的门槛

只有同时满足以下条件，才新增接口、工厂、适配器或管理器：

1. 已经出现至少两个具体实现、调用方式或明确的变化来源；或者边界隔离是外部依赖测试所必需。
2. 能指出抽象前的具体重复、耦合或不可测试问题。
3. 能用一句话说明调用方因此变简单了什么。
4. 本节会立即使用这层抽象，而不是只为遥远未来预留。

否则优先写最直接的具体代码，等问题出现后再重构。

### 4.4 测试策略

测试不再按“尽可能多”推进，而按风险推进：

- 先有可运行行为，再写测试。
- 每节默认只要求：一个成功路径、一个最重要失败路径、一个已发现缺陷的回归测试。
- 纯数据结构、简单构造函数和语言本身已保证的行为，不为覆盖率单独堆测试。
- 只有并发、资源关闭、错误分层、协议顺序、外部边界等高风险逻辑，才扩展矩阵测试。
- 覆盖率只作为发现遗漏的工具，不作为课程目标。
- 阅读旧测试时先回答“它保护了什么用户或系统行为”，再讨论测试写法。

## 5. 当前能力盘点

| 能力 | 实现状态 | 串联/可见状态 | 后续处理 |
|---|---|---|---|
| 配置、日志、应用错误 | 教学实现已归档 | 旧配置、日志和错误包已随自研 Runtime 迁入 `examples/selfbuilt-runtime/internal/`；正式入口使用当前环境配置与错误处理 | 后续出现生产需求时从真实入口重新提取 |
| Message / Model 协议 | 已实现 | 正式入口使用 Eino `schema.Message` 与 OpenAI 兼容模型；自研协议封闭在教学示例 | Phase D 继续沿 Eino 协议扩展 |
| Tool / Registry / Executor | 已实现 | 正式入口使用 Eino Tool/ToolsNode；自研 Tool、Registry、Executor 封闭在教学示例 | Phase D 接真实 MCP |
| Agent Loop | 已实现 | 正式入口使用 Eino ReAct Agent；自研 Loop 封闭在 `examples/selfbuilt-runtime/internal/` | 生产主线只继续演进 Eino |
| Memory | 已实现 | 正式入口维护完整会话历史，并为每轮模型调用生成最近 3 个完整用户轮次的上下文视图；裁剪统计可见，工具调用链不会被拆散 | Phase F 持久化时区分会话历史、上下文策略和长期记忆 |
| Streaming | 已实现 | 正式入口已消费 OpenAI 兼容服务的真实增量流；教学 Pipe 对照保留在历史示例中 | Phase E 映射为 SSE 事件 |
| Agent Factory | 教学实现已归档 | 自研 Factory 仅由 `examples/selfbuilt-runtime` 使用；正式入口直接组装 Eino ReAct Agent | 出现真实重复后再提取生产组装边界 |
| MCP Session / Adapter / Manager | 多 Server 已实现 | 正式入口通过可读配置启动两个 stdio MCP Server；每个 Server 对应一个 Session，Manager 统一发现、命名空间包装和关闭工具连接 | D.4 在真实连接上处理生命周期与稳定性 |
| HTTP / SSE | 最小 HTTP `/chat` 已实现 | `go run ./cmd/agenthub serve` 提供 `POST /chat`，可调用同一 Eino ReAct Agent 与 MCP 工具；SSE 尚未实现 | E.2 以 Hertz + SSE 同时带来框架能力与流式可见效果 |
| 持久化 | 未实现 | 不可见 | 由“重启后会话丢失”这个问题驱动 |
| RAG / Workflow / Multi-Agent | 未实现 | 不可见 | 由具体用户场景驱动 |

详细历史验收记录移至 [`docs/learning-history.md`](docs/learning-history.md)，不再占用主路线的注意力。

## 6. 校准后的课程路线

完整的产品目标、系统边界、技术选型理由、阶段课程与验收标准见 [`docs/product-blueprint.md`](docs/product-blueprint.md)。主计划只保留阶段导航和当前进度，避免再次被大量细节淹没。

| 阶段 | 要解决的核心问题 | 用户可见成果 | Runtime 策略 | 状态 |
|---|---|---|---|---|
| Phase A：恢复可见主线 | 已有零件没有进入应用入口 | 终端跑通并解释一次 Model—Tool—Model 闭环 | 使用现有自研内核学习原理 | 已完成 |
| Phase B：Eino 对照实验 | 继续手写会重复建设，直接换框架又会形成黑盒 | 用 Eino 重做同一用例并完成概念对照 | 冻结自研内核，生产主线切到 Eino | 已完成 |
| Phase C：可用 CLI Agent | 演示 Model 不能解决真实问题 | 真实模型、多轮对话、工具调用和流式终端 | Eino | 已完成：C.1—C.5 已掌握 |
| Phase D：工具中心与 MCP | 本地工具难扩展，MCP 还没有真实连接 | 可发现、调用和诊断真实 MCP 工具 | Eino + AgentHub MCP 配置层 | 暂停：D.1—D.3 已掌握，D.4 第一阶段已实现；后续稳定性按真实故障补齐 |
| Phase E：HTTP 与 Web Playground | CLI 无法被其他应用调用，产品形态不可见 | HTTP/SSE API 和最小聊天控制台 | AgentHub 应用层调用 Eino | 已完成：E.1—E.7 已掌握 |
| Phase F：会话与配置持久化 | 重启后 Agent 和会话丢失 | Agent CRUD、历史会话恢复 | PostgreSQL + Repository | 进行中：F.1 已掌握，下一节 F.2 数据库迁移与连接池配置 |
| Phase G：知识库与 RAG | Agent 不能可靠回答私有文档问题 | 文档上传、检索和带引用回答 | Eino Retriever + pgvector | 待开始 |
| Phase H：Workflow 与 Multi-Agent | 复杂任务需要可控分工和恢复 | 一个有基线对照的编排场景 | Eino Graph/Workflow/Agent | 待开始 |
| Phase I：生产化与部署 | 本机可跑但不可维护、诊断和交付 | Trace、指标、评测、安全、Docker 和 V1 演示 | 生产保障层 | 待开始 |

### 里程碑

| 里程碑 | 完成后能做什么 | 阶段 |
|---|---|---|
| M0 看懂内核 | 能运行并解释一次模型—工具闭环 | A |
| M1 可用 CLI Agent | 能与真实模型多轮、流式对话并调用工具/MCP | B—D |
| M2 Agent 服务 MVP | 能通过 API 和 Web Playground 使用，数据可保存 | E—F |
| M3 知识 Agent | 能上传文档并获得带引用的回答 | G |
| M4 编排平台 | 能运行一个可观察、可恢复的复杂任务流程 | H |
| V1 可部署 AgentHub | 能从零启动并完成配置、问答、工具、知识库和观测演示 | I |

### Phase A 当前课程

| 课程 | 本节先看到的问题 | 学习者核心实现 | 用户可见结果 | 状态 |
|---|---|---|---|---|
| A.0 现状地图与基线 | 代码很多，但入口只打印启动日志 | 沿入口标出已连接与未连接组件 | 能解释“现在为什么看不到 Agent” | 已审计待讲解 |
| A.1 最小终端 Agent：直接回答 | `main` 没有创建 Agent | 写最小演示 Model，并从入口组装、运行一次 Agent | 终端看到回答与 step 数 | 已掌握 |
| A.2 本地工具闭环 | 直接回答还看不出 Agent 与普通聊天的差别 | 注册一个简单本地工具，让演示 Model 先请求工具再回答 | 终端看到 ToolCall、ToolResult 和 FinalAnswer | 已掌握 |
| A.3 调用链复盘 | 能运行但仍可能只会照着写 | 为真实链路画图并逐方法解释；不新增功能 | 能从 `main` 讲到最终回答并指出每层必要性 | 已掌握 |
| A.4 最小入口测试 | 已有单测很多，但入口行为没有保护 | 为纵向切片写一个最小 smoke/integration test | 改坏组装或消息顺序时测试失败 | 已掌握 |

Phase A 明确不接真实模型、不接 Eino、不做 HTTP、不继续 MCP 生命周期。A 阶段结束后立即进入 Eino 对照实验，不再扩建第二套生产 Runtime。

### Phase B 当前课程

| 课程 | 本节先看到的问题 | 学习者核心实现 | 用户可见结果 | 状态 |
|---|---|---|---|---|
| B.1 Eino 最小直接回答对照 | 直接切到 Eino 容易只会照抄框架代码 | 实现最小 `BaseChatModel`，用 `Chain → Compile → Runnable.Invoke` 执行一次请求 | `go run ./examples/eino-direct` 输出固定 Assistant 回答 | 已掌握 |
| B.2 Eino 工具闭环 | 直接回答仍看不出 Eino 如何承担 Agent 循环 | 用 Eino Tool/ToolsNode/Agent 复现 A.2 天气工具链路 | 终端再次看到 ToolCall → ToolResult → FinalAnswer | 已掌握 |
| B.3 流式和回调观察 | 同步结果看不到生成过程和节点边界 | 用最小例子观察 Eino 流式输出与模型、工具节点回调 | 终端可区分增量输出和节点开始/结束 | 已掌握 |
| B.4 架构决策与生产切换 | 两套 Runtime 都能运行，但生产职责与后续边界仍可能混淆 | 用 ADR 固化 Eino 与 AgentHub 的职责边界，不新增运行时代码 | 明确生产只使用 Eino，自研 Runtime 仅保留为学习对照 | 已掌握 |
| B.5 清理双轨风险 | 正式入口名仍运行自研 Runtime，教学入口与生产入口并列 | 将 Eino 流式闭环迁入唯一正式入口，把阶段性实现归档到 `examples/` | `go run ./cmd/agenthub` 明确运行 Eino，教学对照仍可独立运行 | 已掌握 |

B.1—B.3 使用 `github.com/cloudwego/eino v0.9.19`，最初以独立入口完成自研与 Eino 的行为对照。B.5 完成入口收敛后，`cmd/agenthub` 成为唯一正式入口并运行 B.3 的 Eino 流式 ReAct 闭环；自研入口、Eino 直接回答和同步工具闭环分别归档至 `examples/selfbuilt-runtime`、`examples/eino-direct` 与 `examples/eino-tool`。

B.4 新增 [`docs/decisions/0001-use-eino-for-production-runtime.md`](docs/decisions/0001-use-eino-for-production-runtime.md)，确定 Eino 承担通用 Agent 执行，AgentHub 保留配置、入口、会话、持久化、安全和产品观测等应用职责。当前没有真实重复或变化来源，因此不预先创建项目级 `Runtime` 接口；是否抽取边界由 Phase C/E 的真实调用方决定。

## 7. 每节课的计划记录模板

```markdown
### X.Y 课程名称

- 状态：未开始 / 进行中 / 已实现待串联 / 已串联待理解 / 已掌握 / 阻塞
- 用户可见目标：
- 当前现象：
- 本节只解决：
- 不在本节解决：
- 调用链：
- 学习者核心代码：
- 关键 Go 语法：
- 为什么需要当前封装：
- 最小测试：
- 运行验证：
- 理解验收题：
- Git 提交：
- 下一步：
```

## 8. 当前学习位置

- 当前阶段：**Phase F：会话与配置持久化**
- 当前路线决策：**F.1 已完成；SessionManager 从内存 map 迁移到 PostgreSQL，sessions/messages 两张表，message_json 存储完整 schema.Message 序列化，Append 用事务保证一致性，服务重启后会话历史可恢复。下一节 F.2 数据库迁移与连接池配置。**
- 已掌握：**A.1—A.4、B.1—B.5、C.1—C.5、D.1—D.3、E.1—E.7、F.1**
- A.1 可见结果：`go run ./examples/selfbuilt-runtime` 输出启动信息、固定 Assistant 回答、`steps: 1` 和零值 Usage
- A.1 调用链：[`docs/images/a1-direct-answer-flow.svg`](docs/images/a1-direct-answer-flow.svg)
- A.1 理解验收：能解释隐式接口实现与编译期检查的区别、Factory 创建 Memory 的职责、空 Registry 不妨碍直接回答，以及 `Steps` 表示模型调用次数
- A.2 可见结果：终端按顺序输出 `ToolCall`、`ToolResult`、最终回答和 `steps: 2`
- A.2 调用链：[`docs/images/a2-local-tool-loop.svg`](docs/images/a2-local-tool-loop.svg)
- A.2 理解验收：能区分 Definition 与 Handler、Registry 的模型侧与执行侧职责、Tool Name 与 ToolCall ID，并说明 Schema 失败不会进入 Handler，而会作为错误 ToolMessage 交给下一轮模型
- A.3 职责与价值图：[`docs/images/a3-runtime-boundaries.svg`](docs/images/a3-runtime-boundaries.svg)
- A.3 复盘结论：`main.go` 是组合根；Factory 只创建 Memory 并组装 Agent；`Agent.Run` 是同步薄入口，`runWithGenerator` 承担循环；Registry 是工具事实来源，Executor 执行已确定的 ToolCall，Model 决定调用什么工具，Agent Loop 推进历史和步骤
- A.3 抽象判断：Registry、Executor、Message 协议和 Agent Loop 已被真实闭环证明；Factory 有实际价值但阅读成本偏高；Memory 已进入链路但协议安全裁剪仍待验证；MCP、HTTP、持久化、RAG 等尚未被当前入口证明
- A.3 风险发现：`SimpleSliding` 按单条消息裁剪，`MaxKeep` 太小时可能保留孤立 ToolMessage，拆散 Assistant ToolCall 与 ToolMessage 的协议关联
- A.4 最小测试：`examples/selfbuilt-runtime/main_test.go` 直接调用 `runDemoAgent`，保护真实 `demoModel`、天气 Tool、Registry、Factory 与 Agent Loop 的纵向组装
- A.4 稳定断言：四条消息按 User → Assistant ToolCall → Tool → Final Assistant 排列，Tool Name 与 Call ID 前后对应，工具结果进入最终回答，且 `Steps == 2`
- A.4 边界结论：`runDemoAgent` 隔离可验证的业务组装，`run` 保留配置、日志与终端展示；入口测试不重复包内单测已经覆盖的 Agent Loop 内部异常分支
- A.4 理解验收：能解释为何不直接测试混合配置和输出的 `run`、为何入口只保护跨组件闭环，以及 Call ID 与 Steps 分别证明调用关联和两轮模型执行
- B.1 可见结果：`go run ./examples/eino-direct` 通过单节点 Eino Chain 输出 `assistant: 你好，我是 AgentHub 的 Eino 演示 Agent。`
- B.1 代码边界：`demo_model.go` 实现 `BaseChatModel` 的 Generate/Stream；`main.go` 负责创建 Chain、追加模型、Compile 并 Invoke，不修改自研入口
- B.1 类型流：`NewChain[[]*schema.Message, *schema.Message]` 声明整条流程接收消息列表并返回一条消息；Compile 把可修改的流程定义转换为 `Runnable`
- B.1 概念映射：自研 `llm.Message` 对应 `schema.Message`，自研 `llm.Model` 对应 `BaseChatModel`，自研同步运行入口对应 Chain 编排后的 `Runnable.Invoke`
- B.1 理解验收：能说明 Chain 定义数据流、Runnable 才可执行；接口方法集要求教学 Model 同时实现 Generate/Stream；使用 Chain 不是强制规范，而是为后续连接 Prompt、Tool、Retriever 提前掌握可扩展编排方式
- B.1 对照图：[`docs/images/b1-eino-direct-answer-flow.svg`](docs/images/b1-eino-direct-answer-flow.svg)
- B.1 已知环境提示：Sonic 在当前环境回退到 `encoding/json`，不影响本节行为，仅可能影响 JSON 性能
- B.2 可见结果：`go run ./examples/eino-tool` 按顺序输出 User、ToolCall、ToolResult、Final Assistant 和 `steps: 2`
- B.2 代码边界：`demo_tool.go` 用 `InferTool` 包装参数 Schema 与 Handler；`demo_model.go` 实现 `ToolCallingChatModel`；`main.go` 用 `react.NewAgent` 组装工具闭环并用 `WithMessageFuture` 观察消息
- B.2 职责映射：Tool 的 `Info()` 供 Model 选择工具与生成参数，`InvokableRun()` 供 ToolsNode 执行；Eino ReAct Agent 承担自研 Agent Loop 的路由、历史追加与结束判断
- B.2 Steps 口径：两次 Model 产生 Assistant Message，工具绑定发生在创建阶段、工具执行发生在 ToolsNode，二者都不计 Step
- B.2 理解验收：已经能说明主流程和主要职责映射；当前不要求背熟 Eino API，`WithMessageFuture` 只负责旁路观察，不参与闭环执行
- B.2 最小测试：`examples/eino-tool/main_test.go` 保护 ToolCall → ToolMessage → Final Assistant 的消息协议，并验证未绑定工具时 Model 明确拒绝
- B.2 调用链：[`docs/images/b2-eino-tool-loop.svg`](docs/images/b2-eino-tool-loop.svg)
- B.3 可见结果：`go run ./cmd/agenthub` 先显示两轮 ChatModel 和一次 Tool 的 Callback 事件，再逐块打印最终回答；最终回答由两个分时 Chunk 拼接，中间可观察到约 1 秒等待
- B.3 流式实现：最终回答使用无缓冲 `schema.Pipe`；生产 goroutine 通过 `Send` 分时发送，消费端循环 `Recv`，生产端和接收端分别关闭自己的流端，并传播 Context 取消
- B.3 Callback 边界：`compose.WithCallbacks` 经 `agent.WithComposeOptions` 注入本次 ReAct 调用；Callback 只旁路观察 ChatModel 与 Tool 的开始、流就绪、结束和错误，不参与模型决策、工具执行、消息追加或路由
- B.3 生命周期口径：流式 ChatModel 显示 `start → stream_ready`，同步 InvokableTool 显示 `start → end`；`stream_ready` 仅表示 StreamReader 已返回，不代表所有 Chunk 已经产生
- B.3 理解验收：能说明第一轮模型在已绑定工具说明的前提下决定发起 ToolCall，工具结果进入历史后第二轮模型才产生最终回答；删除 Callback 不影响同一条 ReAct 闭环，只会失去观察日志
- B.3 调用链：[`docs/images/b3-stream-callback-flow.svg`](docs/images/b3-stream-callback-flow.svg)
- B.4 架构决策：[`docs/decisions/0001-use-eino-for-production-runtime.md`](docs/decisions/0001-use-eino-for-production-runtime.md)
- B.4 职责边界图：[`docs/images/b4-production-runtime-boundary.svg`](docs/images/b4-production-runtime-boundary.svg)
- B.4 决策结论：后续生产功能统一使用 Eino；自研 Runtime 冻结为学习对照；AgentHub 继续承担配置、产品入口、用例、会话、持久化、安全、错误映射和产品级观测
- B.4 抽象结论：是否新增项目级 `Runtime` 接口取决于真实调用方是否产生重复、耦合或变化来源，而不取决于 Eino 是否存在同名接口；当前不创建预留抽象
- B.4 理解验收：能说明框架降低通用 Agent 执行的自研和维护成本，AgentHub 仍需编写具体产品能力，并接受在真实问题出现前不提前封装 Runtime
- B.5 入口收敛：`cmd/agenthub` 现在是唯一正式入口并运行 Eino 流式 ReAct 闭环；Phase A 自研入口与 B.1/B.2 阶段性实现归档到 `examples/`
- B.5 保留边界：自研 `agent`、`llm`、`tool` 等包在 Phase C 收口时已整体迁入 `examples/selfbuilt-runtime/internal/`，只维持可运行学习对照，不是未来生产底层
- B.5 理解验收：能判断真实模型应从 `cmd/agenthub` 接入；已统一校正自研 Runtime 包的保留原因与生产身份
- C.1 真实模型：`cmd/agenthub/openai_model.go` 从环境变量创建 Eino OpenAI 兼容 ChatModel，并以 60 秒请求超时保护外部调用
- C.1 配置边界：`cmd/agenthub/env.go` 在应用启动时可选加载 `.env`；系统环境变量优先，`.env` 不存在不阻塞生产启动，真实密钥保持在 Git 之外
- C.1 可见结果：真实模型第一轮生成 `get_weather` ToolCall，工具返回结果后第二轮模型流式生成最终天气回答；一次实测最终回答收到 14 个增量 Chunk
- C.1 最小测试：只保护环境变量优先、缺少 `.env` 可启动、缺少 API Key 联网前失败，不用网络测试重复验证 Provider
- C.1 理解验收：能说明 `.env` 属于应用启动配置、系统环境用于部署覆盖，以及流式 Chunk 是服务端增量片段而非固定的一 Token 一 Chunk
- C.2 交互式 CLI：`cmd/agenthub/main.go` 循环读取终端输入，支持空输入跳过、`exit` / `quit` / EOF 退出，并在同一进程内维护会话历史
- C.2 历史收集：每轮把 UserMessage 加入 `history`，用新的 `react.WithMessageFuture()` 收集本轮 Assistant ToolCall、ToolMessage 和最终 AssistantMessage，合并流式分片后追加到历史
- C.2 失败边界：Agent 启动或流接收失败时打印错误并回滚本轮 UserMessage，避免一次模型超时终止整个 CLI 或留下无回答的孤立用户消息；模型请求超时由 30 秒调整为 60 秒
- C.2 可见结果：同一终端先问“杭州天气”，再问“那上海呢？”，第二轮结合历史继续调用 `get_weather` 并回答上海天气；输入 `exit` 后正常退出
- C.2 职责边界：终端 I/O 是 `cmd/agenthub` 的场景适配职责，Eino ReAct Agent 只负责基于消息执行模型—工具循环；当前历史由 CLI 应用层显式传入和追加，不是 Eino 自动持久化
- C.2 理解验收：能说明终端 I/O 不应进入通用 Agent 执行层，并能用跨用户称呼串线的例子解释会话历史必须按用户/会话隔离
- C.2 调用链：[`docs/images/c2-interactive-cli-flow.svg`](docs/images/c2-interactive-cli-flow.svg)
- C.3 真实本地工具：正式入口用 `read_project_file` 替换固定天气演示工具，模型根据 Tool Schema 与 SystemMessage 自主决定是否读取项目文件
- C.3 安全边界：只允许项目根目录内的常见 UTF-8 文本文件；拒绝绝对路径、`../` 穿越、符号链接逃逸、`.env`、目录和超过 64 KiB 的文件
- C.3 错误分层：路径拒绝、文件不存在或过大等可恢复失败以 ToolResult 文本返回，使模型能够解释并继续会话；Context 取消以 Go error 中断已失效的执行
- C.3 最小测试：验证合法文本读取、路径穿越与 `.env` 拒绝、符号链接逃逸拒绝；测试使用 `t.TempDir()`，不读取真实仓库
- C.3 可见结果：自然语言要求概括 `README.md` 时模型直接调用 `read_project_file`；普通 `defer` 问答不调用工具；读取 `.env` 时模型解释安全拒绝且 CLI 继续运行
- C.3 模型兼容：SystemMessage 约束需要文件时直接生成 ToolCall，避免模型先输出计划文字后被默认流式检查器提前判定为最终回答
- C.3 理解验收：已理解 `.env` 等受限访问需要保护敏感信息；统一补全可恢复 ToolResult 与系统 Go error 的区别，以及 Tool 名称、描述、参数 Schema 和 SystemMessage共同参与模型工具决策
- C.3 调用链：[`docs/images/c3-project-file-tool-flow.svg`](docs/images/c3-project-file-tool-flow.svg)
- C.4 流消费边界：新增 `writeAssistantStream`，由单一消费者负责逐块读取、输出、统计 Chunk，并通过 `defer` 在正常 EOF 或提前失败时关闭 StreamReader
- C.4 成功语义：收到任意数量 Chunk 都只代表部分进度；只有 `Recv()` 返回 `io.EOF` 才表示生产端不会再发送数据，本轮回答可以进入后续消息收集
- C.4 失败语义：读取、nil Chunk 或输出写入失败时返回已接收 Chunk 数和错误；终端已显示的部分内容无法撤回，但 `run` 会回滚本轮 UserMessage，不把残缺回答加入历史
- C.4 职责边界：`writeAssistantStream` 管理单个 StreamReader 生命周期，`run` 管理终端会话、Agent 调用和历史一致性；当前仍不抽取新的 Runtime 接口
- C.4 最小测试：由课程辅助补充正常多 Chunk 到 EOF 与半途失败两条测试，学习者聚焦核心流消费实现
- C.4 可见结果：真实模型回答完成后打印 52 个 Chunk 并重新出现输入提示符，普通知识问答未调用项目文件工具
- C.4 理解验收：能说明 EOF 是完整成功边界；流由唯一消费者关闭，历史回滚由掌握会话状态的 `run` 负责
- C.4 生命周期图：[`docs/images/c4-stream-lifecycle.svg`](docs/images/c4-stream-lifecycle.svg)
- C.5 上下文窗口：新增 `internal/session.ContextWindow`，完整 `history` 保留全部成功消息，每轮从中派生只含第一条 SystemMessage 与最近 3 个完整用户轮次的模型视图
- C.5 轮次边界：以 UserMessage 作为裁剪起点，User 之后直到下一个 User 之前的 Assistant ToolCall、ToolMessage 和最终 AssistantMessage 作为整体保留，避免形成孤立工具消息
- C.5 可见结果：第四轮提问时终端显示 `history=8 model=6 dropped=2 turns=3`；最早的代号问答仍在完整历史中，但不再发送给模型，因此模型无法从当前上下文回答代号
- C.5 最小测试：保护非法窗口参数、SystemMessage 永久保留、最近轮次选择、工具调用链完整性及原始 history 不被缩短
- C.5 职责边界：`ContextWindow` 负责无副作用地构建派生视图，`run` 继续拥有完整历史并决定何时追加或回滚；当前按轮次数裁剪，不实现 Token 预算、摘要或长期记忆
- C.5 理解验收：能说明不能用裁剪视图覆盖完整 history，并能解释按最后 N 条消息裁剪会破坏对话起点和 Assistant ToolCall → ToolMessage 协议关联
- C.5 上下文图：[`docs/images/c5-context-window.svg`](docs/images/c5-context-window.svg)
- Phase C 收口边界：自研 Runtime、旧配置日志支持包和绑定旧 Tool 协议的 MCP 原型已整体迁入 `examples/selfbuilt-runtime/`；根 `internal/` 只保留当前生产会话代码
- Phase C 收口编译约束：教学实现放入 `examples/selfbuilt-runtime/internal/` 后，Go 会禁止该目录树以外的 `cmd/agenthub` 导入，从“文档约定”升级为编译期隔离
- Phase C 收口验证：正式入口依赖中只出现 `agenthub/internal/session`；`go run ./examples/selfbuilt-runtime` 仍完成自研 ToolCall 闭环；全项目测试、静态检查和构建通过
- Phase C 收口理解验收：理解 `examples` 名称只表达意图，嵌套 `internal` 才提供可靠导入边界；旧 MCP 原型因绑定自研 Tool/Registry 且缺少真实传输，不作为 Phase D 生产基础
- Phase C 收口图：[`docs/images/c-phase-boundary-cleanup.svg`](docs/images/c-phase-boundary-cleanup.svg)
- D.1 工具目录：新增 `internal/toolcatalog.Catalog`，保存 Eino 工具实例、来源和启用状态；`List` 从 `Tool.Info` 读取名称、描述与参数 Schema，避免维护第二份模型可见元数据
- D.1 执行边界：`Catalog.EnabledTools()` 只筛选交给 ReAct Agent 的工具，ToolCall 执行仍由 Eino ToolsNode 完成；Catalog 不解析参数、不执行 Handler、不推进 Agent Loop
- D.1 CLI：新增 `/tools` 本地控制命令，在创建 UserMessage 前截获并展示全部目录条目，因此禁用工具可被诊断但不会进入模型工具集合，也不会污染会话历史
- D.1 可见结果：`/tools` 展示 1 个启用的本地 `read_project_file` 及真实参数 Schema且无模型 Callback；随后读取 `README.md` 仍正常触发 ToolCall 并生成回答
- D.1 最小测试：保护启用工具筛选、禁用工具仍可见、来源与 Schema 提取，以及 CLI 的来源和状态输出
- D.1 理解验收：能说明 Catalog 的注册仅是条目保存和查询，不是执行 Registry；`List` 展示全部有效条目，非法元数据返回错误，`EnabledTools` 才执行启用过滤
- D.1 调用链：[`docs/images/d1-tool-catalog-flow.svg`](docs/images/d1-tool-catalog-flow.svg)
- D.2 真实连接：正式入口使用当前 AgentHub 可执行文件启动独立 stdio MCP Server 子进程，`internal/mcpclient.Session` 完成初始化握手与工具发现，Eino 官方 MCP Adapter 将 `add_numbers` 转换为 `tool.BaseTool`
- D.2 Catalog 接线：发现到的远程工具以 `SourceMCP` 加入 D.1 Catalog；`/tools` 同时展示本地 `read_project_file` 与远程 `add_numbers`，ReAct Agent 仍通过统一的 `EnabledTools()` 调用
- D.2 安全与生命周期：MCP 子进程不继承 AgentHub 模型密钥环境，stdio stdout 只承载协议消息；CLI 退出时关闭 Session 和子进程，初始化或工具发现失败会关闭 Client 并返回 Go error
- D.2 最小测试：使用测试二进制启动真实 stdio 子进程，保护 Initialize → tools/list → tools/call 成功链路，并验证 Server 在握手前退出时返回带初始化语义的错误
- D.2 可见结果：`/tools` 展示 2 个启用工具；模型调用 MCP `add_numbers` 计算 17.5 + 24.5，Callback 显示 ChatModel → Tool → ChatModel，最终回答为 42
- D.2 理解验收：能说明 Session 持有 MCP Client 连接和已发现工具，Eino MCP Adapter 负责 MCP Tool 与 Eino Tool 的协议转换，Catalog 负责分类与启用筛选，ReAct Agent 决策调用、ToolsNode 执行；能区分可交给模型处理的远程业务失败与必须中断当前执行的连接、进程或 Context 失败
- D.2 调用链：[`docs/images/d2-stdio-mcp-flow.svg`](docs/images/d2-stdio-mcp-flow.svg)
- D.3 多 Server：正式入口以 `ServerConfig` 明确组装 `calculator` 与 `accounting` 两个 stdio MCP Server；`mcpclient.Manager` 为每份配置建立独立 Session，统一保存工具与倒序关闭连接
- D.3 冲突现象：两个 Server 都暴露远程 `add_numbers`；直接拼接工具时 `/tools` 出现重名，Eino ToolsNode 的名称索引会被后加入的同名工具覆盖，路由结果依赖顺序
- D.3 命名空间：`namespacedInvokableTool.Info()` 向模型和 ToolsNode 暴露 `calculator__add_numbers` / `accounting__add_numbers`，`InvokableRun()` 委托原 MCP Adapter，Server 收到的仍是原始 `add_numbers`
- D.3 Catalog 可见性：MCP 条目额外保存所属 Server，`/tools` 同时展示唯一名称、来源、Server、状态、描述和参数 Schema；本地工具不显示空 Server
- D.3 失败清理：`OpenServers` 在后续 Server 启动或工具适配失败时关闭此前已建立的 Session，`errors.Join` 同时保留主错误和资源清理错误
- D.3 最小测试：真实启动两个测试 stdio 子进程，保护 `OpenServers → tools/list → 命名空间代理 → tools/call`，分别断言两个命名空间名称与 Server 标签返回值
- D.3 可见结果：`/tools` 展示 3 个唯一工具；模型分别调用两个命名空间工具，Callback 显示对应名称，结果分别为 `[calculator] 17 + 25 = 42` 与 `[accounting] 10.5 + 20.5 = 31`
- D.3 理解验收：能说明命名空间只改变模型侧路由名、远程调用仍使用原始名；能区分单连接 Session 与多 Server Manager，并理解部分启动失败时必须回收已创建资源。集中纠正了 Manager 不是“同一 Server 多 Client”管理器，以及 `errors.Join` 同时保留主错误和清理错误
- D.3 调用链：[`docs/images/d3-multi-mcp-namespace-flow.svg`](docs/images/d3-multi-mcp-namespace-flow.svg)
- D.4 第一阶段：`internal/mcpclient.Session` 新增 `ready`、`unavailable`、`closed` 状态，以 `sync.RWMutex` 保护并发读写；`Close()` 使用 `sync.Once` 保证底层 Client 最多关闭一次，并稳定返回首次关闭结果
- D.4 状态语义：只有 Initialize 与工具发现全部成功才返回 `ready` Session；开始关闭时先切换到 `closed`，终止状态不会再退回 `unavailable`；连接层后续可通过 `ensureReady` 与 `markUnavailable` 统一检查和记录状态
- 生产主线注释复盘：已覆盖应用入口与 ReAct 消息链、完整 history 与上下文视图、Tool Catalog、MCP Session/Manager/命名空间代理、stdio Server、本地文件工具、`.env` 和真实模型配置，重点记录职责边界、状态与资源生命周期、安全约束和错误归属，不逐行翻译明显代码
- 复盘验证：`go test ./...`、`go vet ./...`、`go build ./...` 与 `git diff --check` 全部通过；D.4 仍处于进行中，尚未实现连接握手超时、有限重试和工具调用超时分类
- 暂停项：**D.4 第二阶段（连接握手超时与有限重试）和 D.5（MCP 配置与诊断）**。它们在当前本地教学 Server 的正常路径上没有新的用户可见效果，且尚未由真实故障驱动；保留 D.4 第一阶段已有的 Session 状态与幂等关闭能力。
- E.1 入口边界：`cmd/agenthub` 只组装模型、本地工具、MCP Manager、Catalog 与 Eino ReAct Agent，再根据参数选择 `internal/cli` 或 `internal/httpapi`；完整 CLI 会话迁入 `internal/cli`，项目文件工具迁入 `internal/projecttool`
- E.1 HTTP 协议：`go run ./cmd/agenthub serve` 在 `127.0.0.1:8080` 提供 `POST /chat`，接收 `{"message":"..."}` 并返回 `{"answer":"..."}`；非法 JSON、空消息返回 400，Agent 执行失败返回 502
- E.1 可见结果：真实 HTTP 请求调用 `calculator__add_numbers` 计算 17 + 25，返回 HTTP 200 与答案 42；原 CLI `/tools`、流式问答和退出行为保持不变
- E.1 Context 边界：Handler 将 `request.Context()` 传给 Agent，使客户端断开、请求取消或上游超时能够取消后续模型与工具链路；`context.Background()` 不携带该请求生命周期
- E.1 测试接缝：`internal/httpapi` 用仅包含 `Generate` 的最小接口隔离真实模型，保护成功响应、非法 JSON、空消息、Agent 错误、nil 消息以及请求 Context 和消息输入；该接口只解决当前 Handler 的外部依赖测试问题
- E.1 验证：用户已真实运行 CLI 与 HTTP 成功路径；`go test ./...`、`go vet ./...`、`go build ./...` 和 `git diff --check` 全部通过
- E.1 理解验收：能说明 `cmd/agenthub` 是组合入口，CLI 与 HTTP 是不同适配层，`internal/httpapi` 不应知道 MCP Server 启动细节；集中校正 `request.Context()` 的核心价值是传播请求取消，而不只是表示单次请求
- E.2 框架迁移：`internal/httpapi` 从标准库 `net/http` 迁入 CloudWeGo Hertz；`NewServer` 返回 `*server.Hertz`，Handler 签名统一为 `func(ctx context.Context, c *app.RequestContext)`，`Run` 使用 `h.Spin()` 启动；`cmd/agenthub/main.go` 调用方式不变
- E.2 SSE 流式端点：新增 `POST /chat/stream`，调用 `reactAgent.Stream()` 获取增量流，通过 `sse.NewWriter(c)` 逐块推送 `chunk` 事件，流正常结束发送 `done` 事件，流中断发送 `error` 事件；`ctx` 携带客户端断开信号，可传播到模型与工具调用
- E.2 事件协议：`event: chunk` + `data: {"content":"..."}` 推送增量文本；`event: done` + `data: {}` 表示正常结束；`event: error` + `data: {"error":"..."}` 表示流中断；SSE writer 创建后只能写事件，不能再调用 `c.JSON`
- E.2 generator 接口：从仅含 `Generate` 扩展为同时含 `Generate` 和 `Stream`，`*react.Agent` 自然同时满足；测试 fake 实现只需覆盖这两个方法，不需要完整实现 react.Agent
- E.2 测试策略：同步 `/chat` 端点使用 Hertz `ut.PerformRequest` 内存测试；SSE `/chat/stream` 端点因 `sse.NewWriter` 需要真实网络 writer，使用 `net.Listen` + `WithTransport(standard.NewTransporter)` 启动随机端口真实服务器测试；覆盖成功路径、非法请求、Stream 启动失败和流中途错误
- E.2 可见结果：`curl -N -X POST /chat/stream` 实时观察到逐字 chunk 事件（实测 12 个 chunk + 1 个空 chunk + done），同步 `/chat` 行为保持不变；空消息和非法 JSON 返回 400
- E.2 验证：`go fmt ./...`、`go vet ./...`、`go build ./...`、`go test ./...` 全部通过；`internal/httpapi` 包 8 个测试全部通过
- E.2 理解验收：能说明 Hertz Handler 双参数职责分离、SSE 事件类型与 JSON 数据体的设计原因、`io.EOF` 与流关闭的语义、客户端断开通过 context 协作式取消的传播机制、Go 隐式接口对最小能力定义的好处
- E.3 静态页面：新增 `internal/httpapi/web/index.html` 单页聊天界面，内联 CSS 与 JS，不引入前端框架；包含消息列表、用户/助手气泡区分、输入框、发送按钮、Enter 发送、自动滚动到底部
- E.3 go embed：使用 `//go:embed web/index.html` 将页面固化进二进制，`indexHandler` 通过 `c.Data(200, "text/html; charset=utf-8", indexHTML)` 返回；运行时不依赖外部文件路径，部署只需一个二进制
- E.3 前端 SSE 消费：因 `/chat/stream` 是 POST，浏览器原生 `EventSource` 只支持 GET，故使用 `fetch()` + `response.body.getReader()` + `TextDecoder` 手动读取字节流；用 buffer 累积数据，按 `\n\n` 分割 SSE 事件，逐行解析 `event:` 和 `data:`
- E.3 流式渲染：收到第一个 `chunk` 事件时替换"正在输入..."占位文本，后续 chunk 追加到同一助手气泡；`done` 事件标记完成，`error` 事件显示错误样式；`isGenerating` 状态防止重复发送
- E.3 分层验证：E.3 完全未修改 `/chat` 和 `/chat/stream` 端点代码，只新增静态页面和前端逻辑；证明 API 协议层与表现层解耦，换 UI 不影响业务 API
- E.3 可见结果：浏览器打开 `http://127.0.0.1:8080/` 看到聊天页面，输入消息后回答流式显示；实测模型调用 `calculator__add_numbers` 计算 17+25，最终回答 42
- E.3 测试：新增 `TestIndexHandlerReturnsHTML` 验证 `GET /` 返回 200、正确 Content-Type 和页面内容；全项目 `go test ./...`、`go vet ./...`、`go build ./...` 全部通过
- E.3 理解验收：能说明 go embed 的作用与编译期固化特性、fetch 替代 EventSource 的原因（POST 限制）、TCP 字节流与 SSE 事件边界不对应因此需要 buffer、firstChunk 替换占位文本的作用、关注点分离原则（换传输协议或 UI 不影响业务逻辑）
- E.4 SSE 事件扩展：新增 `tool_start` 和 `tool_end` 两种 SSE 事件，data 格式为 `{"name":"工具名"}`；原有 `chunk`/`done`/`error` 事件保持不变
- E.4 Callback → Channel 桥梁：Eino Callback 运行在内部 goroutine 中，无法直接写 SSE；通过带缓冲 channel（容量 16）将工具事件从 Callback 传递到 handler 的 select 循环；非阻塞发送（`select { case ch <- evt: default: }`）避免 channel 满时阻塞 Eino 执行
- E.4 阻塞 Recv 转 select：`stream.Recv()` 是阻塞调用，不能直接放在 select 中；用 goroutine 将 Recv 结果打包为 `chunkResult{chunk, err}` 转发到 `chunkCh`，handler 用 select 同时监听工具事件和文本 chunk
- E.4 Callback 提取：`newToolCallback(toolEvents chan<- toolEvent)` 独立函数，只观察 `components.ComponentOfTool` 组件，忽略 ChatModel、Chain 等其他组件；便于单元测试和复用
- E.4 前端工具状态：收到 `tool_start` 显示"🔧 正在调用工具：xxx"，收到 `tool_end` 显示"✅ 工具调用完成：xxx"；`afterToolStatus` 标记确保工具状态与回答文本之间换行分隔；`firstContent` 统一控制第一个内容（无论是 tool_start 还是 chunk）替换"正在输入..."占位
- E.4 可见结果：curl 验证 SSE 流按顺序输出 tool_start → chunk → tool_end → chunk... → done；浏览器 Playground 显示工具调用状态和最终回答，实测 `calculator__add_numbers` 计算 17+25=42
- E.4 测试：新增 `TestNewToolCallbackSendsStartAndEnd` 验证 Tool 组件开始/结束正确发送事件；新增 `TestNewToolCallbackIgnoresNonToolComponents` 验证非 Tool 组件被过滤；全项目 `go test ./...`、`go vet ./...`、`go build ./...` 全部通过
- E.4 理解验收：能说明 Callback 旁路观察与 SSE 写入的 goroutine 隔离、channel 桥梁的必要性、带缓冲与非阻塞发送的设计原因、阻塞 Recv 转 select 的 goroutine 转发模式、select 随机选择对事件顺序的影响
- E.5 SessionManager：独立类型封装会话存储，包含 `sync.Mutex`、`map[string]*Session`、`maxHistory`；所有方法加锁保证并发安全，HTTP 每个请求在独立 goroutine 中运行
- E.5 Session 结构体：`ID`（32 字符 hex，由 crypto/rand 生成）、`History`（[]*schema.Message）、`LastAccess`（time.Time，为未来过期清理预留）
- E.5 核心方法：`GetOrCreate(id)`（空 ID 或不存在时新建）、`GetHistory(id)`（返回副本防止外部修改内部状态）、`Append(id, userMsg, assistantMsg)`（追加一轮对话，超过 maxHistory 截断最旧的）
- E.5 请求/响应改造：`chatRequest` 新增 `SessionID`（omitempty），`chatResponse` 新增 `SessionID`；SSE 通过 `doneEvent{SessionID}` 在 done 事件中返回会话 ID
- E.5 chatHandler 流程：GetOrCreate → GetHistory → 构造 [system, ...history, userMsg] → Generate → Append → 返回 {answer, session_id}
- E.5 streamChatHandler 流程：同上，但用 `strings.Builder` 累积所有 chunk 内容，流结束后构造完整 assistantMsg 并 Append，done 事件携带 session_id
- E.5 前端接入：`localStorage` 保存 `agenthub_session_id`；`loadSessionID()`/`saveSessionID()`/`clearSession()` 三个函数；请求 body 带上 session_id；done 事件解析并保存 session_id；新增"新对话"按钮清除会话并清空聊天区域
- E.5 可见结果：curl 验证两轮对话携带相同 session_id，第二轮 Agent 收到的消息包含第一轮历史；浏览器验证多轮对话记忆、新对话重置、页面刷新后会话保持
- E.5 测试：新增 `session_test.go`（12 个单元测试覆盖创建、获取、副本隔离、截断、ID 格式、唯一性等）；新增 `TestChatHandlerMultiTurnPreservesHistory`（验证第二轮消息包含第一轮历史）、`TestChatHandlerNewSessionWithoutID`、`TestStreamChatHandlerDoneEventIncludesSessionID`；更新旧测试断言以适配新增 session_id 字段
- E.5 理解验收：能说明为什么需要独立 SessionManager（map 并发不安全、逻辑封装）、GetHistory 返回副本的原因（防止数据竞争）、maxHistory 截断的设计（内存上限）、crypto/rand vs math/rand 的区别、流式接口累积完整回答后再存历史的原因、前端 localStorage 持久化的作用
- E.6 过期清理：SessionManager 新增 `ttl`（默认 30 分钟）和 `cleanupInterval`（默认 5 分钟）字段；`StartCleanup(ctx)` 启动后台 goroutine，用 `time.NewTicker` 定期触发 `cleanupExpired()`，用 `ctx.Done()` 控制退出
- E.6 cleanupExpired：加锁遍历所有 session，`now.Sub(s.LastAccess) > sm.ttl` 的 session 用 `delete()` 删除；遍历+删除都在锁内完成，session 数量不大时锁持有时间可忽略
- E.6 并发安全：所有公共方法（GetOrCreate、GetHistory、Append、cleanupExpired）均正确加锁；`go test -race` 运行全部测试无数据竞争报警
- E.6 并发测试：`TestSessionManagerConcurrentAccess` 启动 10 个 goroutine 各循环 100 次并发读写同一 session，验证 maxHistory 截断正确；`TestSessionManagerCleanupExpired` 手动设置 LastAccess 为过去时间验证清理；`TestSessionManagerStartCleanup` 用短 TTL 验证后台 goroutine 自动清理；`TestSessionManagerStartCleanupStopsOnCancel` 验证 context 取消后 goroutine 退出
- E.6 main.go 接入：创建 SessionManager 后立即调用 `sessionManager.StartCleanup(ctx)`，用 run() 的 ctx 作为父 context，服务退出时 goroutine 自动退出
- E.6 理解验收：能说明 time.NewTicker 的工作原理（内部 goroutine 定时往 channel 发值）、select 阻塞不消耗 CPU（goroutine 被 runtime 挂起，channel 就绪时唤醒）、channel 缓冲的作用（削峰，生产者不用等消费者）、goroutine 与 JS Web Worker 的区别（M:N 调度、轻量、共享内存）、context 控制 goroutine 生命周期的模式
- E.7 信号监听：`cmd/agenthub/main.go` 用 `signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)` 监听 Ctrl+C 和 kill 默认信号，收到信号时自动取消返回的 ctx；`defer stop()` 确保函数返回时恢复信号默认处理并释放内部资源
- E.7 Hertz 优雅关闭：`internal/httpapi/server.go` 的 `Run` 函数新增 `ctx context.Context` 参数，用 `h.SetCustomSignalWaiter` 替换 Hertz 默认信号等待逻辑；自定义函数同时监听 `ctx.Done()`（正常关闭信号）和 `errCh`（服务器内部错误），ctx 取消时返回 nil 触发 Hertz `Shutdown()` 优雅关闭（等待进行中请求完成，最多 5 秒超时），服务器错误时返回 err 触发强制退出
- E.7 资源清理顺序：收到信号后 ctx 取消 → sessionManager 清理 goroutine 因 `ctx.Done()` 退出 → httpapi.Run 的 SetCustomSignalWaiter 返回 nil → Hertz Shutdown 等待进行中请求完成 → Run 返回 → `defer mcpManager.Close()` 关闭 MCP 连接；HTTP 先于 MCP 关闭，避免进行中的请求因 MCP 连接先关而失败
- E.7 可见结果：`go run ./cmd/agenthub serve` 启动后按 Ctrl+C，日志依次输出 `shutdown signal received, gracefully stopping HTTP server...` → `Begin graceful shutdown, wait at most 5s` → `Execute OnShutdownHooks finish` → `close MCP servers`；进行中的流式请求在 5 秒超时内未完成时被强制关闭（curl 报 18），这是优雅关闭的正常超时行为
- E.7 验证：`go fmt ./...`、`go vet ./...`、`go build ./...`、`go test ./... -count=1` 全部通过；全项目 20+ 个测试包全部 ok
- E.7 理解验收：能说明 Unix 信号（SIGINT/SIGTERM/SIGKILL）的区别与可捕获性、signal.NotifyContext 把信号转换为 context 取消的机制、Hertz SetCustomSignalWaiter 返回 nil vs err 的语义（优雅关闭 vs 强制退出）、Shutdown 内部流程（关闭监听器→等待请求→关闭连接）、defer LIFO 执行顺序与资源清理顺序的设计原因、为什么 HTTP 必须先于 MCP 关闭
- F.1 数据模型：从当前 Session 结构反推两张表——`sessions`（id/last_access/created_at）和 `messages`（id/session_id/role/content/message_json/created_at）；`message_json` 存储完整 schema.Message 序列化，role/content 单独存便于调试；`ON DELETE CASCADE` 删除 session 时自动删消息；`TIMESTAMPTZ` 带时区避免时区问题
- F.1 PostgreSQL 接入：`cmd/agenthub/main.go` 用 `pgxpool.New` 创建连接池（本地 Unix socket 连接 `postgres:///agenthub?sslmode=disable`，peer authentication 用当前系统用户作为数据库用户，不需要密码），`dbPool.Ping` 健康检查，读取 `configs/schema.sql` 执行 `CREATE TABLE IF NOT EXISTS` 初始化表，`defer dbPool.Close()` 释放连接池
- F.1 SessionManager 改造：`internal/httpapi/session.go` 从内存 map 迁移到 PostgreSQL；删除 `sessions map` 和 `sync.Mutex`（数据库自己处理并发），新增 `db *pgxpool.Pool` 字段；`GetOrCreate` 用 `QueryRow` + `pgx.ErrNoRows` 判断会话是否存在，不存在则 `Exec` INSERT；`GetHistory` 用 `Query` + `rows.Next/Scan` 查询消息，子查询先 DESC 取最新 N 条再 ASC 排列，`json.Unmarshal` 反序列化 message_json；`Append` 用 `db.Begin` 事务保证用户消息和助手消息要么都写入要么都不写入，`defer tx.Rollback` + `tx.Commit` 标准模式；`cleanupExpired` 用 `Exec` DELETE + `NOW() - $1::interval`，`tag.RowsAffected()` 获取删除行数
- F.1 测试：新建 `internal/httpapi/testdb_test.go` 提供 `getTestDB(t)`（连接 `agenthub_test` 数据库，不可用时 `t.Skip`）和 `cleanupTestDB(t, db)`（TRUNCATE CASCADE 清空表）；重写 `session_test.go` 适配 PostgreSQL 版本（删除访问已删除内部字段的测试，用 SQL 直接更新 last_access 测试过期清理）；更新 `server_test.go` 所有 `NewSessionManager(20)` 为 `NewSessionManager(getTestDB(t), 20)`
- F.1 可见结果：发消息后 `psql -d agenthub -c "SELECT * FROM sessions;"` 和 `SELECT * FROM messages;` 能看到数据；重启服务后用相同 session_id 继续聊天，模型能记住之前的对话（如"我叫小明"→"我叫什么名字"→"你叫小明"）
- F.1 验证：`go fmt ./...`、`go vet ./...`、`go build ./...`、`go test ./... -count=1` 全部通过；全项目 18 个测试包全部 ok；httpapi 包 20+ 个测试全部通过（含并发测试、过期清理测试、多轮对话测试）
- F.1 理解验收：能说明连接池的作用（复用 TCP 连接、降低开销）、pgx 三种执行方式的区别（Query 多行/QueryRow 单行/Exec 不返回行）、`$1` 参数占位符防 SQL 注入、`pgx.ErrNoRows` 判断查询不到行、`rows.Next/Scan/Err` 迭代模式、事务的 ACID 含义和 `defer Rollback + Commit` 标准模式、`message_json` 存储完整 Message 的设计原因（避免拆字段、Eino 升级时不改表）、本地 Unix socket 连接不需要密码的原因（peer authentication）
- 下一节：**F.2 数据库迁移与连接池配置**
- 下一节只做：引入版本化迁移工具（golang-migrate），连接池配置（最大连接数、连接生命周期、健康检查），数据库错误分类与重试；不做多数据源、读写分离
- 下一节明确不做：Redis 缓存、消息队列、分布式锁、多租户、读写分离

## 9. 下一节理解验收题

F.1 的集中理解验收已通过：学习者能够说明连接池的作用（复用 TCP 连接、降低开销）、pgx 三种执行方式的区别（Query 多行/QueryRow 单行/Exec 不返回行）、`$1` 参数占位符防 SQL 注入、`pgx.ErrNoRows` 判断查询不到行、`rows.Next/Scan/Err` 迭代模式、事务的 ACID 含义和 `defer Rollback + Commit` 标准模式、`message_json` 存储完整 Message 的设计原因（避免拆字段、Eino 升级时不改表），以及本地 Unix socket 连接不需要密码的原因（peer authentication）。

## 10. 历史路线处理

原 Phase 0、Phase 1.1—1.7.3 的实现与测试均保留，不回退代码，也不否定已有学习成果。

它们的新定位是：

- **已实现的内部组件**，等待进入纵向链路；
- **后续复盘材料**，用于解释抽象和测试；
- **可重构对象**，若串联后发现抽象没有实际价值，可以在保持行为的前提下简化。

历史课程验收详情见 [`docs/learning-history.md`](docs/learning-history.md)。
