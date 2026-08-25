export function Loading({ text = '加载中…' }: { text?: string }) {
  return (
    <div className="loading">
      <div className="loading-spinner" />
      <span>{text}</span>
    </div>
  );
}

export function FullScreenLoading() {
  return (
    <div className="loading-fullscreen">
      <Loading />
    </div>
  );
}
