# Pulse Frontend

用户端 Web 应用，提供身份管理、时间线查看、报告展示等功能。

## 技术栈

- **框架**：React 18+
- **语言**：TypeScript
- **构建工具**：Vite
- **状态管理**：待定
- **UI 组件库**：待定
- **路由**：React Router
- **移动端**：待定（未来可能使用 React Native）

## 项目结构

```
frontend/
├── src/
│   ├── main.tsx              # 应用入口
│   ├── App.tsx               # 根组件
│   ├── components/           # 通用组件
│   ├── pages/                # 页面组件
│   │   ├── Auth/             # 认证页面
│   │   ├── Device/           # 设备管理
│   │   ├── Identity/         # 身份管理
│   │   ├── Timeline/         # 时间线
│   │   └── Report/           # 报告中心
│   ├── services/             # API 服务
│   │   └── api.ts            # API 客户端
│   ├── hooks/                # 自定义 Hooks
│   ├── utils/                # 工具函数
│   ├── types/                # TypeScript 类型定义
│   └── styles/               # 样式文件
├── public/                   # 静态资源
├── index.html                # HTML 入口
├── package.json              # 依赖管理
├── tsconfig.json             # TypeScript 配置
├── vite.config.ts            # Vite 配置
└── README.md                 # 本文件
```

## 快速开始

### 1. 安装依赖

```bash
npm install
# 或
yarn install
# 或
pnpm install
```

### 2. 配置环境

创建 `.env` 文件：

```env
VITE_API_BASE_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/ws
```

### 3. 运行应用

```bash
# 开发模式
npm run dev

# 构建生产版本
npm run build

# 预览生产构建
npm run preview
```

应用将在 `http://localhost:5173` 运行。

## 功能模块

### 认证模块
- 用户注册/登录
- Token 管理
- 生物识别（Face ID / 指纹）

### 设备管理
- 扫码绑定硬件设备
- 配置 WiFi 网络
- 查看设备状态（电量、连接状态）
- 远程控制（开始/暂停录音）

### 身份管理
- 创建/编辑身份
- 设置身份图标和颜色
- 配置自动切换规则

### 时间线
- 按身份过滤查看
- 查看语音转录文本
- 编辑/删除记录

### 报告中心
- 日报/周报查看
- 时间统计图表
- 身份切换分析

### 提醒中心
- 待办事项列表
- 承诺提醒
- 身份切换提示

## 开发

### 运行测试

```bash
flutter test
```

### 代码生成

```bash
# 生成模型序列化代码
flutter pub run build_runner build

# 监听文件变化自动生成
flutter pub run build_runner watch
```

### 代码检查

```bash
flutter analyze
```

### 格式化代码

```bash
flutter format lib/
```

## 构建

### Android

```bash
# Debug APK
flutter build apk --debug

# Release APK
flutter build apk --release

# App Bundle (推荐用于 Google Play)
flutter build appbundle --release
```

### iOS

```bash
# 需要 Mac + Xcode
flutter build ios --release
```

## 目录说明

- `lib/core/` - 核心功能和基础设施代码
- `lib/features/` - 按功能模块组织的业务代码
- `lib/widgets/` - 可复用的 UI 组件
- `assets/` - 图片、图标、字体等资源文件
- `test/` - 单元测试和集成测试

## 主要依赖

```yaml
dependencies:
  flutter_riverpod: ^2.4.0      # 状态管理
  dio: ^5.4.0                    # 网络请求
  hive: ^2.2.3                   # 本地存储
  sqflite: ^2.3.0                # SQLite 数据库
  qr_code_scanner: ^1.0.1        # 扫码绑定设备
  fl_chart: ^0.65.0              # 图表展示
  local_auth: ^2.1.8             # 生物识别
  permission_handler: ^11.1.0    # 权限管理
```

## 许可证

MIT
