import { useCallback, useEffect, useState } from 'react';
import { Button } from '@/components/common/Button';
import { Empty } from '@/components/common/Empty';
import { Loading } from '@/components/common/Loading';
import { TimelineItem } from '@/components/business/TimelineItem';
import { useIdentityMap } from '@/hooks/useIdentityMap';
import { identityService } from '@/services/identity.service';
import { timelineService } from '@/services/timeline.service';
import type { Identity } from '@/types/identity.types';
import type {
  TimelineItem as TimelineItemData,
  TimelineQuery,
} from '@/types/timeline.types';

const PAGE_SIZE = 20;

const STATUS_OPTIONS: { value: string; label: string }[] = [
  { value: '', label: '全部状态' },
  { value: 'pending', label: '待处理' },
  { value: 'processing', label: '处理中' },
  { value: 'completed', label: '已完成' },
  { value: 'failed', label: '失败' },
];

interface Filters {
  identity_id: string;
  from: string;
  to: string;
  status: string;
}

const initialFilters: Filters = { identity_id: '', from: '', to: '', status: '' };

export function Component() {
  const [identities, setIdentities] = useState<Identity[]>([]);
  const [filters, setFilters] = useState<Filters>(initialFilters);
  const [items, setItems] = useState<TimelineItemData[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    identityService
      .list()
      .then(setIdentities)
      .catch(() => {
        // 身份下拉加载失败不阻塞时间线主流程
      });
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const query: TimelineQuery = {
        page,
        size: PAGE_SIZE,
      };
      if (filters.identity_id) query.identity_id = filters.identity_id;
      if (filters.status) query.status = filters.status;
      if (filters.from) query.from = filters.from + 'T00:00:00Z';
      if (filters.to) query.to = filters.to + 'T23:59:59Z';

      const result = await timelineService.list(query);
      setItems(result.items);
      setTotal(result.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败');
    } finally {
      setLoading(false);
    }
  }, [page, filters]);

  useEffect(() => {
    load();
  }, [load]);

  const identityMap = useIdentityMap(identities);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const applyFilters = (next: Filters) => {
    setFilters(next);
    setPage(1);
  };

  const resetFilters = () => {
    setFilters(initialFilters);
    setPage(1);
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">时间线</h1>
          <p className="page-subtitle">查看语音会话的转写文本，按身份、日期与状态过滤。</p>
        </div>
        <Button variant="secondary" onClick={load}>
          刷新
        </Button>
      </div>

      <div className="filter-bar">
        <div className="filter-item">
          <label htmlFor="filter-identity">身份</label>
          <select
            id="filter-identity"
            className="form-select"
            value={filters.identity_id}
            onChange={(e) => applyFilters({ ...filters, identity_id: e.target.value })}
          >
            <option value="">全部身份</option>
            {identities.map((identity) => (
              <option key={identity.id} value={identity.id}>
                {identity.name}
              </option>
            ))}
          </select>
        </div>
        <div className="filter-item">
          <label htmlFor="filter-from">开始日期</label>
          <input
            id="filter-from"
            type="date"
            className="form-input"
            value={filters.from}
            onChange={(e) => applyFilters({ ...filters, from: e.target.value })}
          />
        </div>
        <div className="filter-item">
          <label htmlFor="filter-to">结束日期</label>
          <input
            id="filter-to"
            type="date"
            className="form-input"
            value={filters.to}
            onChange={(e) => applyFilters({ ...filters, to: e.target.value })}
          />
        </div>
        <div className="filter-item">
          <label htmlFor="filter-status">状态</label>
          <select
            id="filter-status"
            className="form-select"
            value={filters.status}
            onChange={(e) => applyFilters({ ...filters, status: e.target.value })}
          >
            {STATUS_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
        <Button variant="ghost" onClick={resetFilters}>
          重置
        </Button>
      </div>

      {loading ? (
        <Loading />
      ) : error ? (
        <div className="error-state">
          <div>{error}</div>
          <Button onClick={load}>重试</Button>
        </div>
      ) : items.length === 0 ? (
        <Empty
          icon="🕒"
          title="暂无时间线记录"
          description="录制语音并完成转写后，会在这里显示。"
        />
      ) : (
        <>
          <div className="timeline-list">
            {items.map((item) => {
              const identity = item.identity_id ? identityMap.get(item.identity_id) : undefined;
              return <TimelineItem key={item.session_id} item={item} identity={identity} />;
            })}
          </div>
          <div className="pagination">
            <Button
              variant="secondary"
              size="small"
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
            >
              上一页
            </Button>
            <span className="pagination-info">
              第 {page} / {totalPages} 页 · 共 {total} 条
            </span>
            <Button
              variant="secondary"
              size="small"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              下一页
            </Button>
          </div>
        </>
      )}
    </div>
  );
}

export default Component;
