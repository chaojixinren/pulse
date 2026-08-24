package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/repository"
)

// TestReportStatsCache 验证大范围统计命中 Redis 缓存后不再查询数据库。
func TestReportStatsCache(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	svc := NewReportService(repository.NewAudioSessionRepo(db), repository.NewIdentityRepo(db), rdb)
	mockAggQueries(mock, "u1")

	r1, err := svc.Stats(context.Background(), "u1", "2024-01-01", "2024-01-07")
	require.NoError(t, err)
	require.NotNil(t, r1)
	assert.NoError(t, mock.ExpectationsWereMet())

	// 第二次调用命中缓存，不再查询数据库（无额外 sqlmock 期望）。
	r2, err := svc.Stats(context.Background(), "u1", "2024-01-01", "2024-01-07")
	require.NoError(t, err)
	assert.Equal(t, r1.From, r2.From)
	assert.Equal(t, r1.To, r2.To)
	assert.Equal(t, r1.SessionCount, r2.SessionCount)
	assert.Equal(t, r1.TotalDuration, r2.TotalDuration)
}
