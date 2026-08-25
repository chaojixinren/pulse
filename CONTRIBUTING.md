# 贡献指南

感谢你对 Pulse 项目的关注！我们欢迎任何形式的贡献。

## 📋 目录

- [贡献流程](#贡献流程)
- [开发规范](#开发规范)
- [提交规范](#提交规范)
- [代码审查](#代码审查)
- [社区准则](#社区准则)
- [获取帮助](#获取帮助)

---

## 🔄 贡献流程

我们使用标准的 **GitHub Flow with Feature Branch** 工作流程：

```mermaid
graph LR
    A[Fork 仓库] --> B[创建 feature/xxx 分支]
    B --> C[本地开发 + 测试]
    C --> D[提交代码<br/>(Conventional Commits)]
    D --> E[推送分支到 GitHub]
    E --> F[创建 Pull Request]
    F --> G[CI/CD 自动检查]
    G --> H[Code Review]
    H --> I{通过?}
    I -->|是| J[Squash Merge]
    I -->|否| C
    J --> K[删除功能分支]
```

### 1️⃣ Fork 仓库

1. 访问 [Pulse GitHub Repository](https://github.com/yourusername/pulse)
2. 点击右上角的 **Fork** 按钮
3. 克隆你的 Fork 到本地：

```bash
git clone https://github.com/<your-username>/pulse.git
cd pulse
```

### 2️⃣ 配置上游仓库

```bash
# 添加上游仓库
git remote add upstream https://github.com/<original-owner>/pulse.git

# 验证远程仓库
git remote -v
```

### 3️⃣ 创建功能分支

**分支命名规范：**

- `feature/xxx` - 新功能开发
- `fix/xxx` - Bug 修复
- `docs/xxx` - 文档更新
- `refactor/xxx` - 代码重构
- `chore/xxx` - 构建/工具变更

**示例：**

```bash
# 从 develop 分支创建
git checkout develop
git pull upstream develop

# 创建功能分支
git checkout -b feature/user-authentication

# 或创建 Bug 修复分支
git checkout -b fix/login-error-handling
```

### 4️⃣ 本地开发

在本地完成开发工作，确保：

- ✅ 遵循代码规范（见[开发规范](#开发规范)）
- ✅ 通过所有测试（后端 `make test`，前端 `npm test`）
- ✅ 通过 Lint 检查（后端 `make lint`，前端 `npm run lint`）
- ✅ 添加必要的测试用例

**Git Hooks（自动检查）：**

项目配置了 Husky，在 `git commit` 时会自动：

- 验证提交信息符合 [Conventional Commits](#提交规范)
- 运行 `make lint`（后端）
- 运行 `npm run lint && npm run type-check`（前端）

如果检查失败，提交将被阻止。

### 5️⃣ 提交代码

**提交信息格式（Conventional Commits）：**

```
<type>(<scope>): <subject>
```

**Type 类型：**

- `feat` ✨ - 新功能
- `fix` 🐛 - Bug 修复
- `docs` 📝 - 文档更新
- `style` 💄 - 代码格式（不影响功能）
- `refactor` ♻️ - 重构
- `perf` ⚡️ - 性能优化
- `test` ✅ - 测试相关
- `chore` 🔧 - 构建工具或辅助工具变更
- `ci` 🎡 - CI/CD 配置

**示例：**

```bash
# ✅ 好的提交信息
git commit -m "feat(auth): add OAuth2 authentication"
git commit -m "fix(api): handle nil pointer in user service"
git commit -m "docs(readme): update installation guide"

# ❌ 不好的提交信息
git commit -m "update"
git commit -m "fix bug"
git commit -m "修改代码"
```

**Scope（可选）：**

表示影响范围，例如：`auth`、`api`、`frontend`、`database`

**多次提交的建议：**

建议拆分提交，每个提交只做一件事：

```bash
git commit -m "feat(auth): add login endpoint"
git commit -m "feat(auth): add JWT token validation"
git commit -m "test(auth): add login endpoint tests"
```

### 6️⃣ 推送到 GitHub

```bash
# 推送到你的 Fork
git push origin feature/user-authentication
```

### 7️⃣ 创建 Pull Request

1. 访问你的 Fork 仓库页面
2. 点击 **"Compare & pull request"** 按钮
3. 填写 PR 模板（见 [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md)）
4. 提交 PR

**PR 标题格式：**

```
<type>(<scope>): <subject>
```

**示例：**

- `feat(auth): add OAuth2 authentication support`
- `fix(api): handle nil pointer in user service`
- `docs(readme): update installation guide`

### 8️⃣ Code Review

- 维护者会 review 你的代码
- 可能需要修改，重复 **步骤 4-7** 直到通过
- 当维护者评论 **"LGTM"**（Looks Good To Me）时，表示审查通过

### 9️⃣ Squash Merge

- 维护者会使用 **Squash and Merge** 合并你的 PR
- 这会保留一个清晰的提交历史
- 你的贡献会被自动添加到 changelog

### 🔟 清理分支

PR 合并后，删除你的功能分支：

```bash
# 删除本地分支
git branch -d feature/user-authentication

# 删除远程分支
git push origin --delete feature/user-authentication

# 同步上游代码
git checkout develop
git pull upstream develop
```

---

## 📝 提交规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范来保持提交历史的清晰和一致性。

### 格式

```
<type>(<scope>): <subject>
```

### Type（必需）

- `feat` ✨ - 新功能
- `fix` 🐛 - Bug 修复
- `docs` 📝 - 文档更新
- `style` 💄 - 代码格式（不影响功能）
- `refactor` ♻️ - 重构
- `perf` ⚡️ - 性能优化
- `test` ✅ - 测试相关
- `chore` 🔧 - 构建工具或辅助工具变更
- `ci` 🎡 - CI/CD 配置

### Scope（可选）

表示影响范围，例如：`auth`、`api`、`frontend`、`database`

### Subject（必需）

简短描述（不超过 72 个字符）

### 示例

```bash
# ✅ 好的提交信息
git commit -m "feat(auth): add OAuth2 authentication"
git commit -m "fix(api): handle nil pointer in user service"
git commit -m "docs(readme): update installation guide"

# ❌ 不好的提交信息
git commit -m "update"
git commit -m "fix bug"
git commit -m "修改代码"
```

---

## 👀 代码审查

### 审查流程

1. **CI/CD 自动检查**

   PR 提交后会触发以下检查：

   - **后端**：
     - ✅ Lint（gofmt + go vet）
     - ✅ 单元测试 & 集成测试
     - ✅ E2E 测试
     - ✅ Docker 镜像构建

   - **前端**：
     - ✅ ESLint 检查
     - ✅ TypeScript 类型检查
     - ✅ 单元测试
     - ✅ 构建验证

2. **人工审查**

   - 至少一位维护者需要审查代码
   - 维护者会检查代码质量、可读性、潜在问题等

3. **反馈与修改**

   - 根据审查意见修改代码
   - 推送新的 commits 到同一个 PR
   - 继续审查直到通过

4. **LGTM（Looks Good To Me）**

   - 当维护者评论 **"LGTM"** 时，表示代码审查通过
   - PR 可以被合并了

5. **Squash and Merge**

   - 维护者会使用 **Squash and Merge** 合并 PR
   - 将所有 commits 合并为一个干净的提交
   - 保持主分支历史整洁

### PR 合并前检查清单

在请求审查之前，请确认：

- [ ] 代码遵循项目规范
- [ ] 通过所有测试
- [ ] 通过所有 Lint 检查
- [ ] CI/CD 检查全部通过 ✅
- [ ] 提交信息符合 Conventional Commits 规范
- [ ] 已更新相关文档
- [ ] 在本地测试功能正常
- [ ] PR 描述清晰完整（使用模板）

---

## 🌟 社区准则