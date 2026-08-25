import type { Rules, RuleConfigSeverity } from '@commitlint/types';

export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'feat',     // 新功能
        'fix',      // Bug 修复
        'docs',     // 文档更新
        'style',    // 代码格式（不影响功能）
        'refactor', // 重构
        'perf',     // 性能优化
        'test',     // 测试相关
        'chore',    // 构建工具或辅助工具变更
        'ci',       // CI/CD 配置
        'revert',   // 回滚
        'build',    // 构建系统
      ],
    ],
    'subject-case': [0], // 不限制 subject 大小写
    'subject-full-stop': [2, 'never', '.'],
    'subject-max-length': [2, 'always', 72],
    'header-max-length': [2, 'always', 100],
  },
  prompt: {
    questions: {
      type: {
        description: '选择提交类型',
        enum: {
          feat: { description: '✨ feat: 新功能', value: 'feat' },
          fix: { description: '🐛 fix: Bug 修复', value: 'fix' },
          docs: { description: '📝 docs: 文档更新', value: 'docs' },
          style: { description: '💄 style: 代码格式', value: 'style' },
          refactor: { description: '♻️  refactor: 重构', value: 'refactor' },
          perf: { description: '⚡️ perf: 性能优化', value: 'perf' },
          test: { description: '✅ test: 测试相关', value: 'test' },
          chore: { description: '🔧 chore: 构建/工具', value: 'chore' },
          ci: { description: '🎡 ci: CI/CD', value: 'ci' },
          revert: { description: '⏪ revert: 回滚', value: 'revert' },
          build: { description: '📦 build: 构建系统', value: 'build' },
        },
      },
      scope: {
        description: '影响范围（可选）',
      },
      subject: {
        description: '简短描述',
      },
      body: {
        description: '详细描述（可选）',
      },
      isBreaking: {
        description: '是否为破坏性变更？',
      },
      breakingBody: {
        description: '破坏性变更说明',
      },
      breaking: {
        description: '破坏性变更描述',
      },
      footer: {
        description: '附加说明（可选）',
      },
    },
  },
};
