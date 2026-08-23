package utils

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// RFC 4122 version 4, variant 10xx。
var uuidV4Re = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")

func TestNewUUIDFormat(t *testing.T) {
	for i := 0; i < 1000; i++ {
		id := NewUUID()
		assert.Regexp(t, uuidV4Re, id, "生成的 UUID 应为 RFC4122 v4 格式: %s", id)
	}
}

func TestNewUUIDUnique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := NewUUID()
		assert.False(t, seen[id], "UUID 不应重复")
		seen[id] = true
	}
}
