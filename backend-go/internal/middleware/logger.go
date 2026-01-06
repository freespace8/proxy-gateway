package middleware

import (
	"net/http"
	"strings"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/gin-gonic/gin"
)

// 默认跳过日志的路径前缀（仅 GET 请求）
// 这些端点在任何情况下都不输出日志（用于降低高频轮询噪音）
var alwaysSkipPrefixes = []string{
	"/api/responses/live",
	"/api/responses/logs",
}

// 默认跳过日志的路径前缀（仅 GET 请求）
var defaultSkipPrefixes = []string{
	"/api/messages/channels",
	"/api/responses/channels",
	"/api/gemini/channels",
	"/api/messages/global/stats",
	"/api/responses/global/stats",
	"/api/gemini/global/stats",
}

// FilteredLogger 创建一个可过滤路径的 Logger 中间件
// 仅对 GET 请求且匹配 skipPrefixes 前缀的路径跳过日志输出
// POST/PUT/DELETE 等管理操作始终记录日志以保留审计跟踪
func FilteredLogger(envCfg *config.EnvConfig, skipPrefixes ...string) gin.HandlerFunc {
	var quietSkipPrefixes []string
	if envCfg.QuietPollingLogs {
		if len(skipPrefixes) == 0 {
			quietSkipPrefixes = defaultSkipPrefixes
		} else {
			quietSkipPrefixes = skipPrefixes
		}
	}

	return gin.LoggerWithConfig(gin.LoggerConfig{
		Skip: func(c *gin.Context) bool {
			// 只跳过 GET 请求，保留其他方法的审计日志
			if c.Request.Method != http.MethodGet {
				return false
			}

			path := c.Request.URL.Path

			// 永久静默（不受 QuietPollingLogs 影响）
			for _, prefix := range alwaysSkipPrefixes {
				if strings.HasPrefix(path, prefix) {
					return true
				}
			}

			// QuietPollingLogs 为 false 时，仅应用永久静默规则
			if !envCfg.QuietPollingLogs {
				return false
			}

			// QuietPollingLogs 为 true 时，额外应用可配置的轮询端点过滤
			for _, prefix := range quietSkipPrefixes {
				if strings.HasPrefix(path, prefix) {
					return true
				}
			}
			return false
		},
	})
}
