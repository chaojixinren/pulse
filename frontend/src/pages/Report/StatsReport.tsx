import { useCallback, useEffect, useState } from 'react';
import { Button } from '@/components/common/Button';
import { Empty } from '@/components/common/Empty';
import { Loading } from '@/components/common/Loading';
import { IdentityPie } from '@/components/business/IdentityPie';
import { TrendChart } from '@/components/business/TrendChart';
import { reportService } from '@/services/report.service';
import type { StatsReport as StatsReportData } from '@/types/report.types';
import { shiftDate, todayStr } from '@/utils/date';
import { formatDuration, formatDurationShort } from '@/utils/format';

interface Range {
  from: string;
  to: string;
}

function defaultRange(): Range {
  return { from: shiftDate(todayStr(), -29), to: todayStr() };
}

export function Component() {
  const [from, setFrom] = useState<string>(() => defaultRange().from);
  const [to, setTo] = useState<string>(() => defaultRange().to);
  const [query, setQuery] = useState<Range>(() => defaultRange());
  const [report, setReport] = useState<StatsReportData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rangeError, setRangeError] = useState<string | null>(null);

  const load = useCallback(async (range: Range) => {
    setLoading(true);
    setError(null);
    try {
      setReport(await reportService.stats(range.from, range.to));
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败');
      setReport(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(query);
  }, [query, load]);

  const applyRange = () => {
    if (!from || !to) {
      setRangeError('请选择起始与结束日期');
      return;
    }
    if (from > to) {
      setRangeError('起始日期不能晚于结束日期');
      return;
    }
    setRangeError(null);
    setQuery({ from, to });
  };

  const hasData = Boolean(
    report && (report.session_count > 0 || report.daily_trend.length > 0 || report.by_identity.length > 0),
  );

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">统计</h1>
          <p className="page-subtitle">自定义区间查看会话趋势与身份占比。</p>
        </div>
      </div>

      <div className="report-toolbar">
        <div className="filter-item">
          <label htmlFor="stats-from">起始日期</label>
          <input
            id="stats-from"
            type="date"
            className="form-input"
            style={{ maxWidth: 160 }}
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          />
        </div>
        <div className="filter-item">
          <label htmlFor="stats-to">结束日期</label>
          <input
            id="stats-to"
            type="date"
            className="form-input"
            style={{ maxWidth: 160 }}
            value={to}
            onChange={(e) => setTo(e.target.value)}
          />
        </div>
        <Button onClick={applyRange}>查询</Button>
      </div>
      {rangeError && <div className="form-error">{rangeError}</div>}

      {loading ? (
        <Loading />
      ) : error ? (
        <div className="error-state">
          <div>{error}</div>
          <Button onClick={() => load(query)}>重试</Button>
        </div>
      ) : !report || !hasData ? (
        <Empty title="该区间暂无数据" description="所选日期范围内没有语音会话记录。" />
      ) : (
        <>
          <div className="summary-grid">
            <div className="summary-card">
              <div className="summary-card-label">会话数</div>
              <div className="summary-card-value">{report.session_count}</div>
            </div>
            <div className="summary-card">
              <div className="summary-card-label">总时长</div>
              <div className="summary-card-value">{formatDuration(report.total_duration)}</div>
            </div>
            <div className="summary-card">
              <div className="summary-card-label">身份数</div>
              <div className="summary-card-value">{report.by_identity.length}</div>
            </div>
          </div>

          <div className="report-section">
            <h2 className="section-title">每日会话数趋势</h2>
            <div className="card">
              <TrendChart points={report.daily_trend} metric="session_count" />
            </div>
          </div>

          <div className="report-section">
            <h2 className="section-title">每日时长趋势</h2>
            <div className="card">
              <TrendChart
                points={report.daily_trend}
                metric="total_duration"
                formatValue={formatDurationShort}
              />
            </div>
          </div>

          <div className="report-section">
            <h2 className="section-title">身份占比</h2>
            <div className="card">
              <IdentityPie data={report.by_identity} />
            </div>
          </div>
        </>
      )}
    </div>
  );
}

export default Component;
