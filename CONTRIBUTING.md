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

## 💻 开发规范

## 📝 提交规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范来保持提交历史的清晰和一致性。

### 格式

```
<type>(<scope>): <subject>
<body>
<footer>
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

表示影响范围，例如：

- `auth` - 认证相关
- `api` - API 接口
- `frontend` - 前端
- `backend` - 后端
- `database` - 数据库
- `hardware` - 硬件相关

### Subject（必需）

简短描述（不超过 72 个字符）

- 使用祈使语气（"add" 而非 "added"）
- 首字母小写
- 不加句末标点

### Body（可选）

详细描述变更内容、原因和实现方式

### Footer（可选）

- `Closes #123` - 关联并关闭 Issue
- `BREAKING CHANGE:` - 破坏性变更说明

### 示例

```bash
# ✅ 简洁的提交
feat(auth): add password reset functionality

# ✅ 带详细说明的提交
feat(api): add user profile endpoint

Implement GET /api/users/:id endpoint to retrieve user profiles.
Includes caching for improved performance.

Closes #123

# ✅ 破坏性变更
feat(api)!: change user response format

The user API response now includes additional fields.
Existing clients will need to update their parsing logic.

BREAKING CHANGE: user response format has changed
```

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
   - 维护者会检查：
     - 代码质量和可读性
     - 是否符合项目规范
     - 是否有潜在问题
     - 测试覆盖率是否足够

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

## 💻 开发规范

### 后端代码规范

```go
// 使用有意义的命名和错误处理
func GetUserByID(ctx context.Context, userID string) (*model.User, error) {
    user, err := repo.FindUserByID(ctx, userID)
    if err != nil {
        logger.Error("获取用户失败", zap.String("user_id", userID), zap.Error(err))
        return nil, err
    }
    return user, nil
}

// 使用有意义的变量名
totalDuration := 0
for _, s := range sessions {
    totalDuration += s.Duration
}

// 添加必要的注释
// AnalyzeTranscript 分析语音转录文本，提取结构化信息
func (s *AIService) AnalyzeTranscript(ctx context.Context, transcript, identityID string) (*ExtractedData, error) {
    // ...
    return data, nil
}
```

### 前端代码规范（React/TypeScript）

```tsx
// 使用函数组件和 TypeScript 类型
interface IdentityCardProps {
  identity: Identity;
  onTap: (id: string) => void;
}

export const IdentityCard: React.FC<IdentityCardProps> = ({ identity, onTap }) => {
  return (
    <div className="identity-card" onClick={() => onTap(identity.id)}>
      {/* ... */}
    </div>
  );
};

// 格式化
npm run format
```

### 硬件代码规范（C++）

```cpp
// 使用驼峰命名
class AudioCapture {
public:
    bool init();
    size_t read(uint8_t* buffer, size_t size);
    
private:
    bool initialized;
    int sampleRate;
};

// 添加注释
/**
 * 初始化 I2S 音频采集
 * @return 成功返回 true，失败返回 false
 */
bool AudioCapture::init() {
    // ...
}
```

## 测试

### 后端测试

```bash
cd backend
go test ./...
```

### 前端测试

```bash
cd frontend
npm test
```

### 硬件测试

- 在真实硬件上测试
- 检查串口输出
- 验证音频质量

## 文档

- 如果添加了新功能，更新相关文档（`docs/` 目录）
- 如果修改了 API，更新 `docs/api.md`
- 如果修改了 Prompt，更新 `docs/ai-prompts.md`

## Code Review

你的 Pull Request 会经过以下检查：

- [ ] 代码风格符合规范
- [ ] 通过所有测试
- [ ] 没有引入新的 Warning
- [ ] 文档已更新（如果需要）
- [ ] Commit 消息清晰

---

## 🌟 社区准则

### 我们的承诺

为了营造一个开放和友好的环境，我们承诺让参与 Pulse 项目的每个人都享有无骚扰的体验，无论年龄、体型、残疾、民族、性别特征、性别认同与表达、经验水平、教育程度、社会经济地位、国籍、个人外貌、种族、信仰或性取向如何。

### 我们的标准

**积极行为包括：**

- ✅ 使用友好和包容的语言
- ✅ 尊重不同的观点和经验
- ✅ 优雅地接受建设性批评
- ✅ 关注对社区最有利的事情
- ✅ 对其他社区成员表示同理心

**不可接受的行为包括：**

- ❌ 使用性化的语言或图像，以及不受欢迎的性关注或示好
- ❌ 恶意攻击、侮辱/贬损的评论，以及人身攻击或政治攻击
- ❌ 公开或私下的骚扰
- ❌ 未经明确许可发布他人的私人信息
- ❌ 其他在专业场合可能被认为不恰当的行为

###  enforcement

项目维护者有责任阐明可接受行为的标准，并应对任何不可接受的行为采取适当和公平的纠正措施。

---

## ❓ 获取帮助

如果你需要帮助，可以通过以下方式联系我们：

- **Discord**：[加入我们的 Discord](https://discord.gg/pulse) - 实时聊天和讨论
- **邮件**：[dev@pulse.example.com](mailto:dev@pulse.example.com) - 正式问题和技术支持
- **文档**：[https://docs.pulse.example.com](https://docs.pulse.example.com) - 官方文档和 API 参考
- **GitHub Issues**：[提交 Issue](https://github.com/yourusername/pulse/issues) - Bug 报告和功能请求

### 常见问题

**Q: 如何开始贡献？**
A: 查看 [Good First Issue](https://github.com/yourusername/pulse/labels/good%20first%20issue) 标签，选择感兴趣的任务开始！

**Q: 我遇到了 Bug，应该怎么做？**
A: 先搜索是否已有相关 Issue，如果没有，请[创建 Bug Report](https://github.com/yourusername/pulse/issues/new?template=bug_report.md)。

**Q: 如何提出新功能建议？**
A: 先[创建 Feature Request](https://github.com/yourusername/pulse/issues/new?template=feature_request.md) 与我们讨论。

**Q: 我的 PR 多久会被审查？**
A: 我们通常在 3-5 个工作日内审查 PR。复杂的功能可能需要更长时间。

---

## 📜 许可证

通过贡献你的代码，你同意你的贡献将在 [LICENSE](LICENSE) 下发布。

---

再次感谢你的贡献！🎉

你的参与让 Pulse 变得更好！💪
