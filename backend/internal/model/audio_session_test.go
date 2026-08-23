package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidStatus(t *testing.T) {
	for _, s := range []string{StatusPending, StatusProcessing, StatusCompleted, StatusFailed} {
		assert.True(t, IsValidStatus(s), "status %q 应为合法状态", s)
	}
	for _, s := range []string{"", "unknown", "PENDING", "done", "error"} {
		assert.False(t, IsValidStatus(s), "status %q 应为非法状态", s)
	}
}

func TestCanTransition(t *testing.T) {
	// 合法流转
	assert.True(t, CanTransition(StatusPending, StatusProcessing))
	assert.True(t, CanTransition(StatusPending, StatusFailed))
	assert.True(t, CanTransition(StatusProcessing, StatusCompleted))
	assert.True(t, CanTransition(StatusProcessing, StatusFailed))
	assert.True(t, CanTransition(StatusFailed, StatusProcessing))
	assert.False(t, CanTransition(StatusFailed, StatusPending), "重试应直接进入 processing 而非 pending")

	// 非法流转：completed 是终态，不允许任何跳转
	for _, to := range []string{StatusPending, StatusProcessing, StatusCompleted, StatusFailed} {
		assert.False(t, CanTransition(StatusCompleted, to), "completed -> %s 应被禁止", to)
	}

	// 非法流转：不允许原地停留 / 回退
	assert.False(t, CanTransition(StatusPending, StatusPending))
	assert.False(t, CanTransition(StatusPending, StatusCompleted))
	assert.False(t, CanTransition(StatusProcessing, StatusProcessing))
	assert.False(t, CanTransition(StatusProcessing, StatusPending))
	assert.False(t, CanTransition(StatusFailed, StatusFailed))
	assert.False(t, CanTransition(StatusFailed, StatusCompleted))

	// 未知状态
	assert.False(t, CanTransition("unknown", StatusProcessing))
	assert.False(t, CanTransition(StatusPending, "unknown"))
}
