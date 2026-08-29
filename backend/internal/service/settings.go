package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// secretPrefix 标记密文，便于回读时区分「已加密」与「明文/空」。
const secretPrefix = "enc:"

// EncryptSecret 使用 AES-256-GCM 加密敏感字符串，返回 "enc:" + base64(nonce||ciphertext)。
// key 为空时退回明文存储（与音频加密一致：未配置密钥则关闭加密）。
func EncryptSecret(plain string, key []byte) (string, error) {
	if plain == "" {
		return "", nil
	}
	if len(key) == 0 {
		return plain, nil
	}
	ct, err := EncryptAudio([]byte(plain), key)
	if err != nil {
		return "", err
	}
	return secretPrefix + base64.StdEncoding.EncodeToString(ct), nil
}

// DecryptSecret 解密 EncryptSecret 产生的密文；无 "enc:" 前缀时视为明文直接返回。
func DecryptSecret(s string, key []byte) (string, error) {
	if s == "" || !strings.HasPrefix(s, secretPrefix) {
		return s, nil
	}
	if len(key) == 0 {
		// 密文但无密钥：无法解密，返回空并交给调用方回退全局默认。
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s, secretPrefix))
	if err != nil {
		return "", err
	}
	pt, err := DecryptAudio(raw, key)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// maskSecret 将密钥脱敏为「尾部 4 位可见」的掩码；过短则整体遮蔽。
func maskSecret(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}

// userSettings 是 users.settings 列的顶层结构。仅承载用户可配置的模型项，
// 未来如需扩展其他偏好项，在此追加字段即可（保持向后兼容）。
type userSettings struct {
	ASR *AsrSettings `json:"asr,omitempty"`
	AI  *AiSettings  `json:"ai,omitempty"`
}

// AsrSettings 是用户级语音转写（ASR）配置。字段留空即回退全局默认。
// EnableITN 用指针：nil 表示未设置（回退默认 true），非 nil 表示显式开关。
type AsrSettings struct {
	APIKey    string `json:"api_key,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	Model     string `json:"model,omitempty"`
	Language  string `json:"language,omitempty"`
	EnableITN *bool  `json:"enable_itn,omitempty"`
}

// AiSettings 是用户级 AI 分析（身份识别 + 信息提取）配置。
type AiSettings struct {
	APIKey              string   `json:"api_key,omitempty"`
	BaseURL             string   `json:"base_url,omitempty"`
	Model               string   `json:"model,omitempty"`
	ConfidenceThreshold *float64 `json:"confidence_threshold,omitempty"`
}

// parseUserSettings 解析 settings JSON；空串或非法 JSON 返回空结构。
func parseUserSettings(raw string) *userSettings {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return &userSettings{}
	}
	var s userSettings
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return &userSettings{}
	}
	return &s
}

// serializeUserSettings 序列化 settings，保证输出合法 JSON（空对象也返回 "{}"）。
func serializeUserSettings(s *userSettings) (string, error) {
	if s == nil {
		return "{}", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}
