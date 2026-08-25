import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import Register from './Register';

const { register } = vi.hoisted(() => ({ register: vi.fn() }));

vi.mock('@/contexts/AuthContext', () => ({
  useAuth: () => ({ register }),
}));

function renderRegister() {
  return render(
    <MemoryRouter initialEntries={['/auth/register']}>
      <Routes>
        <Route path="/auth/register" element={<Register />} />
        <Route path="/" element={<div>首页内容</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('Register 页面', () => {
  beforeEach(() => {
    register.mockReset();
  });

  it('空表单提交展示校验错误', async () => {
    renderRegister();
    await userEvent.click(screen.getByRole('button', { name: '注册并登录' }));
    expect(await screen.findByText('请输入姓名')).toBeInTheDocument();
    expect(screen.getByText('请输入邮箱')).toBeInTheDocument();
    expect(screen.getByText('请输入密码')).toBeInTheDocument();
    expect(register).not.toHaveBeenCalled();
  });

  it('密码不足 8 位提示', async () => {
    renderRegister();
    await userEvent.type(screen.getByLabelText('姓名'), 'Alice');
    await userEvent.type(screen.getByLabelText('邮箱'), 'a@b.com');
    await userEvent.type(screen.getByLabelText('密码'), '123');
    await userEvent.type(screen.getByLabelText('确认密码'), '123');
    await userEvent.click(screen.getByRole('button', { name: '注册并登录' }));
    expect(await screen.findByText('密码至少 8 位')).toBeInTheDocument();
    expect(register).not.toHaveBeenCalled();
  });

  it('两次密码不一致提示', async () => {
    renderRegister();
    await userEvent.type(screen.getByLabelText('姓名'), 'Alice');
    await userEvent.type(screen.getByLabelText('邮箱'), 'a@b.com');
    await userEvent.type(screen.getByLabelText('密码'), '12345678');
    await userEvent.type(screen.getByLabelText('确认密码'), '87654321');
    await userEvent.click(screen.getByRole('button', { name: '注册并登录' }));
    expect(await screen.findByText('两次输入的密码不一致')).toBeInTheDocument();
    expect(register).not.toHaveBeenCalled();
  });

  it('注册成功后跳转首页', async () => {
    register.mockResolvedValue(undefined);
    renderRegister();
    await userEvent.type(screen.getByLabelText('姓名'), ' Alice ');
    await userEvent.type(screen.getByLabelText('邮箱'), 'a@b.com');
    await userEvent.type(screen.getByLabelText('密码'), '12345678');
    await userEvent.type(screen.getByLabelText('确认密码'), '12345678');
    await userEvent.click(screen.getByRole('button', { name: '注册并登录' }));

    expect(await screen.findByText('首页内容')).toBeInTheDocument();
    expect(register).toHaveBeenCalledWith('a@b.com', '12345678', 'Alice');
  });

  it('注册失败展示错误信息', async () => {
    register.mockRejectedValue(new Error('邮箱已注册'));
    renderRegister();
    await userEvent.type(screen.getByLabelText('姓名'), 'Alice');
    await userEvent.type(screen.getByLabelText('邮箱'), 'a@b.com');
    await userEvent.type(screen.getByLabelText('密码'), '12345678');
    await userEvent.type(screen.getByLabelText('确认密码'), '12345678');
    await userEvent.click(screen.getByRole('button', { name: '注册并登录' }));

    expect(await screen.findByText('邮箱已注册')).toBeInTheDocument();
  });
});
