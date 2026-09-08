# AgentHub

AgentHub 是一个用于学习企业级 Go 服务与 AI Agent 系统设计的渐进式项目。当前生产主线使用 [Eino](https://github.com/cloudwego/eino) 承担通用 Agent Runtime，AgentHub 负责产品入口、配置、用例、会话、持久化、安全和产品级观测。

## 当前可见能力

正式入口已接入 OpenAI 兼容模型，使用 Eino ReAct Agent 完成真实模型工具调用、流式输出与 Callback 观察：

```bash
go run ./cmd/agenthub
```

当前输入仍是固定的天气问题；交互式多轮 CLI、HTTP/SSE 和持久化将在后续课程实现。详细路线见 [`AGENTHUB_PLAN.md`](AGENTHUB_PLAN.md)。

## 本地模型配置

复制示例文件并填写自己的兼容接口配置：

```bash
cp .env.example .env
```

```dotenv
AGENTHUB_MODEL_API_KEY=your-api-key
AGENTHUB_MODEL_BASE_URL=https://your-provider.example/v1
AGENTHUB_MODEL_NAME=your-tool-calling-model
```

程序启动时会可选加载项目根目录的 `.env`；文件不存在时继续使用系统环境变量。已经由终端、容器或部署平台设置的环境变量优先，不会被 `.env` 覆盖。

`.env` 已被 Git 忽略，不得提交真实密钥。所选模型必须支持 OpenAI Chat Completions、Tools/Function Calling 和 Streaming。

## 目录定位

- `cmd/agenthub/`：唯一正式应用入口；后续生产能力从这里演进。
- `examples/`：Phase A、B 的可运行学习对照，不继续承载生产功能。
- `internal/`：私有代码。现有 `agent`、`agentfactory`、`llm`、`memory`、`tokenizer`、`tool` 等自研 Runtime 包只为学习对照和早期实验保留，正式入口不依赖它们。
- `docs/`：产品蓝图、架构决策和长期学习图示。
- `configs/`：应用配置。

示例的用途和运行方式见 [`examples/README.md`](examples/README.md)。生产 Runtime 的选择依据见 [`ADR-0001`](docs/decisions/0001-use-eino-for-production-runtime.md)。

## 常用验证

```bash
go test ./...
go vet ./...
go build ./...
```

项目不会自动执行 `git push`。
