import { useCallback, useEffect, useState } from 'react';
import { Button } from '@/components/common/Button';
import { Empty } from '@/components/common/Empty';
import { Loading } from '@/components/common/Loading';
import { ExtractedList } from '@/components/business/ExtractedList';
import { IdentityPie } from '@/components/business/IdentityPie';
import { TrendChart } from '@/components/business/TrendChart';
import { reportService } from '@/services/report.service';
import type { WeeklyReport as WeeklyReportData } from '@/types/report.types';
import { mondayOf, shiftDate, todayStr } from '@/utils/date';
import { formatDuration } from '@/utils/format';

export function Component() {
  const [week, setWeek] = useState<string>(() => mondayOf(todayStr()));
  const [report, setReport] = useState<WeeklyReportData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (targetWeek: string) => {
    setLoading(true);
    setError(null);
    try {
      setReport(await reportService.weekly(targetWeek));
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败');
      setReport(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(week);
  }, [week, load]);

  const prevWeek = () => setWeek((w) => shiftDate(w, -7));
  const nextWeek = () => setWeek((w) => shiftDate(w, 7));

  const hasData = Boolean(
    report &&
      (report.session_count > 0 ||
        report.daily_trend.length > 0 ||
        report.by_identity.length > 0 ||
        report.top_todos.length > 0),
  );

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">周报</h1>
          <p className="page-subtitle">查看一周的会话趋势、身份占比与 AI 提取。</p>
        </div>
      </div>

      <div className="report-toolbar">
        <Button variant="secondary" onClick={prevWeek}>
          ← 上一周
        </Button>
        <span className="report-date">
          {week} ~ {shiftDate(week, 6)}
        </span>
        <Button variant="secondary" onClick={nextWeek}>
          下一周 →
        </Button>
      </div>

      {loading ? (
        <Loading />
      ) : error ? (
        <div className="error-state">
          <div>{error}</div>
          <Button onClick={() => load(week)}>重试</Button>
        </div>
      ) : !report || !hasData ? (
        <Empty icon="📅" title="本周暂无数据" description="这一周还没有语音会话记录。" />
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
              <div className="summary-card-label">完成承诺</div>
              <div className="summary-card-value">{report.commitments_done}</div>
            </div>
            <div className="summary-card">
              <div className="summary-card-label">身份数</div>
              <div className="summary-card-value">{report.by_identity.length}</div>
            </div>
          </div>

          <div className="report-section">
            <h2 className="section-title">每日趋势</h2>
            <div className="card">
              <TrendChart points={report.daily_trend} metric="session_count" />
            </div>
          </div>

          <div className="report-section">
            <h2 className="section-title">身份占比</h2>
            <div className="card">
              <IdentityPie data={report.by_identity} />
            </div>
          </div>

          <div className="report-section">
            <h2 className="section-title">Top 待办</h2>
            <div className="card">
              <ExtractedList
                items={report.top_todos}
                emptyIcon="✅"
                emptyTitle="暂无待办"
                checkable
              />
            </div>
          </div>
        </>
      )}
    </div>
  );
}

export default Component;
