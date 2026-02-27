package metrics

// RequestLogStore 抽象请求日志存储（用于管理端 /api/{type}/logs）。
// 目标：使用内存环形缓冲保存最近 N 条请求日志。
type RequestLogStore interface {
	AddRequestLog(logRecord RequestLogRecord) error
	QueryRequestLogs(apiType string, limit, offset int) ([]RequestLogRecord, int64, error)
}

// RequestLogDetailProvider 提供按单条记录查询请求详情的能力（用于管理端查看请求头/Body/cURL）。
type RequestLogDetailProvider interface {
	// GetRequestLogDetail 按 id 获取请求日志详情；不存在返回 (nil, false)。
	GetRequestLogDetail(apiType string, id int64) (*RequestLogRecord, bool)
}

// RequestLogStatsProvider 提供请求日志的计数与“清空/重置”能力。
//
// 说明：用于让“渠道编排”按 Key 显示累计请求数，并支持“重置统计”同时清空对应请求日志。
// 该统计口径不受日志环形缓冲容量影响（本次进程内累计递增），但会受 ResetKey/ResetChannel 影响而清零。
type RequestLogStatsProvider interface {
	// GetTotalRequestCount 返回指定 apiType 的累计请求数（全渠道汇总）。
	GetTotalRequestCount(apiType string) int64
	// GetKeyRequestCount 返回指定 Key 的累计请求数（按 channelIndex + keyId）。
	GetKeyRequestCount(apiType string, channelIndex int, keyID string) int64
	// ResetKey 清空单个 Key 的累计计数与可查询日志（按 channelIndex + keyId）。
	ResetKey(apiType string, channelIndex int, keyID string)
	// ResetChannel 清空单个渠道的累计计数与可查询日志（按 channelIndex）。
	ResetChannel(apiType string, channelIndex int)
}
