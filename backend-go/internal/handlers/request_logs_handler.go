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

type requestLogDetailResponse struct {
	Log *metrics.RequestLogRecord `json:"log"`
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

	// 列表接口仅返回概要信息，避免传输/渲染大字段（请求头/Body/cURL 由详情接口获取）。
	for i := range logs {
		logs[i].RequestMethod = ""
		logs[i].RequestURL = ""
		logs[i].RequestHeaders = nil
		logs[i].RequestBody = ""
		logs[i].RequestBodyTruncated = false
	}

	var totalRequests int64
	if stats, ok := h.store.(metrics.RequestLogStatsProvider); ok && stats != nil {
		totalRequests = stats.GetTotalRequestCount(apiType)
	}

	c.JSON(http.StatusOK, metrics.RequestLogsResponse{
		Logs:          logs,
		Total:         total,
		TotalRequests: totalRequests,
		Limit:         limit,
		Offset:        offset,
	})
}

// GetLogDetail 获取单条请求日志详情
// GET /api/{messages|responses|gemini}/logs/:id
func (h *RequestLogsHandler) GetLogDetail(c *gin.Context) {
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

	rawID := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 id"})
		return
	}

	detailProvider, ok := h.store.(metrics.RequestLogDetailProvider)
	if !ok || detailProvider == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "请求详情未启用"})
		return
	}

	logRecord, found := detailProvider.GetRequestLogDetail(apiType, id)
	if !found || logRecord == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "日志不存在"})
		return
	}

	c.JSON(http.StatusOK, requestLogDetailResponse{Log: logRecord})
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
