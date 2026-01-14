package metrics

// RequestLogStore 抽象请求日志存储（用于管理端 /api/{type}/logs）。
// 目标：允许在不启用 SQLite 的情况下使用内存环形缓冲保存最近 N 条请求日志。
type RequestLogStore interface {
	AddRequestLog(logRecord RequestLogRecord) error
	QueryRequestLogs(apiType string, limit, offset int) ([]RequestLogRecord, int64, error)
}
