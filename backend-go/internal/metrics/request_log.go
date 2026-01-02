package metrics

import "time"

// RequestLogRecord 请求日志记录（用于持久化和 API 返回）
type RequestLogRecord struct {
	ID                  int64     `json:"id"`
	RequestID           string    `json:"requestId"`
	ChannelIndex        int       `json:"channelIndex"`
	ChannelName         string    `json:"channelName"`
	KeyMask             string    `json:"keyMask"`
	Timestamp           time.Time `json:"timestamp"`
	DurationMs          int64     `json:"durationMs"`
	StatusCode          int       `json:"statusCode"`
	Success             bool      `json:"success"`
	Model               string    `json:"model"`
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
	Logs   []RequestLogRecord `json:"logs"`
	Total  int64              `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}
