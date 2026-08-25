#!/bin/bash
# Pulse Git Worktree 管理脚本
# 帮助开发者快速创建和管理功能分支 worktree

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 获取远程 URL
get_remote_url() {
    git remote get-url origin 2>/dev/null || echo ""
}

# 显示用法
show_usage() {
    echo "Pulse Git Worktree 管理工具"
    echo ""
    echo "用法: ./worktree.sh <command> [branch-name]"
    echo ""
    echo "命令:"
    echo "  create <branch>    创建新功能分支 worktree"
    echo "  list              列出所有 worktree"
    echo "  remove <branch>   删除 worktree 和分支"
    echo "  switch <branch>   切换到指定 worktree"
    echo ""
    echo "示例:"
    echo "  ./worktree.sh create feature/user-auth"
    echo "  ./worktree.sh list"
    echo "  ./worktree.sh remove feature/user-auth"
}

# 创建 worktree
create_worktree() {
    local branch=$1
    if [ -z "$branch" ]; then
        echo -e "${RED}错误: 请提供分支名称${NC}"
        show_usage
        exit 1
    fi

    echo -e "${GREEN}Creating worktree for branch: $branch${NC}"

    # 检查分支是否已存在
    if git show-ref --verify --quiet "refs/heads/$branch"; then
        echo -e "${YELLOW}分支 $branch 已存在，直接创建 worktree${NC}"
    else
        echo -e "${GREEN}创建新分支: $branch${NC}"
        git worktree add -b "$branch" "../pulse-$branch" "origin/develop" 2>/dev/null || {
            echo -e "${YELLOW}origin/develop 不存在，使用 main 分支${NC}"
            git worktree add -b "$branch" "../pulse-$branch" "origin/main"
        }
    fi

    echo ""
    echo -e "${GREEN}✅ Worktree 创建成功！${NC}"
    echo "路径: $(pwd)/../pulse-$branch"
    echo ""
    echo "下一步:"
    echo "  cd ../pulse-$branch"
    echo "  npm install  # 安装前端依赖"
    echo "  make setup   # 配置环境"
}

# 列出所有 worktree
list_worktrees() {
    echo -e "${GREEN}当前 Worktree 列表:${NC}"
    git worktree list
}

# 删除 worktree
remove_worktree() {
    local branch=$1
    if [ -z "$branch" ]; then
        echo -e "${RED}错误: 请提供分支名称${NC}"
        show_usage
        exit 1
    fi

    echo -e "${YELLOW}删除 worktree: $branch${NC}"
    git worktree remove "../pulse-$branch"
    git branch -d "$branch" 2>/dev/null || echo -e "${YELLOW}分支可能未合并，跳过删除${NC}"
    echo -e "${GREEN}✅ Worktree 删除完成${NC}"
}

# 切换到 worktree
switch_worktree() {
    local branch=$1
    if [ -z "$branch" ]; then
        echo -e "${RED}错误: 请提供分支名称${NC}"
        show_usage
        exit 1
    fi

    local worktree_path="../pulse-$branch"
    if [ -d "$worktree_path" ]; then
        echo -e "${GREEN}切换到 worktree: $branch${NC}"
        cd "$worktree_path"
        pwd
    else
        echo -e "${RED}错误: Worktree $branch 不存在${NC}"
        exit 1
    fi
}

# 主逻辑
case "$1" in
    create)
        create_worktree "$2"
        ;;
    list)
        list_worktrees
        ;;
    remove)
        remove_worktree "$2"
        ;;
    switch)
        switch_worktree "$2"
        ;;
    *)
        show_usage
        exit 1
        ;;
esac
