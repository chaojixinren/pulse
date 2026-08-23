package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSHA256Hex(t *testing.T) {
	assert.Equal(t,
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		SHA256Hex(""),
		"空串的 SHA256 应与已知摘要一致",
	)

	assert.Equal(t,
		"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		SHA256Hex("abc"),
		"abc 的 SHA256 应与已知摘要一致",
	)
}

func TestSHA256HexDeterministic(t *testing.T) {
	const input = "pulse-refresh-token"
	assert.Equal(t, SHA256Hex(input), SHA256Hex(input), "相同输入应产生相同摘要")
	assert.Len(t, SHA256Hex(input), 64, "SHA256 十六进制摘要应为 64 字符")
}
