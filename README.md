# AgentHub

AgentHub 是一个用于学习企业级 Go 服务与 AI Agent 系统设计的渐进式项目。当前生产主线使用 [Eino](https://github.com/cloudwego/eino) 承担通用 Agent Runtime，AgentHub 负责产品入口、配置、用例、会话、持久化、安全和产品级观测。

## 当前可见能力

正式入口目前使用固定教学 Model 和天气 Tool，演示 Eino ReAct 工具调用、流式输出与 Callback：

```bash
go run ./cmd/agenthub
```

当前仍是教学闭环，尚未接入真实模型、交互式 CLI、HTTP/SSE 或持久化。详细路线见 [`AGENTHUB_PLAN.md`](AGENTHUB_PLAN.md)。

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
