package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	"github.com/chaojixinren/pulse/internal/service"
	"github.com/chaojixinren/pulse/pkg/logger"
)

// AudioProcessor 轮询 pending 会话并执行转写。
type AudioProcessor struct {
	sessions  *repository.AudioSessionRepo
	stt       *service.SttService
	redis     *redis.Client
	interval  time.Duration
	batchSize int
}

func NewAudioProcessor(sessions *repository.AudioSessionRepo, stt *service.SttService, rdb *redis.Client) *AudioProcessor {
	return &AudioProcessor{
		sessions:  sessions,
		stt:       stt,
		redis:     rdb,
		interval:  5 * time.Second,
		batchSize: 5,
	}
}

func (w *AudioProcessor) Run(ctx context.Context) {
	logger.Log.Info("音频处理 worker 已启动", zap.Duration("interval", w.interval))
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("音频处理 worker 已停止")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *AudioProcessor) processBatch(ctx context.Context) {
	sessions, err := w.sessions.ListPending(ctx, w.batchSize)
	if err != nil {
		logger.Log.Error("拉取 pending 会话失败", zap.Error(err))
		return
	}
	for i := range sessions {
		w.processOne(ctx, &sessions[i])
	}
}

func (w *AudioProcessor) processOne(ctx context.Context, sess *model.AudioSession) {
	claimed, err := w.sessions.ClaimProcessing(ctx, sess.ID)
	if err != nil {
		logger.Log.Error("认领会话失败", zap.String("session_id", sess.ID), zap.Error(err))
		return
	}
	if !claimed {
		return
	}

	log := logger.Log.With(zap.String("session_id", sess.ID))
	log.Info("开始转写", zap.Int("audio_bytes", len(sess.AudioData)))

	mime := ""
	if sess.AudioMime != nil {
		mime = *sess.AudioMime
	}
	text, err := w.stt.Transcribe(ctx, sess.AudioData, service.FilenameForMime(mime))
	if err != nil {
		log.Error("转写失败", zap.Error(err))
		if uerr := w.sessions.UpdateStatus(ctx, sess.ID, model.StatusFailed, err.Error()); uerr != nil {
			log.Error("更新失败状态出错", zap.Error(uerr))
		}
		return
	}

	if err := w.sessions.UpdateTranscript(ctx, sess.ID, text); err != nil {
		log.Error("写入转写文本失败", zap.Error(err))
		if uerr := w.sessions.UpdateStatus(ctx, sess.ID, model.StatusFailed, err.Error()); uerr != nil {
			log.Error("更新失败状态出错", zap.Error(uerr))
		}
		return
	}
	if err := w.sessions.UpdateStatus(ctx, sess.ID, model.StatusCompleted, ""); err != nil {
		log.Error("更新完成状态失败", zap.Error(err))
		return
	}

	if w.redis != nil {
		key := fmt.Sprintf("session:%s", sess.ID)
		if err := w.redis.Set(ctx, key, text, 24*time.Hour).Err(); err != nil {
			log.Warn("写 Redis 缓存失败", zap.Error(err))
		}
	}
	log.Info("转写完成", zap.Int("text_len", len(text)))
}
