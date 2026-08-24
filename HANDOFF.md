# Go Agent 学习项目 · 智能体交接文档（v2）

> 本文档用于交接给后续智能体，使其无需重新询问背景即可继续带用户学习 Go Agent 开发。
> 最后更新：2026-08-24

---

## 1. 交接目的

本文件不是普通课程大纲，而是给"下一个智能体"的完整上下文。后续智能体应直接沿当前路线继续，不应重新从 Go 基础语法开始，也不要一上来让用户使用 Eino、LangGraph、OpenAI SDK 等高层框架。

**核心教学原则**：先亲手写 Runtime，再学框架。

---

## 2. 用户状态

| 项目 | 状态 |
|---|---|
| Go 基础水平 | 掌握基础语法（变量、if/for、struct、func），但进阶语法（interface、指针接收者、错误包装、类型断言、goroutine/channel、defer、make）需要在实战中讲解 |
| 学习方式 | 项目驱动，一步一步实操，用户自己写代码，智能体给提示和讲解 |
| 当前进度 | Lesson 1-9 已完成，Lesson 10（Coding Agent）进行中 |
| 项目位置 | `/Users/bytedance/Desktop/study/go-agent` |
| 模型配置 | 通过 `.env` 文件配置 `LLM_BASE_URL`、`LLM_API_KEY`、`LLM_MODEL`，使用 OpenAI 兼容接口 |

---

## 3. 课程总览与当前进度

| 阶段 | 课程 | 状态 | 版本目标 |
|---|---|---|---|
| 01 | HTTP + LLM Client | ✅ 完成 | v0.1 |
| 02 | Model 抽象 + Tool Calling 数据结构 | ✅ 完成 | v0.2 |
| 03 | Tool Registry（真正执行工具） | ✅ 完成 | v0.3 |
| 04 | Agent Loop（循环调用工具） | ✅ 完成 | v0.4 |
| 05 | Memory / Session（多轮对话记忆） | ✅ 完成 | v0.5 |
| 06 | Streaming / Retry（流式输出 + 重试） | ✅ 完成（流式在自研 Client 层实现，Eino 适配器暂未实现流式） | v0.6 |
| 07 | File / Code Tools（文件读取、目录列表、代码搜索） | ✅ 完成 | v0.7 |
| 08 | MCP（Model Context Protocol，外置工具） | ✅ 完成 | v0.8 |
| 09 | Eino 重构（用 Eino 模型替换自研 Client） | ✅ 完成 | v0.9 |
| 10 | Coding Agent（整合所有组件，代码分析助手） | 🔄 进行中 | v1.0 |

---

## 4. 项目结构

```
go-agent/
├── .env                          # 环境变量（LLM_BASE_URL, LLM_API_KEY, LLM_MODEL），已 gitignore
├── .gitignore
├── go.mod                        # module: go-agent, go 1.25.0
├── go.sum
├── main.go                       # 入口：整合所有组件，交互式对话
├── mcp-server                    # 编译后的 MCP Server 可执行文件（已 gitignore）
├── HANDOFF.md                    # 本文档
├── cmd/
│   └── mcp-server/
│       └── main.go               # 最简单的 MCP Server，提供 echo 工具
└── internal/
    ├── agent/
    │   ├── agent.go              # Agent 核心：Run() 方法实现 Agent Loop
    │   └── session.go            # Session + SessionManager：多轮对话记忆
    ├── llm/
    │   ├── client.go             # 自研 LLM Client：HTTP 调用、Retry、Streaming
    │   ├── model.go              # Model interface 定义
    │   ├── tool.go               # Tool / ToolCall / Function / ToolCallFunction 数据结构
    │   └── eino_adapter.go       # Eino 适配器：把 Eino 的 openai.ChatModel 包装成 llm.Model 接口
    ├── mcp/
    │   └── client.go             # MCP Client：stdio 通信、JSON-RPC 2.0、工具调用
    └── tool/
        ├── tool.go               # Tool interface 定义
        ├── registry.go           # Tool Registry：注册、查找、导出为 llm.Tool
        ├── time.go               # TimeTool：获取当前时间
        ├── file.go               # ListDirTool / ReadFileTool / SearchCodeTool：文件操作
        └── mcp.go                # MCPTool 适配器：把 MCP 工具包装成 tool.Tool 接口
```

---

## 5. 各模块详细说明

### 5.1 `internal/llm/` —— LLM 层

