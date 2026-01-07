package common

import (
	"github.com/BenedictKing/claude-proxy/internal/metrics"
)

// RecordFailureAndStoreLastFailureLog 记录失败并写入“最后一次失败日志”。
func RecordFailureAndStoreLastFailureLog(
	store *metrics.SQLiteStore,
	metricsManager *metrics.MetricsManager,
	apiType string,
	baseURL string,
	apiKey string,
	statusCode int,
	respBody []byte,
	err error,
	recordFailure func(),
) {
	if metricsManager == nil || recordFailure == nil {
		return
	}

	recordFailure()

	if store == nil {
		return
	}

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	// 记录“最后一次失败日志”（无论是否已进入熔断）
	logStr := metrics.BuildKeyCircuitLogJSON(baseURL, statusCode, string(respBody), errMsg)
	_ = store.UpsertKeyCircuitLog(apiType, metrics.HashAPIKey(apiKey), logStr)
}
