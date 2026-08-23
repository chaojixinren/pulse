package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// RandomToken 生成 nBytes 字节的加密随机令牌（十六进制编码）。
func RandomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
