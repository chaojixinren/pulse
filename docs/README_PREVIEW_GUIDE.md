# 📖 README 预览指南

## 🎯 推荐方式：VSCode Markdown Preview

### 方式一：VSCode 预览（最快）

1. 在 VSCode 中打开 `README.md`
2. 按 `Ctrl+Shift+V`（Windows/Linux）或 `Cmd+Shift+V`（Mac）
3. 实时查看渲染效果 ✨

**优势**：
- ✅ 即时预览
- ✅ 支持数学公式、Mermaid 图表
- ✅ 与 GitHub 渲染效果高度一致

---

## 🌐 在线预览方式

### 方式二：Markdown Live Preview

1. 访问：https://markdownlivepreview.com/
2. 在左侧粘贴 README 内容
3. 右侧实时查看渲染效果

**优势**：
- ✅ 无需安装
- ✅ 支持实时编辑
- ✅ 多主题切换

### 方式三：GitHub 网页预览

访问：
```
https://github.com/chaojixinren/pulse/blob/feature/frontend-style-config/README.md
```

**优势**：
- ✅ GitHub 官方渲染效果
- ✅ 100% 还原最终效果

### 方式四：Dillinger 在线编辑器

访问：https://dillinger.io/

**优势**：
- ✅ 功能强大的在线 Markdown 编辑器
- ✅ 支持导出 PDF/HTML
- ✅ 支持本地文件导入

---

## 📂 本地预览脚本

我已经为你创建了两个预览脚本：

### Windows 用户
双击运行：`preview_readme.bat`

### Mac/Linux 用户
```bash
chmod +x preview_readme.sh
./preview_readme.sh
```

---

## 🔍 重点预览区域

以下章节在优化中改动最大，建议重点关注：

### 1. 🌟 为什么选择 Pulse（第 17-28 行）
```markdown
## 🌟 为什么选择 Pulse

在忙碌的现代生活中，我们每个人都在扮演着多重角色——
作为员工处理工作事务，作为父母陪伴孩子成长，作为朋友分享欢笑，
作为子女倾听父母的唠叨。

每一个身份，都是一段独特的人生旅程。
每一次对话，都承载着重要的承诺与待办。

**Pulse（时笺）** 致力于温柔地记录这些瞬间，
用智能技术帮助你理解自己的生活轨迹，
让重要的事情不再被遗忘。
```

### 2. 🏗️ 系统架构图（第 86-130 行）
```
┌─────────────────────────────────────────────────────────────┐
│                    🎙️ 可穿戴硬件 (ESP32-S3)                    │
│                  语音采集 → WiFi 加密上传                       │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTPS
                       ↓
┌─────────────────────────────────────────────────────────────┐
│              🚀 Gin Web Server (Go 1.26.5)                   │
│  ┌──────────┬──────────┬──────────┬──────────────────────┐ │
│  │ 认证中间件 │ 日志中间件 │ CORS 中间件│ 限流中间件           │ │
│  └──────────┴──────────┴──────────┴──────────────────────┘ │
└──────────────────────┬──────────────────────────────────────┘
                       │
            ┌──────────┴──────────┐
            │                     │
            ↓                     ↓
┌──────────────────┐       ┌──────────────────┐
│   🗄️ MySQL 8.0   │       │   🤖 AI Service  │
└──────────────────┘       └──────────────────┘
```

### 3. 🎯 核心功能（第 54-83 行）
- 🎤 智能语音采集
- 🧠 AI 身份识别
- 📝 智能信息提取
- 📊 多维报告生成
- 🔒 隐私保护

### 4. 💫 核心理念（第 32-51 行）
- 身份流转，智能记录
- 数据主权，隐私优先
- 科技服务于人

---

## 📊 改进对比预览

### Before（原版）
```
# Pulse - 时笺

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

> 通过语音记录生活，让 AI 帮你在不同身份间自由切换

## 项目简介
时笺 是一个智能身份管理系统，帮助用户在多重生活角色中自由切换...
```

### After（优化版）
```
# Pulse · 时笺

![Version](https://img.shields.io/badge/version-0.1.0-blue.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)
![Status](https://img.shields.io/badge/status-active-success.svg)
![Go Version](https://img.shields.io/badge/Go-1.26.5-00ADD8.svg?logo=go)
![React](https://img.shields.io/badge/React-18.2-61DAFB.svg?logo=react)
![ESP32](https://img.shields.io/badge/ESP32-S3-lightgrey.svg)

> **通过语音记录生活，让 AI 帮你在不同身份间自由切换**

[文档](./docs/) · [快速开始](#-快速开始) · [贡献指南](./CONTRIBUTING.md)

---

## 🌟 为什么选择 Pulse

在忙碌的现代生活中，我们每个人都在扮演着多重角色——
作为员工处理工作事务，作为父母陪伴孩子成长，作为朋友分享欢笑，
作为子女倾听父母的唠叨。

每一个身份，都是一段独特的人生旅程。
每一次对话，都承载着重要的承诺与待办。

**Pulse（时笺）** 致力于温柔地记录这些瞬间，
用智能技术帮助你理解自己的生活轨迹，
让重要的事情不再被遗忘。
```

---

## 💡 预览技巧

### VSCode 快捷键
- `Ctrl+Shift+V` / `Cmd+Shift+V` — 打开预览
- `Ctrl+K V` / `Cmd+K V` — 侧边预览（并排查看）

### GitHub 特色渲染
- ✅ 徽章（Badges）会自动渲染为彩色图片
- ✅ 表情符号显示为彩色图标
- ✅ 代码块带语法高亮
- ✅ 表格带样式
- ✅ ASCII 架构图对齐展示

### 常见预览问题
1. **徽章不显示**：在 GitHub 网页查看时才会渲染
2. **表情符号乱码**：确保使用支持 emoji 的字体
3. **ASCII 图错位**：GitHub 使用等宽字体，本地可能需要手动调整

---

## ✅ 检查清单

预览时请重点检查：

- [ ] 徽章是否正常显示（需要 GitHub 渲染）
- [ ] 表情符号是否清晰可见
- [ ] 标题层级是否合理
- [ ] ASCII 架构图是否对齐
- [ ] 代码块是否有语法高亮
- [ ] 表格格式是否整齐
- [ ] 内部链接是否可点击
- [ ] 整体视觉效果是否符合预期

---

## 🎨 GitHub 渲染效果确认

**最终效果请务必在 GitHub 上确认**，因为：

1. GitHub 有自己的 Markdown 渲染引擎
2. 部分效果（如徽章）仅在 GitHub 上显示
3. 字体、间距、颜色可能与本地预览略有差异

**查看链接**：
```
https://github.com/chaojixinren/pulse/blob/feature/frontend-style-config/README.md
```
