import { Navigate, Outlet } from 'react-router-dom';
import { FullScreenLoading } from '@/components/common/Loading';
import { Layout } from '@/components/layout/Layout';
import { useAuth } from '@/contexts/AuthContext';

// 未登录跳登录页，登录后渲染受保护布局与子路由。
export function RequireAuth() {
  const { isAuthenticated, loading } = useAuth();

  if (loading) return <FullScreenLoading />;
  if (!isAuthenticated) return <Navigate to="/auth/login" replace />;

  return (
    <Layout>
      <Outlet />
    </Layout>
  );
}
