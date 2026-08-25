import { Outlet } from 'react-router-dom';

// 根组件：仅渲染子路由出口。
export default function App() {
  return <Outlet />;
}