#### `client.go`（自研 Client，已被 Eino 适配器替代但仍保留）
- `Message` 结构体：`Role`、`Content`、`ToolCalls`、`ToolCallID`
- `Client` 结构体：`BaseURL`、`APIKey`、`Model`、`HTTPClient`、`MaxRetries`
- `Chat(ctx, messages, tools) (Message, error)`：HTTP POST 调用 `/chat/completions`，支持 Retry（指数退避）
- `ChatStream(ctx, messages, tools) (<-chan StreamChunk, error)`：SSE 流式输出，goroutine + channel
- 错误处理：非 2xx 状态码把 body 带进 error

#### `model.go`
- `Model` interface：`Chat()` 和 `ChatStream()` 两个方法
- `*Client` 和 `*EinoModelAdapter` 都满足这个接口

#### `tool.go`
- `Tool`、`Function`、`ToolCall`、`ToolCallFunction`、`chatRequest`、`chatResponse`、`StreamChunk` 等数据结构

#### `eino_adapter.go`（当前使用的模型实现）
- `EinoModelAdapter` 结构体：持有 `*openai.ChatModel`（Eino 的模型）
- `NewEinoModelAdapter(ctx, baseURL, apiKey, modelName)`：创建 Eino 模型
- `Chat(ctx, messages, tools)`：适配器核心，做 4 种转换：
  1. 消息转换：`[]Message` → `[]*schema.Message`（`toEinoMessages`）
  2. 工具转换：`[]Tool` → `[]*schema.ToolInfo`（`toEinoTools` + `jsonSchemaToParams`）
  3. 调用 Eino 的 `Generate(ctx, messages, opts...)`
  4. 结果转换：`*schema.Message` → `Message`（`toOurMessage`）
- `ChatStream`：暂未实现，返回错误
- **设计原因**：Eino 的接口（`Generate`、`schema.Message`、`WithTools` option）和我们自己的接口不兼容，用适配器模式在不改变上层架构的前提下替换模型实现

### 5.2 `internal/tool/` —— 工具层

#### `tool.go`
- `Tool` interface：`Name()`、`Description()`、`Parameters()`、`Execute()` 四个方法

#### `registry.go`
- `Registry` 结构体：`map[string]Tool`
- `Register(tool)`、`Get(name)`、`ToLLMTools()`（导出为 `[]llm.Tool` 发给模型）

#### `time.go`
- `TimeTool`：`get_current_time`，无参数，返回格式化时间

#### `file.go`
- `ListDirTool`：`list_directory`，列出目录内容
- `ReadFileTool`：`read_file`，读取文件内容（暂无大小/行数限制）
- `SearchCodeTool`：`search_code`，遍历目录搜索关键词，跳过隐藏目录和非文本文件
- `safePath(workDir, userPath)`：路径安全校验，防止路径穿越（`../../etc/passwd`）

#### `mcp.go`
- `MCPTool` 适配器：把 MCP 的工具（`mcp.ToolInfo` + `mcp.Client`）包装成 `tool.Tool` 接口
- `Execute()` 内部调用 `mcp.Client.CallTool(name, arguments)`

### 5.3 `internal/agent/` —— Agent 层

#### `agent.go`
- `Agent` 结构体：`model llm.Model`、`registry *tool.Registry`、`maxSteps int`
- `Run(ctx, session, userInput) (string, error)`：Agent Loop 核心
  1. 把用户输入加入 session
  2. 循环（最多 maxSteps 次）：
     - 调用 `model.Chat(ctx, session.Messages, tools)`
     - 把 assistant 消息加入 session
     - 如果没有 `ToolCalls`，返回最终回答
     - 否则遍历 `ToolCalls`，从 Registry 找工具执行，把 tool 结果加入 session
  3. 超过 maxSteps 报错
- `executeTool(ctx, tc)`：执行单个工具，工具不存在或出错时把错误信息作为结果返回（让模型自己处理）

#### `session.go`
- `Session` 结构体：`ID`、`Messages []llm.Message`
- `SessionManager`：`map[string]*Session`，`GetOrCreate(id)`
- `AddMessage(message)`：追加消息到历史

### 5.4 `internal/mcp/` —— MCP 层

