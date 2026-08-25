#!/bin/bash
# Pulse 项目开发环境配置脚本
# 适用于新贡献者快速 setup

set -e

echo "🚀 Setting up Pulse development environment..."
echo ""

# 检查必要工具
command -v git >/dev/null 2>&1 || { echo "❌ git is required but not installed."; exit 1; }
command -v go >/dev/null 2>&1 || { echo "❌ Go is required but not installed. Visit https://go.dev/dl/"; exit 1; }
command -v node >/dev/null 2>&1 || { echo "❌ Node.js is required but not installed. Visit https://nodejs.org/"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "❌ npm is required but not installed."; exit 1; }

echo "✅ All required tools are installed"
echo ""

# 配置 Git 用户信息
echo "📝 Configuring Git..."
read -p "Enter your name: " git_name
read -p "Enter your email: " git_email

git config user.name "$git_name"
git config user.email "$git_email"
git config commit.template .gitmessage.txt

echo "✅ Git configured"
echo ""

# 安装 Node 依赖
echo "📦 Installing Node.js dependencies..."
cd frontend
npm install
cd ..
echo "✅ Node dependencies installed"
echo ""

# 安装 Go 依赖
echo "📦 Installing Go dependencies..."
cd backend
go mod download
cd ..
echo "✅ Go dependencies installed"
echo ""

# 安装 Husky Git hooks
echo "🪝 Installing Husky Git hooks..."
npm run prepare
echo "✅ Husky installed"
echo ""

echo "🎉 Setup complete!"
echo ""
echo "Next steps:"
echo "1. Create a new branch: git checkout -b feature/your-feature"
echo "2. Start developing!"
echo "3. Run 'make test' to run tests"
echo "4. Run 'make lint' to check code style"
echo ""
echo "📖 For more information, see CONTRIBUTING.md"
