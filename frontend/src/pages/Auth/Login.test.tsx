import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import Login from './Login';

const { login } = vi.hoisted(() => ({ login: vi.fn() }));

vi.mock('@/contexts/AuthContext', () => ({
  useAuth: () => ({ login }),
}));

function renderLogin() {
  return render(
    <MemoryRouter initialEntries={['/auth/login']}>
      <Routes>
        <Route path="/auth/login" element={<Login />} />
        <Route path="/" element={<div>首页内容</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('Login 页面', () => {
  beforeEach(() => {
    login.mockReset();
  });

  it('渲染登录表单', () => {
    renderLogin();
    expect(screen.getByRole('heading', { name: 'Pulse' })).toBeInTheDocument();
    expect(screen.getByLabelText('邮箱')).toBeInTheDocument();
    expect(screen.getByLabelText('密码')).toBeInTheDocument();
  });

  it('空表单提交展示校验错误', async () => {
    renderLogin();
    await userEvent.click(screen.getByRole('button', { name: '登录' }));
    expect(await screen.findByText('请输入邮箱')).toBeInTheDocument();
    expect(screen.getByText('请输入密码')).toBeInTheDocument();
    expect(login).not.toHaveBeenCalled();
  });

  it('邮箱格式错误提示', async () => {
    renderLogin();
    await userEvent.type(screen.getByLabelText('邮箱'), 'invalid');
    await userEvent.type(screen.getByLabelText('密码'), 'password');
    await userEvent.click(screen.getByRole('button', { name: '登录' }));
    expect(await screen.findByText('邮箱格式不正确')).toBeInTheDocument();
    expect(login).not.toHaveBeenCalled();
  });

  it('登录成功后跳转首页', async () => {
    login.mockResolvedValue(undefined);
    renderLogin();
    await userEvent.type(screen.getByLabelText('邮箱'), '  a@b.com  ');
    await userEvent.type(screen.getByLabelText('密码'), 'password');
    await userEvent.click(screen.getByRole('button', { name: '登录' }));

    expect(await screen.findByText('首页内容')).toBeInTheDocument();
    expect(login).toHaveBeenCalledWith('a@b.com', 'password');
  });

  it('登录失败展示后端错误信息', async () => {
    login.mockRejectedValue(new Error('密码错误'));
    renderLogin();
    await userEvent.type(screen.getByLabelText('邮箱'), 'a@b.com');
    await userEvent.type(screen.getByLabelText('密码'), 'wrong-password');
    await userEvent.click(screen.getByRole('button', { name: '登录' }));

    expect(await screen.findByText('密码错误')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole('button', { name: '登录' })).toBeEnabled());
  });
});
