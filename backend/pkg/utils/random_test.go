package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomToken(t *testing.T) {
	tok, err := RandomToken(32)
	require.NoError(t, err)
	assert.Len(t, tok, 64, "32 字节随机令牌应编码为 64 个十六进制字符")
}

func TestRandomTokenZeroBytes(t *testing.T) {
	tok, err := RandomToken(0)
	require.NoError(t, err)
	assert.Empty(t, tok)
}

func TestRandomTokenUnique(t *testing.T) {
	seen := make(map[string]bool, 256)
	for i := 0; i < 256; i++ {
		tok, err := RandomToken(16)
		require.NoError(t, err)
		assert.False(t, seen[tok], "随机令牌不应重复")
		seen[tok] = true
	}
}
