import { lazy, Suspense, type ComponentType } from 'react';
import { Navigate, createBrowserRouter } from 'react-router-dom';
import App from '@/App';
import { ErrorBoundary } from '@/components/common/ErrorBoundary';
import { FullScreenLoading } from '@/components/common/Loading';
import { NotFound } from './NotFound';
import { RequireAuth } from './RequireAuth';

type PageModule = { default: ComponentType };

// 每个路由包裹 ErrorBoundary + Suspense：崩溃展示错误态，代码分包加载展示全屏加载态。
function pageElement(load: () => Promise<PageModule>) {
  const Lazy = lazy(load);
  return (
    <ErrorBoundary>
      <Suspense fallback={<FullScreenLoading />}>
        <Lazy />
      </Suspense>
    </ErrorBoundary>
  );
}

export const router = createBrowserRouter([
  {
    element: <App />,
    children: [
      { path: '/auth/login', element: pageElement(() => import('@/pages/Auth/Login')) },
      { path: '/auth/register', element: pageElement(() => import('@/pages/Auth/Register')) },
      {
        path: '/',
        element: <RequireAuth />,
        children: [
          { index: true, element: <Navigate to="/identity" replace /> },
          { path: 'identity', element: pageElement(() => import('@/pages/Identity/IdentityList')) },
          { path: 'timeline', element: pageElement(() => import('@/pages/Timeline/TimelineList')) },
          { path: 'reports/daily', element: pageElement(() => import('@/pages/Report/DailyReport')) },
          { path: 'reports/weekly', element: pageElement(() => import('@/pages/Report/WeeklyReport')) },
          { path: 'reports/stats', element: pageElement(() => import('@/pages/Report/StatsReport')) },
          { path: 'account', element: pageElement(() => import('@/pages/Account/AccountSettings')) },
          { path: 'devices', element: pageElement(() => import('@/pages/Device/DeviceList')) },
          { path: 'devices/bind', element: pageElement(() => import('@/pages/Device/BindDevice')) },
          { path: 'devices/:id', element: pageElement(() => import('@/pages/Device/DeviceDetail')) },
        ],
      },
      { path: '*', element: <NotFound /> },
    ],
  },
]);
