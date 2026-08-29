import { useCallback, useEffect, useState } from 'react';
import { Button } from '@/components/common/Button';
import { Empty } from '@/components/common/Empty';
import { Loading } from '@/components/common/Loading';
import { ExtractedList } from '@/components/business/ExtractedList';
import { reportService } from '@/services/report.service';
import type { DailyReport } from '@/types/report.types';
import { shiftDate, todayStr } from '@/utils/date';
import { formatDuration } from '@/utils/format';

export function Component() {
  const [date, setDate] = useState<string>(() => todayStr());
  const [report, setReport] = useState<DailyReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (targetDate: string) => {
    setLoading(true);
    setError(null);
    try {
      const data = await reportService.daily(targetDate);
      setReport(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败');
      setReport(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(date);
  }, [date, load]);

  const prevDay = () => setDate((d) => shiftDate(d, -1));
  const nextDay = () => setDate((d) => shiftDate(d, 1));

  const hasData = Boolean(
    report &&
      (report.session_count > 0 ||
        report.by_identity.length > 0 ||
        report.todos.length > 0 ||
        report.notes.length > 0),
  );

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">日报</h1>
          <p className="page-subtitle">查看某日的会话汇总与 AI 提取的待办、笔记。</p>
        </div>
      </div>

      <div className="report-toolbar">
        <Button variant="secondary" onClick={prevDay}>
          ← 前一天
        </Button>
        <input
          type="date"
          className="form-input"
          style={{ maxWidth: 160 }}
          value={date}
          onChange={(e) => setDate(e.target.value)}
        />
        <Button variant="secondary" onClick={nextDay}>
          后一天 →
        </Button>
      </div>

      {loading ? (
        <Loading />
      ) : error ? (
        <div className="error-state">
          <div>{error}</div>
          <Button onClick={() => load(date)}>重试</Button>
        </div>
      ) : !report || !hasData ? (
        <Empty
          title="该日期暂无数据"
          description="这一天没有语音会话记录。"
        />
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
            <h2 className="section-title">按身份拆分</h2>
            <div className="card">
              {report.by_identity.length === 0 ? (
                <Empty title="暂无身份统计" />
              ) : (
                report.by_identity.map((stat) => (
                  <div key={stat.identity_id} className="identity-stat-row">
                    <span className="identity-stat-name">{stat.name || '未分配'}</span>
                    <span className="identity-stat-value">
                      {stat.session_count} 次 · {formatDuration(stat.total_duration)}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>

          <div className="report-section">
            <h2 className="section-title">待办</h2>
            <div className="card">
              <ExtractedList
                items={report.todos}
                emptyTitle="暂无待办"
                checkable
              />
            </div>
          </div>

          <div className="report-section">
            <h2 className="section-title">笔记</h2>
            <div className="card">
              <ExtractedList items={report.notes} emptyTitle="暂无笔记" />
            </div>
          </div>
        </>
      )}
    </div>
  );
}

export default Component;
