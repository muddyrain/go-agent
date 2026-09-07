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
- Tool Definition、Registry、Executor、Memory、Factory 和 Agent Loop 的首次纵向串联。

当前仍没有：

- 可交互的多轮终端输入；
- 真实模型和真实 Token Usage；
- HTTP 服务、聊天接口或 SSE 输出；
- 真实 MCP SDK/传输连接。

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
| 配置、日志、应用错误 | 已实现 | 已进入入口，可见启动日志 | 在终端 Agent 入口中复用并解释 |
| Message / Model 协议 | 已实现 | 自研与 Eino 的直接回答、工具闭环均已进入独立入口，可对照消息与模型接口 | B.3 观察 Eino 流式与回调，Phase C 接真实模型 |
| Tool / Registry / Executor | 已实现 | 自研天气工具与 Eino Tool/ToolsNode 均已进入入口，职责映射已完成 | Phase D 接真实 MCP |
| Agent Loop | 已实现 | 自研 Loop 与 Eino ReAct Agent 都已完成 Model—Tool—Model 闭环，并由纵向测试保护 | B.3 观察 Eino 流式和节点回调 |
| Memory | 已实现 | 已进入入口，但尚未观察裁剪效果；已识别按单消息裁剪可能拆散 ToolCall/ToolMessage | 在多轮 CLI 中展示并修复协议裁剪风险 |
| Streaming | 已实现 | 未连接终端或网络输出 | 先接终端，再接 SSE |
| Agent Factory | 已实现 | 已用于入口组装；两种 Memory 策略有价值，但存在阅读跳转成本 | Eino 对照时重新判断是否保留或简化 |
| MCP Session / Adapter / Manager | 已实现协议边界 | 无真实传输，未进入入口 | 延后到本地工具链跑通后再接回 |
| HTTP / SSE | 未实现 | 不可见 | 在 CLI Agent 稳定后推进 |
| 持久化 | 未实现 | 不可见 | 由“重启后会话丢失”这个问题驱动 |
| RAG / Workflow / Multi-Agent | 未实现 | 不可见 | 由具体用户场景驱动 |

详细历史验收记录移至 [`docs/learning-history.md`](docs/learning-history.md)，不再占用主路线的注意力。

## 6. 校准后的课程路线

完整的产品目标、系统边界、技术选型理由、阶段课程与验收标准见 [`docs/product-blueprint.md`](docs/product-blueprint.md)。主计划只保留阶段导航和当前进度，避免再次被大量细节淹没。

| 阶段 | 要解决的核心问题 | 用户可见成果 | Runtime 策略 | 状态 |
|---|---|---|---|---|
| Phase A：恢复可见主线 | 已有零件没有进入应用入口 | 终端跑通并解释一次 Model—Tool—Model 闭环 | 使用现有自研内核学习原理 | 已完成 |
| Phase B：Eino 对照实验 | 继续手写会重复建设，直接换框架又会形成黑盒 | 用 Eino 重做同一用例并完成概念对照 | 冻结自研内核，生产主线切到 Eino | 进行中：B.2 已掌握 |
| Phase C：可用 CLI Agent | 演示 Model 不能解决真实问题 | 真实模型、多轮对话、工具调用和流式终端 | Eino | 待开始 |
| Phase D：工具中心与 MCP | 本地工具难扩展，MCP 还没有真实连接 | 可发现、调用和诊断真实 MCP 工具 | Eino + AgentHub MCP 配置层 | 待开始 |
| Phase E：HTTP 与 Web Playground | CLI 无法被其他应用调用，产品形态不可见 | HTTP/SSE API 和最小聊天控制台 | AgentHub 应用层调用 Eino | 待开始 |
| Phase F：会话与配置持久化 | 重启后 Agent 和会话丢失 | Agent CRUD、历史会话恢复 | PostgreSQL + Repository | 待开始 |
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
| B.1 Eino 最小直接回答对照 | 直接切到 Eino 容易只会照抄框架代码 | 实现最小 `BaseChatModel`，用 `Chain → Compile → Runnable.Invoke` 执行一次请求 | `go run ./cmd/eino-direct` 输出固定 Assistant 回答 | 已掌握 |
| B.2 Eino 工具闭环 | 直接回答仍看不出 Eino 如何承担 Agent 循环 | 用 Eino Tool/ToolsNode/Agent 复现 A.2 天气工具链路 | 终端再次看到 ToolCall → ToolResult → FinalAnswer | 已掌握 |
| B.3 流式和回调观察 | 同步结果看不到生成过程和节点边界 | 用最小例子观察 Eino 流式输出与模型、工具节点回调 | 终端可区分增量输出和节点开始/结束 | 未开始 |

