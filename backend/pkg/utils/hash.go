package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// SHA256Hex 返回输入字符串的 SHA-256 十六进制摘要。
func SHA256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
