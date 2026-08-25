# 开发者指南

本文档提供 Pulse 项目的详细开发指南，帮助新贡献者快速上手。

## 📚 目录

1. [开发环境搭建](#开发环境搭建)
2. [Git 工作流程](#git-工作流程)
3. [代码规范](#代码规范)
4. [测试指南](#测试指南)
5. [调试技巧](#调试技巧)
6. [常见问题](#常见问题)

---

## 开发环境搭建

### 方式一：快速配置（推荐）

```bash
./setup.sh
```

脚本会自动检查并安装所有依赖。

### 方式二：手动配置

#### 1. 安装前置工具

**必需：**

- Go 1.21+ - https://go.dev/dl/
- Node.js 18+ - https://nodejs.org/
- Git - https://git-scm.com/

**可选：**

- Docker & Docker Compose - https://www.docker.com/
- golangci-lint - `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- Make - 通常已预装

#### 2. 克隆仓库

```bash
git clone https://github.com/yourusername/pulse.git
cd pulse
```

#### 3. 配置 Git Hooks

```bash
npm run prepare
```

#### 4. 安装依赖

```bash
# 后端
cd backend && go mod download && cd ..

# 前端
cd frontend && npm install && cd ..
```

#### 5. 启动开发环境

```bash
make docker-up
```

访问：

- 后端 API: http://localhost:8080
- 前端: http://localhost:5173

---

## Git 工作流程

### 分支管理

我们使用 **Feature Branch Workflow**：

```
main (生产环境)
  ↑
develop (开发分支)
  ↑
feature/* (功能分支)
fix/* (Bug 修复)
docs/* (文档)
```

### 完整工作流程

#### 1. 开始新功能

```bash
# 从最新的 develop 分支创建功能分支
git checkout develop
git pull upstream develop
git checkout -b feature/your-feature
```

#### 2. 本地开发

```bash
# 开发代码...

# 运行测试
make test
cd frontend && npm test

# 检查代码
make lint
cd frontend && npm run lint
```

#### 3. 提交代码

```bash
# 查看状态
git status

# 添加文件
git add .

# 提交（自动触发 Git hooks）
git commit -m "feat(auth): add user login"

# 推送
git push origin feature/your-feature
```

#### 4. 创建 Pull Request

1. 访问 GitHub 仓库
2. 点击 "Compare & pull request"
3. 填写 PR 模板
4. 提交 PR

#### 5. Code Review

- 维护者会评论你的代码
- 如需修改，继续提交到同一分支
- 收到 **LGTM** 后等待合并

#### 6. 合并后清理

```bash
# 切回 develop
git checkout develop
git pull upstream develop

# 删除本地分支
git branch -d feature/your-feature

# 删除远程分支
git push origin --delete feature/your-feature
```

### 使用 Worktree（推荐）

为不同功能分支创建独立工作目录：

```bash
# 创建工作目录
./worktree.sh create feature/user-auth

# 进入工作目录
cd ../pulse-feature-user-auth

# 开发完成后清理
./worktree.sh remove feature/user-auth
```

---

## 代码规范

### 后端（Go）

**命名：**

```go
// ✅ 使用驼峰，有意义的名字
func GetUserByID(ctx context.Context, userID string) (*User, error) {
    user, err := repo.FindUserByID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return user, nil
}

// ❌ 避免
func get(id string) (*User, error) { ... }
```

**错误处理：**

```go
// ✅ 包含上下文信息
if err != nil {
    return fmt.Errorf("failed to process user %s: %w", userID, err)
}

// ✅ 使用结构化日志
logger.Error("database query failed",
    zap.String("table", "users"),
    zap.String("user_id", userID),
    zap.Error(err))
```

**注释：**

```go
// AnalyzeTranscript 分析语音转录文本，提取结构化信息
// 包括：说话人识别、情感分析、关键词提取
func (s *AIService) AnalyzeTranscript(ctx context.Context, transcript string) (*AnalysisResult, error) {
    // ...
}
```

### 前端（TypeScript/React）

**组件：**

```tsx
// ✅ 函数组件 + TypeScript
interface ButtonProps {
  label: string;
  onClick: () => void;
  variant?: 'primary' | 'secondary';
}

export const Button: React.FC<ButtonProps> = ({ label, onClick, variant = 'primary' }) => {
  return <button onClick={onClick}>{label}</button>;
};

// ❌ 避免
const Button = (props) => { ... }
```

**Hook：**

```tsx
// ✅ 自定义 Hook 命名以 use 开头
function useUserProfile(userId: string) {
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    fetchUser(userId).then(setUser);
  }, [userId]);

  return user;
}
```

**格式化：**

```bash
# 自动格式化
npm run lint:fix

# 类型检查
npm run type-check
```

### 格式化工具

项目使用 **Prettier** 统一代码格式：

```bash
# 前端
cd frontend && npm run lint:fix

# 后端
make fmt
```

### EditorConfig

项目使用 `.editorconfig` 配置编辑器：

- Indent: 2 spaces（前端）/ tabs（Go）
- LF line endings
- UTF-8 编码
- Trim trailing whitespace

---

## 测试指南

### 后端测试

```bash
# 运行所有测试
make test

# 带竞态检测
make test-race

# 运行 E2E 测试
make test-e2e

# 覆盖率报告
make test
open coverage.html
```

**编写测试：**

```go
func TestGetUserByID(t *testing.T) {
    // 准备
    mockRepo := &MockUserRepo{}
    service := NewUserService(mockRepo)

    // 执行
    user, err := service.GetUserByID(context.Background(), "123")

    // 验证
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if user.ID != "123" {
        t.Errorf("expected user ID 123, got %s", user.ID)
    }
}
```

### 前端测试

```bash
# 运行所有测试
cd frontend && npm test

# 生成覆盖率报告
cd frontend && npm run test:coverage

# Watch 模式
cd frontend && npm test -- --watch
```

**编写测试：**

```tsx
import { render, screen } from '@testing-library/react';
import { Button } from './Button';

describe('Button', () => {
  it('should render with label', () => {
    render(<Button label="Click me" onClick={() => {}} />);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });

  it('should call onClick when clicked', () => {
    const handleClick = jest.fn();
    render(<Button label="Click" onClick={handleClick} />);

    fireEvent.click(screen.getByText('Click'));

    expect(handleClick).toHaveBeenCalledTimes(1);
  });
});
```

---

## 调试技巧

### 后端调试

```bash
# 使用 Delve 调试器
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/server

# 添加日志
logger.Debug("debug info", zap.Any("data", data))
```

### 前端调试

```bash
# React DevTools
# 在 Chrome/Edge 安装 React DevTools 扩展

# Vite 调试
npm run dev -- --debug

# ESLint 问题
npm run lint
```

### Docker 调试

```bash
# 查看日志
docker logs pulse-backend-1
docker logs pulse-frontend-1

# 进入容器
docker exec -it pulse-backend-1 /bin/sh

# 重新构建
docker compose up -d --build
```

---

## 常见问题

### Q: Git hooks 阻止我提交怎么办？

**A:** 检查提交信息格式和代码质量：

```bash
# 手动检查
npx commitlint --edit $1
make lint
cd frontend && npm run lint

# 临时跳过 hooks（不推荐）
git commit --no-verify -m "feat: xxx"
```

### Q: CI/CD 失败了怎么办？

**A:** 本地先通过所有检查：

```bash
# 后端
make lint
make test

# 前端
cd frontend && npm run lint && npm run type-check && npm test
```

### Q: 如何更新上游代码？

**A:**

```bash
git checkout develop
git pull upstream develop
git push origin develop
```

### Q: Worktree 出现冲突？

**A:** 清理并重建：

```bash
git worktree remove ../pulse-feature-xxx
git worktree prune
```

### Q: 如何添加新的 Husky hooks？

**A:** 编辑 `.huskyrc.json`：

```json
{
  "hooks": {
    "pre-commit": "make lint",
    "commit-msg": "commitlint --edit $1",
    "pre-push": "make test"
  }
}
```

然后重新安装：

```bash
npm run prepare
```

---

## 📖 更多资源

- [Go 官方文档](https://go.dev/doc/)
- [React 官方文档](https://react.dev/)
- [TypeScript 手册](https://www.typescriptlang.org/docs/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [GitHub Flow](https://guides.github.com/introduction/flow/)

---

遇到问题？查看 [CONTRIBUTING.md](CONTRIBUTING.md) 或在 [Discord](https://discord.gg/pulse) 寻求帮助。