#### `client.go`
- `Client` 结构体：`cmd *exec.Cmd`、`stdin io.WriteCloser`、`stdout *bufio.Scanner`、`mu sync.Mutex`、`nextID int`、`pending map[int]chan json.RawMessage`
- `Start(ctx, command, args...)`：启动子进程，建立 stdin/stdout 管道，启动 goroutine 读 stdout
- `Initialize()`：MCP 初始化握手（initialize 请求 + notifications/initialized 通知）
- `ListTools()`：调用 `tools/list`，返回 `[]ToolInfo`
- `CallTool(name, arguments)`：调用 `tools/call`，返回工具结果文本
- `Close()`：关闭 stdin，等待子进程退出
- `readLoop()`：后台 goroutine，逐行读 stdout，解析 JSON-RPC 响应，通过 pending map 匹配请求并发送到 channel
- `sendRequest(method, params)`：生成请求 ID，存入 pending map，写入 stdin，阻塞等待 channel 返回
- 核心并发模型：goroutine 读响应 + channel 传结果 + Mutex 保护 pending map

### 5.5 `cmd/mcp-server/main.go` —— MCP Server
- 最简单的 MCP Server，提供 `echo` 工具
- 从 stdin 读 JSON-RPC 请求，处理 `initialize`、`tools/list`、`tools/call`、`notifications/initialized`
- 通过 stdout 返回 JSON-RPC 响应

### 5.6 `main.go` —— 入口
- 加载 `.env` 配置
- 创建 Eino 模型适配器
- 创建 Tool Registry，注册 TimeTool、ListDirTool、ReadFileTool、SearchCodeTool
- 启动 MCP Server 子进程，初始化，列出工具，包装后注册到 Registry
- 创建 Agent（maxSteps=10）
- 创建 SessionManager，获取 session
- **注意**：System Prompt 尚未加入（Lesson 10 待完成项）
- 交互式循环：读取用户输入，调用 `a.Run()`，打印回答

---

## 6. 环境配置与运行

### 环境变量（`.env` 文件）
```
LLM_BASE_URL=你的 OpenAI 兼容接口地址
LLM_API_KEY=你的 API Key
LLM_MODEL=模型名
```

### 编译 MCP Server
```bash
go build -o mcp-server ./cmd/mcp-server
```

### 运行
```bash
go run .
```

### 常用验证命令
```bash
go fmt ./...
go vet ./...
go build ./...
go mod tidy
```

---

## 7. 教学原则（后续智能体必须遵守）

1. **不要重新问用户"会不会 Go 基础"** —— 已知答案是会基础语法，但进阶语法需要在实战中讲解
2. **不要从变量、if、for 开始重新授课**
3. **每次只推进一个可验证的小目标**，不要一次把整个项目生成完
4. **解释"为什么这么设计"**，尤其是 interface、context、error wrapping、适配器模式、并发模型
5. **用户贴报错时，优先定位当前报错**，不要借机大改架构
6. **练习阶段优先使用提示 → 用户尝试 → 再解释**；核心课程代码可以给完整示例，但必须解释关键行
7. **每节课结束前给出明确验收项**：命令、预期输出、用户需要能解释的概念
8. **保持项目持续演进**，不要每课重新创建独立 Demo
9. **不要过早引入 Eino、MCP、Multi-Agent、RAG、向量数据库** —— 它们应该在用户理解 Agent Loop 后再出现（MCP 和 Eino 已学）
10. **用户说"不需要帮我编辑，我要自己学习"时，必须停止直接编辑代码，改为给提示和讲解**
11. **用户基础语法不熟悉时，要耐心讲解，不要假设用户懂**

---

## 8. Lesson 10（当前进行中）待完成项

Lesson 10 目标：整合所有组件，做一个能分析代码项目的完整 Coding Agent。

### 已完成
- ✅ 所有组件已整合到 main.go（模型、工具、MCP、Agent、Session）
- ✅ 交互式对话循环
- ✅ 文件工具（list_directory、read_file、search_code）

### 待完成
1. **加入 System Prompt** —— 在 session 创建后加入一条 `role: "system"` 的消息，定义 Agent 角色（Go 代码分析助手）、工具列表、分析流程、回答要求
2. **加入欢迎信息** —— 启动时打印 Agent 能力说明
3. **优化 read_file 输出** —— 可选：限制文件大小或行数，防止大文件导致 context 溢出
4. **测试代码分析能力** —— 测试"分析项目结构"、"读某个文件"、"搜索代码"、"分析执行流程"等场景
5. **课程总结** —— 回顾整个学习路径，总结核心概念

