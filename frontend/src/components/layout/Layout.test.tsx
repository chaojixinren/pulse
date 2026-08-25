import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Layout } from './Layout';

vi.mock('./Header', () => ({
  Header: () => <div data-testid="header" />,
}));

vi.mock('./Sidebar', () => ({
  Sidebar: () => <div data-testid="sidebar" />,
}));

describe('Layout', () => {
  it('渲染 Header、Sidebar 与内容区', () => {
    render(
      <Layout>
        <div>页面内容</div>
      </Layout>,
    );
    expect(screen.getByTestId('header')).toBeInTheDocument();
    expect(screen.getByTestId('sidebar')).toBeInTheDocument();
    expect(screen.getByText('页面内容')).toBeInTheDocument();
  });
});
