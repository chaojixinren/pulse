import type { Page } from '@playwright/test';

interface Identity {
  id: string;
  user_id: string;
  name: string;
  description?: string;
  color: string;
  icon: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

interface TimelineItem {
  session_id: string;
  identity_id?: string;
  transcript: string;
  duration: number;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  recorded_at: string;
}

const USER = {
  id: 'u1',
  email: 'user@example.com',
  name: '测试用户',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const initialIdentities: Identity[] = [
  {
    id: 'i1',
    user_id: 'u1',
    name: '产品经理',
    description: '负责产品规划',
    color: '#3b82f6',
    icon: '🙂',
    is_default: true,
    created_at: '',
    updated_at: '',
  },
  {
    id: 'i2',
    user_id: 'u1',
    name: '健身教练',
    color: '#10b981',
    icon: '🏋️',
    is_default: false,
    created_at: '',
    updated_at: '',
  },
];

const timelineItems: TimelineItem[] = Array.from({ length: 25 }, (_, i) => {
  const n = i + 1;
  const status: TimelineItem['status'] =
    n <= 10 ? 'completed' : n <= 20 ? 'processing' : 'pending';
  return {
    session_id: `s${n}`,
    identity_id: n % 2 === 1 ? 'i1' : 'i2',
    transcript: `第 ${n} 条会话转写文本`,
    duration: 60 + n,
    status,
    recorded_at: `2024-06-05T09:${String(n % 60).padStart(2, '0')}:00Z`,
  };
});

/**
 * 以 page.route 拦截所有 `/api/v1/**` 请求，提供确定性的内存后端。
 * 每个测试拥有独立的内存状态（每次调用重新创建闭包数据）。
 */
export async function mockApi(page: Page): Promise<void> {
  let identities = initialIdentities.map((i) => ({ ...i }));

  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname.replace(/^\/api\/v1/, '');
    const method = route.request().method();
    const respond = (data: unknown, code = 0, message = 'ok') =>
      route.fulfill({ json: { code, message, data } });
    const body = (): Record<string, unknown> => {
      try {
        return (route.request().postDataJSON() ?? {}) as Record<string, unknown>;
      } catch {
        return {};
      }
    };

    // ---- 认证 ----
    if (method === 'POST' && path === '/auth/register') {
      return respond(USER);
    }
    if (method === 'POST' && path === '/auth/login') {
      const { password } = body();
      if (password === 'wrong-password') {
        return respond(undefined, 401, '邮箱或密码错误');
      }
      return respond({ access_token: 'e2e-access', refresh_token: 'e2e-refresh' });
    }
    if (method === 'POST' && path === '/auth/logout') {
      return respond(null);
    }
    if (method === 'GET' && path === '/auth/me') {
      return respond(USER);
    }

    // ---- 身份管理 ----
    if (method === 'GET' && path === '/identities') {
      return respond(identities);
    }
    if (method === 'POST' && path === '/identities') {
      const b = body();
      const created: Identity = {
        id: `i-${Date.now()}`,
        user_id: 'u1',
        name: String(b.name ?? ''),
        description: b.description ? String(b.description) : undefined,
        color: String(b.color ?? '#3b82f6'),
        icon: String(b.icon ?? '🙂'),
        is_default: Boolean(b.is_default),
        created_at: '',
        updated_at: '',
      };
      identities.push(created);
      return respond(created);
    }
    const defaultMatch = path.match(/^\/identities\/([^/]+)\/default$/);
    if (method === 'PUT' && defaultMatch) {
      identities = identities.map((i) => ({ ...i, is_default: i.id === defaultMatch[1] }));
      return respond(null);
    }
    const idMatch = path.match(/^\/identities\/([^/]+)$/);
    if (method === 'PUT' && idMatch) {
      const b = body();
      identities = identities.map((i) => (i.id === idMatch[1] ? { ...i, ...b } : i));
      return respond(identities.find((i) => i.id === idMatch[1]));
    }
    if (method === 'DELETE' && idMatch) {
      identities = identities.filter((i) => i.id !== idMatch[1]);
      return respond(null);
    }

    // ---- 时间线 ----
    if (method === 'GET' && path === '/timeline') {
      let items = [...timelineItems];
      const identityId = url.searchParams.get('identity_id');
      const status = url.searchParams.get('status');
      if (identityId) items = items.filter((i) => i.identity_id === identityId);
      if (status) items = items.filter((i) => i.status === status);

      const pageNum = Number(url.searchParams.get('page') ?? '1');
      const size = Number(url.searchParams.get('size') ?? '20');
      const total = items.length;
      const start = (pageNum - 1) * size;
      return respond({
        items: items.slice(start, start + size),
        total,
        page: pageNum,
        size,
      });
    }

    // ---- 日报 ----
    if (method === 'GET' && path === '/reports/daily') {
      const date = url.searchParams.get('date') ?? '';
      return respond({
        date,
        session_count: 2,
        total_duration: 3660,
        by_identity: [
          { identity_id: 'i1', name: '产品经理', session_count: 1, total_duration: 3660 },
          { identity_id: 'i2', name: '健身教练', session_count: 1, total_duration: 0 },
        ],
        todos: ['整理产品需求文档'],
        notes: ['讨论了产品路线图'],
      });
    }

    return respond(null, 404, 'not found');
  });
}

/** 登录并等待进入受保护页面（默认重定向到 /identity）。 */
export async function login(page: Page): Promise<void> {
  await mockApi(page);
  await page.goto('/auth/login');
  await page.getByLabel('邮箱').fill('user@example.com');
  await page.getByLabel('密码').fill('password');
  await page.getByRole('button', { name: '登录' }).click();
  await page.waitForURL('**/identity');
}
