import { Component, type ErrorInfo, type ReactNode } from 'react';
import { Button } from '@/components/common/Button';

export interface ErrorBoundaryProps {
  children: ReactNode;
  title?: string;
  fallback?: (error: Error, reset: () => void) => ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

// 路由级错误边界：捕获子树渲染错误，展示错误态与重试，避免白屏。
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // 记录错误，便于排查；生产环境可接入错误上报。
    console.error('[ErrorBoundary]', error, info.componentStack);
  }

  reset = (): void => {
    this.setState({ error: null });
  };

  render(): ReactNode {
    const { error } = this.state;
    if (error) {
      if (this.props.fallback) {
        return this.props.fallback(error, this.reset);
      }
      return (
        <div className="error-state" role="alert">
          <div className="error-state-title">{this.props.title ?? '页面出错了'}</div>
          <div className="error-state-message">{error.message || '发生未知错误'}</div>
          <Button onClick={this.reset}>重试</Button>
        </div>
      );
    }
    return this.props.children;
  }
}
