import { Navigate, createBrowserRouter } from 'react-router-dom';
import App from '@/App';
import { NotFound } from './NotFound';
import { RequireAuth } from './RequireAuth';

export const router = createBrowserRouter([
  {
    element: <App />,
    children: [
      { path: '/auth/login', lazy: () => import('@/pages/Auth/Login') },
      { path: '/auth/register', lazy: () => import('@/pages/Auth/Register') },
      {
        path: '/',
        element: <RequireAuth />,
        children: [
          { index: true, element: <Navigate to="/identity" replace /> },
          { path: 'identity', lazy: () => import('@/pages/Identity/IdentityList') },
          { path: 'timeline', lazy: () => import('@/pages/Timeline/TimelineList') },
          { path: 'reports/daily', lazy: () => import('@/pages/Report/DailyReport') },
          { path: 'devices', lazy: () => import('@/pages/Device/DeviceList') },
          { path: 'devices/bind', lazy: () => import('@/pages/Device/BindDevice') },
          { path: 'devices/:id', lazy: () => import('@/pages/Device/DeviceDetail') },
        ],
      },
      { path: '*', element: <NotFound /> },
    ],
  },
]);
