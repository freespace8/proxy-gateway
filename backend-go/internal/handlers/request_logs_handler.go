package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/BenedictKing/claude-proxy/internal/metrics"
	"github.com/gin-gonic/gin"
)

// RequestLogsHandler 处理请求日志 API
type RequestLogsHandler struct {
	store metrics.RequestLogStore
}

// NewRequestLogsHandler 创建 handler
func NewRequestLogsHandler(store metrics.RequestLogStore) *RequestLogsHandler {
	return &RequestLogsHandler{store: store}
}

// GetLogs 获取请求日志
// GET /api/{messages|responses|gemini}/logs?limit=50&offset=0
func (h *RequestLogsHandler) GetLogs(c *gin.Context) {
	if h == nil || h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "请求日志未启用"})
		return
	}

	apiType := apiTypeFromAdminLogsPath(c.FullPath())
	if apiType == "" {
		apiType = apiTypeFromAdminLogsPath(c.Request.URL.Path)
	}
	if apiType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 apiType"})
		return
	}

	limit := parseLimit(c.Query("limit"))
	offset := parseOffset(c.Query("offset"))

	logs, total, err := h.store.QueryRequestLogs(apiType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询请求日志失败"})
		return
	}

	c.JSON(http.StatusOK, metrics.RequestLogsResponse{
		Logs:   logs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func parseLimit(raw string) int {
	if raw == "" {
		return 50
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 50
	}
	if n > 200 {
		return 200
	}
	return n
}

func parseOffset(raw string) int {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func apiTypeFromAdminLogsPath(path string) string {
	// 期望格式：/api/{messages|responses|gemini}/logs
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return ""
	}
	if parts[0] != "api" || parts[2] != "logs" {
		return ""
	}

	apiType := parts[1]
	switch apiType {
	case "messages", "responses", "gemini":
		return apiType
	default:
		return ""
	}
}
