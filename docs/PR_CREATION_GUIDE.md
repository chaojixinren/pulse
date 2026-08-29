# 📝 创建 Pull Request 指南

## 🎯 推荐方式

### 方式一：GitHub CLI（最快捷）

#### 1. 安装 GitHub CLI

**Windows**:
```powershell
# 使用 winget
winget install --id GitHub.cli

# 或使用 Chocolatey
choco install gh
```

**Mac**:
```bash
brew install gh
```

**Linux**:
```bash
# 参考官方文档
https://cli.github.com/
```

#### 2. 登录 GitHub CLI

```bash
gh auth login
```

按照提示完成登录。

#### 3. 创建 PR

```bash
cd pulse
gh pr create \
  --title "docs(readme): 修正项目名称并优化内容" \
  --body-file docs/PR_FINAL.md \
  --base main \
  --head feature/frontend-style-config
```

---

### 方式二：浏览器（最直接）

直接访问以下链接创建 PR：

```
https://github.com/chaojixinren/pulse/pull/new/feature/frontend-style-config
```

**填写信息**：

1. **Title**（标题）:
   ```
   docs(readme): 修正项目名称并优化内容
   ```

2. **Description**（描述）:
   - 复制 `docs/PR_FINAL.md` 的完整内容
   - 或者直接粘贴以下内容（简化版）

3. **Labels**（标签）:
   - `documentation`
   - `enhancement`

4. **Reviewers**（审阅者）:
   - `@chaojixinren`（如果需要）

5. **Assignees**（指派人）:
   - 选择负责人

6. **Projects**（项目看板）:
   - 关联到对应的项目板

---

## 📋 PR 描述（完整版）

### 复制以下内容到 PR 描述

```markdown
## 📋 PR 概述

优化 Pulse 项目的 README.md，修正项目名称并丰富内容，提升项目的吸引力和专业感。

## ✅ 主要变更

### 1. 项目名称修正
- ✅ 修正：`时笺` → `拾笺`（共 10 处）

### 2. 标语升级
- **Before**: 通过语音记录生活，让 AI 帮你在不同身份间自由切换
- **After**: **拾起每一个瞬间 · 笺记每一段人生**

### 3. 内容优化

#### ✨ 情感化叙事
- 新增"为什么选择拾笺"章节
- 丰富"我们的理念"章节（原"核心理念"）
- 增加更多生活化的场景描述

#### 🎯 功能展示
- "核心功能" → "核心能力"
- 新增表格化信息提取展示
- 新增关键特性列表和金句引用

#### 🗺️ 使用场景（新增）
- 👔 职场人士 — 会议记录、客户沟通、项目跟进
- 👨‍👩‍👧 家庭生活 — 亲子时光、夫妻沟通、家庭决策
- 👥 社交达人 — 朋友聚会、社交网络、活动策划
- 🎯 自我提升 — 个人反思、习惯养成、时间管理

#### 💫 理念深化
- 🔐 隐私理念强化："隐私不是功能，而是权利"
- 🌱 科技哲学："技术在后台运行，生活在台前继续"
- 💡 金句引用："你只需要做自己，剩下的交给拾笺"

### 4. 技术文档优化
- 技术栈表格增加"用途"列
- 快速开始章节增加详细步骤
- 贡献指南增加完整流程

### 5. 品牌形象强化
- 新增品牌页脚区
- 增加社交媒体链接
- 增加"Made with ❤️ by Pulse Team"

## 📊 数据对比

| 指标 | 优化前 | 最终版 | 提升 |
|------|--------|--------|------|
| **总行数** | 144 行 | 548 行 | +280% |
| **项目名称** | ❌ 错误 | ✅ 拾笺 | 修正 |
| **使用场景** | 0 | 4 个 | +4 |
| **表格数量** | 1 | 5 个 | +400% |
| **代码块** | 4 | 16 个 | +300% |
| **图标** | 8 | 35+ | +337% |

## ✅ 质量检查

- ✅ 项目名称全部修正为"拾笺"
- ✅ 无临时文件残留
- ✅ Markdown 语法正确
- ✅ 所有内部链接有效
- ✅ Git 工作区干净
- ✅ 提交信息符合 Conventional Commits

## 🔗 相关资源

- [项目主页](https://github.com/chaojixinren/pulse)
- [文档目录](docs/)
- [贡献指南](CONTRIBUTING.md)
```

---

## 🔍 PR 创建后

### 在 GitHub 上完成以下操作：

1. **预览 README 渲染效果**
   - 在 PR 页面查看 "Files changed" 标签
   - 点击 "Conversation" 查看渲染效果

2. **邀请审阅者**
   - 点击 "Reviewers" → "Request review"
   - 选择团队成员

3. **关联 Issue**（如果有）
   - 点击 "Linked issues" → "Add"

4. **添加标签**
   - 点击 "Labels" 选择相关标签

5. **分配给负责人**
   - 点击 "Assignees" 选择负责人

6. **等待审阅**
   - 团队成员审阅后会添加评论
   - 根据反馈进行必要的修改

7. **合并 PR**
   - 审阅通过后，点击 "Merge pull request"
   - 选择合适的合并策略（建议 "Squash and merge"）

8. **删除分支**（可选）
   - PR 合并后，GitHub 会提示删除 feature 分支
   - 建议删除以保持仓库整洁

---

## 📝 PR 模板（简化版）

如果不需要完整的 PR 描述，可以使用简化版：

```markdown
## 变更概述

优化 README.md，修正项目名称并丰富内容。

## 主要变更

- ✅ 修正项目名称：`时笺` → `拾笺`
- ✨ 优化内容结构，增加情感化叙事
- 🗺️ 新增使用场景章节
- 📊 优化技术栈文档

## 检查清单

- [x] 项目名称已修正
- [x] 临时文件已清理
- [x] Markdown 语法正确
- [x] 内部链接有效
- [x] Git 提交规范

---

详细说明见 [docs/PR_FINAL.md](docs/PR_FINAL.md)
```

---

## 💡 提示

### 使用命令行创建 PR 的优势

- ✅ 自动读取 PR 描述文件
- ✅ 自动关联相关 Issue
- ✅ 可以在终端直接预览
- ✅ 更快的创建速度

### 使用浏览器创建 PR 的优势

- ✅ 可以实时预览渲染效果
- ✅ 可以添加标签和审阅者
- ✅ 可以关联项目和里程碑
- ✅ 可视化操作更直观

---

## 🚀 快速链接

**PR 创建链接**:
```
https://github.com/chaojixinren/pulse/pull/new/feature/frontend-style-config
```

**分支对比**:
```
https://github.com/chaojixinren/pulse/compare/main...feature/frontend-style-config
```

**PR 列表**:
```
https://github.com/chaojixinren/pulse/pulls
```

---

**准备好后即可创建 PR！** 🎉
