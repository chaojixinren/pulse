import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Button } from '@/components/common/Button';
import { Input } from '@/components/common/Input';
import { useAuth } from '@/contexts/AuthContext';

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function Component() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [errors, setErrors] = useState<{ name?: string; email?: string; password?: string; confirm?: string }>({});
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const validate = () => {
    const next: { name?: string; email?: string; password?: string; confirm?: string } = {};
    if (!name.trim()) next.name = '请输入姓名';
    if (!email.trim()) next.email = '请输入邮箱';
    else if (!EMAIL_RE.test(email.trim())) next.email = '邮箱格式不正确';
    if (!password) next.password = '请输入密码';
    else if (password.length < 8) next.password = '密码至少 8 位';
    if (confirm !== password) next.confirm = '两次输入的密码不一致';
    setErrors(next);
    return Object.keys(next).length === 0;
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitError(null);
    if (!validate()) return;
    setSubmitting(true);
    try {
      await register(email.trim(), password, name.trim());
      navigate('/', { replace: true });
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : '注册失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1 className="auth-title">注册</h1>
        <p className="auth-subtitle">创建账号，开始使用 Pulse</p>
        {submitError && <div className="auth-error">{submitError}</div>}
        <form onSubmit={handleSubmit} noValidate>
          <div className="form-field">
            <Input
              label="姓名"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              error={errors.name}
              placeholder="你的名字"
              autoComplete="name"
            />
          </div>
          <div className="form-field">
            <Input
              label="邮箱"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              error={errors.email}
              placeholder="you@example.com"
              autoComplete="email"
            />
          </div>
          <div className="form-field">
            <Input
              label="密码"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              error={errors.password}
              placeholder="至少 8 位"
              autoComplete="new-password"
            />
          </div>
          <div className="form-field">
            <Input
              label="确认密码"
              type="password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              error={errors.confirm}
              placeholder="再次输入密码"
              autoComplete="new-password"
            />
          </div>
          <Button type="submit" block loading={submitting}>
            注册并登录
          </Button>
        </form>
        <div className="auth-footer">
          已有账号？<Link to="/auth/login">去登录</Link>
        </div>
      </div>
    </div>
  );
}

export default Component;
