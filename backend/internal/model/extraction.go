package model

import "time"

// Todo 是从对话中提取的一条待办。
type Todo struct {
	Text  string     `json:"text"`
	DueAt *time.Time `json:"due_at,omitempty"`
}

// Commitment 是从对话中提取的一条承诺。
type Commitment struct {
	Text  string     `json:"text"`
	From  string     `json:"from"`
	To    string     `json:"to"`
	DueAt *time.Time `json:"due_at,omitempty"`
}

// ExtractedData 是 AI 信息提取的结构化结果。
type ExtractedData struct {
	Todos       []Todo       `json:"todos"`
	Commitments []Commitment `json:"commitments"`
	Notes       []string     `json:"notes"`
}

// AnalysisResult 是 AI 分析的最终结果。
type AnalysisResult struct {
	IdentityID  *string       `json:"identity_id,omitempty"`
	Confidence  float64       `json:"confidence"`
	Extracted   ExtractedData `json:"extracted"`
	RawResponse string        `json:"raw_response"`
	Todos       []Todo        `json:"todos"`
	Commitments []Commitment  `json:"commitments"`
	Notes       []string      `json:"notes"`
}
