---
name: agenthub-learning-checkpoint
description: "AgentHub 教学项目的 Git 学习检查点流程。每当一节课、一个阶段或可独立验收的小目标完成时使用；负责验证实现、更新学习计划、检查敏感信息、仅暂存本节相关文件，并自动创建符合 Conventional Commits 的 Git 提交。"
---

# AgentHub Learning Checkpoint

课程节点完成后，执行验证、更新文档并自动提交。不得自动推送远端。

## 工作流

1. 运行本节适用的最小验证；失败时停止，不提交。
2. 更新 `AGENTHUB_PLAN.md` 的状态、验收记录和下一步。
3. 检查 `git status --short`、`git diff --stat` 和必要的文件 diff。
4. 确定本节相关文件清单，排除无关工作区改动。
5. 检查待提交内容，不得包含 `.env`、密钥、令牌、个人资料或其他敏感信息；发现风险时停止并说明原因。
6. 选择 Conventional Commits 类型：
   - `feat`: 新增能力
   - `fix`: 修复缺陷
   - `refactor`: 重构且不改变行为
   - `test`: 测试变化
   - `docs`: 仅文档变化
   - `chore`: 工程、依赖、构建或仓库维护
7. 使用英文 type、可选英文 scope 和简洁中文摘要生成提交主题；不加句号，建议不超过 72 个字符。
8. 仅暂存本节相关文件，使用 `git diff --cached --stat` 和必要的 cached diff 复核范围。
9. 自动执行 `git commit -m "<message>"`。
10. 报告提交哈希、提交主题、验证结果，以及仍留在工作区的无关改动。

## 边界

- 一个提交只包含一个清晰主题；多个主题应拆分提交。
- 不使用 `git add .`、`git add -A` 等可能混入无关改动的宽范围命令，除非已经确认整个工作区只包含本节内容。
- 不修改、还原或删除无关工作区改动。
- 不自动执行 `git push`、rebase、reset、amend 或 force 操作。
- 没有实际变更时不创建空提交。
