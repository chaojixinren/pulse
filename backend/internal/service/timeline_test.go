package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/repository"
)

func newTimelineService(t *testing.T) (*TimelineService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewTimelineService(repository.NewAudioSessionRepo(db)), mock
}

func TestTimelineListClampsPagination(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		svc, mock := newTimelineService(t)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM audio_sessions")).
			WithArgs("u1").
			WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(0)))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime")).
			WithArgs("u1", 20, 0).
			WillReturnRows(sqlmock.NewRows(sessionListCols))

		items, total, err := svc.List(context.Background(), "u1", TimelineFilter{}, 0, 0)
		require.NoError(t, err)
		assert.Empty(t, items)
		assert.Equal(t, int64(0), total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("caps_size_at_100", func(t *testing.T) {
		svc, mock := newTimelineService(t)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM audio_sessions")).
			WithArgs("u1").
			WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(0)))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime")).
			WithArgs("u1", 100, 0).
			WillReturnRows(sqlmock.NewRows(sessionListCols))

		_, _, err := svc.List(context.Background(), "u1", TimelineFilter{}, 1, 1000)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTimelineListMapsFields(t *testing.T) {
	svc, mock := newTimelineService(t)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM audio_sessions")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(2)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime")).
		WithArgs("u1", 20, 0).
		WillReturnRows(sqlmock.NewRows(sessionListCols).
			AddRow("s1", "u1", "i1", nil, "audio/wav", "hello", 10, int64(5), "completed", nil, "{}", nil, now, nil, now, now).
			AddRow("s2", "u1", nil, nil, "audio/mpeg", nil, 5, int64(3), "pending", nil, "{}", nil, now, nil, now, now))

	items, total, err := svc.List(context.Background(), "u1", TimelineFilter{}, 1, 20)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, int64(2), total)

	assert.Equal(t, "s1", items[0].SessionID)
	assert.Equal(t, "i1", items[0].IdentityID)
	assert.Equal(t, "hello", items[0].Transcript)
	assert.Equal(t, "completed", items[0].Status)
	assert.Equal(t, 10, items[0].Duration)

	assert.Equal(t, "", items[1].IdentityID, "无身份时 IdentityID 应为空")
	assert.Equal(t, "", items[1].Transcript, "无转写时 Transcript 应为空")
	assert.NoError(t, mock.ExpectationsWereMet())
}
