package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/repository"
)

func testKey32() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := testKey32()
	data := []byte("hello audio world, this is a longer payload for coverage")

	enc, err := EncryptAudio(data, key)
	require.NoError(t, err)
	assert.NotEqual(t, data, enc, "密文应与明文不同")
	assert.NotContains(t, string(enc), "hello audio", "密文不应直接包含明文")

	dec, err := DecryptAudio(enc, key)
	require.NoError(t, err)
	assert.Equal(t, data, dec)
}

func TestDecryptWrongKeyFails(t *testing.T) {
	key := testKey32()
	wrong := testKey32()
	wrong[0] ^= 0xFF

	enc, err := EncryptAudio([]byte("data"), key)
	require.NoError(t, err)
	_, err = DecryptAudio(enc, wrong)
	require.Error(t, err)
}

func TestDecryptTooShort(t *testing.T) {
	_, err := DecryptAudio([]byte("short"), testKey32())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "密文过短")
}

func TestAudioUploadEncryptsWhenKeySet(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	key := testKey32()
	svc := NewAudioService(&config.Config{MaxAudioSize: 1024, AudioEncryptionKey: key}, repository.NewAudioSessionRepo(db))

	data := validWAVBytes()
	recordedAt := time.Now().UTC()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audio_sessions")).
		WithArgs(sqlmock.AnyArg(), "u1", nil, nil, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, 10, int64(len(data)), "pending", nil, "{}", nil, recordedAt, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s, err := svc.Upload(context.Background(), "u1", UploadInput{
		Data: data, Filename: "clip.wav", Duration: 10, RecordedAt: recordedAt,
	})
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.NotEqual(t, data, s.AudioData, "落库的 audio_data 应为密文")

	dec, err := DecryptAudio(s.AudioData, key)
	require.NoError(t, err)
	assert.Equal(t, data, dec, "密文应能解密回原文")
	assert.Equal(t, int64(len(data)), *s.FileSize, "file_size 应记录原文大小")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioUploadNoKeyStoresPlaintext(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	svc := NewAudioService(&config.Config{MaxAudioSize: 1024}, repository.NewAudioSessionRepo(db))

	data := validWAVBytes()
	recordedAt := time.Now().UTC()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audio_sessions")).
		WithArgs(sqlmock.AnyArg(), "u1", nil, nil, data, sqlmock.AnyArg(), nil, 10, int64(len(data)), "pending", nil, "{}", nil, recordedAt, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s, err := svc.Upload(context.Background(), "u1", UploadInput{
		Data: data, Filename: "clip.wav", Duration: 10, RecordedAt: recordedAt,
	})
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, data, s.AudioData, "未配置密钥时应存明文")
	assert.NoError(t, mock.ExpectationsWereMet())
}
