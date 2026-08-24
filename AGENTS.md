# AgentHub 项目协作指南

## 项目目标

通过持续实现 AgentHub，学习企业级 Go 服务与 AI Agent 系统设计。课程路线和进度以 `AGENTHUB_PLAN.md` 为准。

## 工作方式

- 每次只推进一个可运行、可验证的小目标。
- 先解释目标、设计和关键语法，再由学习者编写核心代码。
- 除非明确要求，不直接完成核心实现。
- 出现报错时先定位当前问题，不顺带重构无关代码。
- 不在仓库中记录对话内容、个人资料或能力画像。

## 课程完成条件

一节课只有同时满足以下条件才算完成：

1. 代码已实现。
2. 格式化、测试、静态检查或构建中的适用项已通过。
3. 学习者能解释本节关键概念。
4. `AGENTHUB_PLAN.md` 已更新状态、验收记录和下一步。
5. 已按仓库内 `.agents/skills/agenthub-learning-checkpoint/SKILL.md` 执行 Git 检查点流程。

## Git 规则

- 使用 Conventional Commits。
- 每节课验收通过并更新学习计划后，自动暂存本节相关文件并创建提交。
- 提交前检查暂存范围，只包含当前课程的单一主题；无关改动留在工作区。
- 不自动执行 `git push`。
- 提交前检查 `.env`、密钥、令牌和个人信息；发现风险时停止提交并说明原因。

## 常用命令

```bash
go fmt ./...
go test ./...
go vet ./...
go build ./...
go run ./cmd/agenthub
```

按当前课程选择最小且有效的验证命令；不要为了形式运行无关检查。

## 项目约定

- 应用入口：`cmd/agenthub/`
- 私有业务代码：`internal/`
- 配置文件：`configs/`
- 项目脚本：`scripts/`
- 项目文档：`docs/`
- 学习计划：`AGENTHUB_PLAN.md`
- 项目级 Skills：`.agents/skills/`
- 敏感配置不得提交；`.env` 必须保持在 `.gitignore` 中。