### System Prompt 参考模板
```
你是一个专业的 Go 代码分析助手，擅长阅读和分析 Go 项目代码。

你拥有以下工具：
- list_directory: 列出指定目录下的文件和子目录
- read_file: 读取指定文件的完整内容
- search_code: 在项目中搜索关键词，返回匹配的文件名和行号
- get_current_time: 获取当前时间
- echo: 回显输入的文本（测试用）

分析代码时请遵循以下流程：
1. 先用 list_directory 了解项目整体结构
2. 再用 read_file 读取关键文件的内容
3. 需要定位特定代码时用 search_code 搜索
4. 基于实际读到的代码内容回答，不要猜测或编造

回答要求：
- 简洁准确，基于代码事实
- 引用代码时注明文件名和行号
- 如果需要读取多个文件才能回答，分步读取
```

---

## 9. 已知问题与技术债务

1. **Eino 适配器未实现流式输出** —— `ChatStream()` 直接返回错误，自研 `Client` 的流式实现仍保留但未使用
2. **read_file 无大小限制** —— 大文件可能导致 context 溢出，建议加行数限制（如前 200 行）
3. **MCP Server 只有 echo 工具** —— 可扩展更多工具，或接入真实的 MCP Server（如 filesystem、github）
4. **自研 `llm.Client` 已被 Eino 适配器替代但仍保留在代码中** —— 可作为参考保留，或在确认 Eino 稳定后移除
5. **Retry 只在自研 Client 中实现** —— Eino 模型的重试依赖 Eino 框架本身
6. **Session 无持久化** —— 程序重启后对话历史丢失，可扩展文件持久化
7. **无单元测试** —— 整个项目没有测试代码

---

## 10. 核心概念清单（用户应能独立回答）

完成整个课程后，用户应能独立回答：

- [x] Model、Message、Tool、ToolCall、ToolResult 分别是什么？
- [x] Tool Calling 为什么不是"模型自己执行工具"？
- [x] Agent Loop 如何决定继续调用工具还是返回最终答案？
- [x] 为什么需要 maxSteps、context cancellation、timeout、retry？
- [x] 短期 Memory、Session、持久化 Memory 的区别是什么？
- [x] Streaming 为什么会涉及 channel / goroutine / select？
- [x] MCP 在本质上替代了 Agent 与工具之间的哪一层连接？
- [x] Eino 等框架究竟替自己实现的 Runtime 封装了什么？
- [x] 适配器模式解决了什么问题？（Eino 适配器、MCPTool 适配器）
- [x] 为什么需要 pending map + channel？（MCP Client 的请求-响应匹配）
- [x] goroutine、channel、sync.Mutex 各自的作用是什么？

---

## 11. 整门课程的核心心智模型

遇到任何 Agent 框架时，都先找这 8 个东西：

1. **Model 在哪？** —— 调用 LLM 的抽象
2. **Message 在哪？** —— 对话消息的数据结构
3. **Tool 在哪？** —— 工具的定义和执行
4. **Agent Loop 在哪？** —— 循环调用模型和工具的核心
5. **State / Context 在哪？** —— 会话状态和上下文
6. **Memory 在哪？** —— 对话历史的存储和管理
7. **Workflow 在哪？** —— 多步骤的编排
8. **Runtime 在哪？** —— 真正执行的运行时

框架会变，但这些概念不会消失。

---

## 12. 可直接复制给新智能体的接手提示

```
请读取这份《Go Agent 学习项目·智能体交接文档 v2》（HANDOFF.md），直接继续当前课程，不要重新从 Go 基础语法开始。

用户会 Go 基础语法，但进阶语法（interface、指针接收者、错误包装、类型断言、goroutine/channel、defer、make）需要在实战中讲解。用户希望自己动手写代码，智能体给提示和讲解，不要直接替用户编辑代码。

当前进度：Lesson 1-9 已完成，Lesson 10（Coding Agent）进行中。当前正在做 System Prompt 加入和代码分析能力测试。

请采用项目驱动、一步一步实操的方式教学。每一步告诉我：做什么、为什么、代码放哪里、如何运行、预期结果、常见错误。完成当前步骤后再继续下一步。

最终目标是让用户亲手实现了一个完整的 Go Agent Runtime，包括 Tool Calling、Agent Loop、Memory、Streaming、MCP，并在理解底层后学习了 Eino 框架。
```

---

—— 交接终点：下一位智能体应从 Lesson 10 的 System Prompt 加入继续 ——
