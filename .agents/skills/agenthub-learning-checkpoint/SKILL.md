---
name: agenthub-learning-checkpoint
description: "AgentHub 教学项目的 Git 学习检查点流程。每当一节课、一个阶段或可独立验收的小目标完成时使用；负责核对验证结果、更新学习计划、检查待提交变更，并生成 Conventional Commits 风格的 commit message，同时提醒学习者亲自提交。"
---

# AgentHub Learning Checkpoint

在课程节点完成时执行一致的收尾流程。默认只检查并建议，不替学习者执行 `git add`、`git commit` 或 `git push`，除非学习者明确要求。

## 工作流

1. 确认本节课的验收命令已成功运行；未通过时不要建议提交。
2. 更新项目的学习计划：标记完成项，并写明当前进度和下一步。
3. 读取 `git status --short` 与 `git diff --stat`；必要时读取相关 diff，识别变更主题。
4. 检查敏感内容：不得提交 `.env`、密钥、令牌、邮箱、姓名、单位或其他不必要的个人信息。
5. 按 Conventional Commits 生成一条建议提交信息：
   - `feat`: 新增用户可见能力
   - `fix`: 修复缺陷
   - `refactor`: 重构且不改变功能
   - `test`: 新增或调整测试
   - `docs`: 仅文档变化
   - `chore`: 工程、依赖、构建或仓库维护
6. 提交主题使用英文 type，可选英文 scope，中文简洁描述；使用祈使/结果表达，不加句号，建议不超过 72 个字符。
7. 向学习者展示：本节完成内容、验证结果、待提交文件摘要、建议 commit message，以及可自行执行的命令。

## 建议输出格式

```text
本节课已通过验收，可以建立 Git 检查点。

变更摘要：
- ...

建议 commit message：
<type>(<scope>): <中文摘要>

请确认变更后自行执行：
git add <相关文件>
git commit -m "<message>"
```

## 边界

- 一个 commit 只包含一个清晰主题；存在多个无关主题时建议拆分。
- 不为了凑提交而跳过测试或文档更新。
- 不自动推送远端。
- 发现敏感信息时先停止提交建议，提示清理并重新检查。