B.1—B.2 使用 `github.com/cloudwego/eino v0.9.19`；保留 `cmd/agenthub` 作为自研对照入口，分别新增 `cmd/eino-direct` 与 `cmd/eino-tool`。B.2 使用固定教学 Model 和天气 Tool，只验证 Eino ReAct 工具闭环，没有接真实模型、HTTP、MCP 或额外应用抽象。

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

- 当前阶段：**Phase B：Eino 对照实验进行中，B.2 已掌握**
- 路线蓝图：**已明确最终产品形态、Eino 切换边界、Phase A—I 交付与验收标准**
- 已掌握：**A.1 最小终端 Agent：直接回答、A.2 本地工具闭环、A.3 调用链复盘、A.4 最小入口测试、B.1 Eino 最小直接回答对照、B.2 Eino 工具闭环**
- A.1 可见结果：`go run ./cmd/agenthub` 输出启动信息、固定 Assistant 回答、`steps: 1` 和零值 Usage
- A.1 调用链：[`docs/images/a1-direct-answer-flow.svg`](docs/images/a1-direct-answer-flow.svg)
- A.1 理解验收：能解释隐式接口实现与编译期检查的区别、Factory 创建 Memory 的职责、空 Registry 不妨碍直接回答，以及 `Steps` 表示模型调用次数
- A.2 可见结果：终端按顺序输出 `ToolCall`、`ToolResult`、最终回答和 `steps: 2`
- A.2 调用链：[`docs/images/a2-local-tool-loop.svg`](docs/images/a2-local-tool-loop.svg)
- A.2 理解验收：能区分 Definition 与 Handler、Registry 的模型侧与执行侧职责、Tool Name 与 ToolCall ID，并说明 Schema 失败不会进入 Handler，而会作为错误 ToolMessage 交给下一轮模型
- A.3 职责与价值图：[`docs/images/a3-runtime-boundaries.svg`](docs/images/a3-runtime-boundaries.svg)
- A.3 复盘结论：`main.go` 是组合根；Factory 只创建 Memory 并组装 Agent；`Agent.Run` 是同步薄入口，`runWithGenerator` 承担循环；Registry 是工具事实来源，Executor 执行已确定的 ToolCall，Model 决定调用什么工具，Agent Loop 推进历史和步骤
- A.3 抽象判断：Registry、Executor、Message 协议和 Agent Loop 已被真实闭环证明；Factory 有实际价值但阅读成本偏高；Memory 已进入链路但协议安全裁剪仍待验证；MCP、HTTP、持久化、RAG 等尚未被当前入口证明
- A.3 风险发现：`SimpleSliding` 按单条消息裁剪，`MaxKeep` 太小时可能保留孤立 ToolMessage，拆散 Assistant ToolCall 与 ToolMessage 的协议关联
- A.4 最小测试：`cmd/agenthub/main_test.go` 直接调用 `runDemoAgent`，保护真实 `demoModel`、天气 Tool、Registry、Factory 与 Agent Loop 的纵向组装
- A.4 稳定断言：四条消息按 User → Assistant ToolCall → Tool → Final Assistant 排列，Tool Name 与 Call ID 前后对应，工具结果进入最终回答，且 `Steps == 2`
- A.4 边界结论：`runDemoAgent` 隔离可验证的业务组装，`run` 保留配置、日志与终端展示；入口测试不重复包内单测已经覆盖的 Agent Loop 内部异常分支
- A.4 理解验收：能解释为何不直接测试混合配置和输出的 `run`、为何入口只保护跨组件闭环，以及 Call ID 与 Steps 分别证明调用关联和两轮模型执行
- B.1 可见结果：`go run ./cmd/eino-direct` 通过单节点 Eino Chain 输出 `assistant: 你好，我是 AgentHub 的 Eino 演示 Agent。`
- B.1 代码边界：`demo_model.go` 实现 `BaseChatModel` 的 Generate/Stream；`main.go` 负责创建 Chain、追加模型、Compile 并 Invoke，不修改自研入口
- B.1 类型流：`NewChain[[]*schema.Message, *schema.Message]` 声明整条流程接收消息列表并返回一条消息；Compile 把可修改的流程定义转换为 `Runnable`
- B.1 概念映射：自研 `llm.Message` 对应 `schema.Message`，自研 `llm.Model` 对应 `BaseChatModel`，自研同步运行入口对应 Chain 编排后的 `Runnable.Invoke`
- B.1 理解验收：能说明 Chain 定义数据流、Runnable 才可执行；接口方法集要求教学 Model 同时实现 Generate/Stream；使用 Chain 不是强制规范，而是为后续连接 Prompt、Tool、Retriever 提前掌握可扩展编排方式
- B.1 对照图：[`docs/images/b1-eino-direct-answer-flow.svg`](docs/images/b1-eino-direct-answer-flow.svg)
- B.1 已知环境提示：Sonic 在当前环境回退到 `encoding/json`，不影响本节行为，仅可能影响 JSON 性能
- B.2 可见结果：`go run ./cmd/eino-tool` 按顺序输出 User、ToolCall、ToolResult、Final Assistant 和 `steps: 2`
- B.2 代码边界：`demo_tool.go` 用 `InferTool` 包装参数 Schema 与 Handler；`demo_model.go` 实现 `ToolCallingChatModel`；`main.go` 用 `react.NewAgent` 组装工具闭环并用 `WithMessageFuture` 观察消息
- B.2 职责映射：Tool 的 `Info()` 供 Model 选择工具与生成参数，`InvokableRun()` 供 ToolsNode 执行；Eino ReAct Agent 承担自研 Agent Loop 的路由、历史追加与结束判断
- B.2 Steps 口径：两次 Model 产生 Assistant Message，工具绑定发生在创建阶段、工具执行发生在 ToolsNode，二者都不计 Step
- B.2 理解验收：已经能说明主流程和主要职责映射；当前不要求背熟 Eino API，`WithMessageFuture` 只负责旁路观察，不参与闭环执行
- B.2 最小测试：`cmd/eino-tool/main_test.go` 保护 ToolCall → ToolMessage → Final Assistant 的消息协议，并验证未绑定工具时 Model 明确拒绝
- B.2 调用链：[`docs/images/b2-eino-tool-loop.svg`](docs/images/b2-eino-tool-loop.svg)
- 暂停项：**原 1.7.4 MCP 超时、关闭、断线与重连**
- 下一节：**B.3 流式和回调观察**
- 下一节只做：在独立最小入口中观察 Eino 的流式增量，以及模型和工具节点的开始、结束、错误回调
- 下一节明确不做：真实模型、HTTP/SSE 服务、MCP、数据库、RAG、生产级可观测平台或额外封装

## 9. 下一节理解验收题

B.3 仍只进行一轮集中验收，只解决一个问题：同步 `Generate` 看不到的执行过程，Eino 的 Stream 和 Callback 分别怎样暴露。验收只要求能沿一次请求指出增量内容来自哪里、节点回调发生在何时，以及它们为什么不是两套 Agent Loop。

## 10. 历史路线处理

原 Phase 0、Phase 1.1—1.7.3 的实现与测试均保留，不回退代码，也不否定已有学习成果。

它们的新定位是：

- **已实现的内部组件**，等待进入纵向链路；
- **后续复盘材料**，用于解释抽象和测试；
- **可重构对象**，若串联后发现抽象没有实际价值，可以在保持行为的前提下简化。

历史课程验收详情见 [`docs/learning-history.md`](docs/learning-history.md)。
