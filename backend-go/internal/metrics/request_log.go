package metrics

import "time"

// RequestLogRecord 请求日志记录（用于持久化和 API 返回）
type RequestLogRecord struct {
	ID                  int64     `json:"id"`
	RequestID           string    `json:"requestId"`
	ChannelIndex        int       `json:"channelIndex"`
	ChannelName         string    `json:"channelName"`
	KeyMask             string    `json:"keyMask"`
	KeyID               string    `json:"keyId,omitempty"`
	Timestamp           time.Time `json:"timestamp"`
	DurationMs          int64     `json:"durationMs"`
	StatusCode          int       `json:"statusCode"`
	Success             bool      `json:"success"`
	Model               string    `json:"model"`
	ReasoningEffort     string    `json:"reasoningEffort,omitempty"`
	InputTokens         int64     `json:"inputTokens"`
	OutputTokens        int64     `json:"outputTokens"`
	CacheCreationTokens int64     `json:"cacheCreationTokens"`
	CacheReadTokens     int64     `json:"cacheReadTokens"`
	CostCents           int64     `json:"costCents"`
	ErrorMessage        string    `json:"errorMessage,omitempty"`
	APIType             string    `json:"apiType"` // messages, responses, gemini
}

// RequestLogsResponse API 响应
type RequestLogsResponse struct {
	Logs  []RequestLogRecord `json:"logs"`
	Total int64              `json:"total"`
	// TotalRequests 为本次进程内的累计请求数（受“重置统计/清空日志”影响，不受日志容量上限影响）。
	TotalRequests int64 `json:"totalRequests,omitempty"`
	Limit         int   `json:"limit"`
	Offset        int   `json:"offset"`
}
