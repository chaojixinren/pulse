# 贡献指南

感谢你对 Pulse 项目的关注！我们欢迎任何形式的贡献。

## 如何贡献

### 报告 Bug

如果你发现了 Bug，请[创建 Issue](https://github.com/yourusername/pulse/issues/new)，并包含：

- **Bug 描述**：简短描述问题
- **复现步骤**：详细的复现步骤
- **期望行为**：你期望发生什么
- **实际行为**：实际发生了什么
- **环境信息**：操作系统、版本号、设备型号等
- **截图/日志**：如果适用

### 提出新功能

在开始开发新功能之前，请先[创建 Issue](https://github.com/yourusername/pulse/issues/new) 讨论：

- **功能描述**：这个功能是什么？
- **使用场景**：为什么需要这个功能？
- **解决方案**：你认为应该如何实现？

### 提交代码

1. **Fork 仓库**

2. **创建分支**
   ```bash
   git checkout -b feature/amazing-feature
   # 或
   git checkout -b fix/bug-description
   ```

3. **编写代码**
   - 遵循现有的代码风格
   - 添加必要的注释
   - 编写测试（如果适用）

4. **提交更改**
   ```bash
   git add .
   git commit -m "feat: add amazing feature"
   ```

   **Commit 消息规范**：
   - `feat:` 新功能
   - `fix:` Bug 修复
   - `docs:` 文档更新
   - `style:` 代码格式（不影响功能）
   - `refactor:` 重构
   - `test:` 测试相关
   - `chore:` 构建工具或辅助工具变更

5. **推送到 GitHub**
   ```bash
   git push origin feature/amazing-feature
   ```

6. **创建 Pull Request**
   - 描述你的更改
   - 关联相关的 Issue
   - 等待 Review

## 开发规范

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

## 社区准则

- **尊重他人**：友好、包容地交流
- **建设性反馈**：提出问题时给出具体建议
- **保持专业**：专注于技术讨论
- **保护隐私**：不要泄露敏感信息

## 获取帮助

- **Discord**: [加入我们的 Discord](https://discord.gg/pulse)
- **邮件**: dev@pulse.example.com
- **文档**: https://docs.pulse.example.com

---

再次感谢你的贡献！🎉
