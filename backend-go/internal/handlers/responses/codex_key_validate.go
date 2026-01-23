package responses

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/handlers/common"
	"github.com/BenedictKing/claude-proxy/internal/httpclient"
	"github.com/BenedictKing/claude-proxy/internal/metrics"
	"github.com/BenedictKing/claude-proxy/internal/utils"
	"github.com/gin-gonic/gin"
)

type validateCodexKeyRequest struct {
	APIKey      string `json:"apiKey"`
	BaseURL     string `json:"baseUrl"`
	SummaryOnly bool   `json:"summaryOnly,omitempty"` // 仅获取 Right.Codes 余额/状态，不发起 /responses 探测
}

type validateCodexKeyResponse struct {
	Success       bool                      `json:"success"`
	StatusCode    int                       `json:"statusCode,omitempty"`
	UpstreamError string                    `json:"upstreamError,omitempty"`
	RightCodes    *rightCodesAccountSummary `json:"rightCodes,omitempty"`
}

type rightCodesSubscriptionSummary struct {
	UsedQuota      *float64 `json:"usedQuota,omitempty"`
	TotalQuota     *float64 `json:"totalQuota,omitempty"`
	RemainingQuota *float64 `json:"remainingQuota,omitempty"`
}

type rightCodesAccountSummary struct {
	Balance      float64                        `json:"balance"`
	Subscription *rightCodesSubscriptionSummary `json:"subscription,omitempty"`
	IsActive     *bool                          `json:"isActive,omitempty"`
}

type rightCodesAccountSummaryUpstream struct {
	Balance      float64 `json:"balance"`
	Subscription *struct {
		UsedQuota      *float64 `json:"used_quota"`
		TotalQuota     *float64 `json:"total_quota"`
		RemainingQuota *float64 `json:"remaining_quota"`
	} `json:"subscription"`
	IsActive *bool `json:"is_active"`
}

func ValidateCodexRightKey(metricsManager *metrics.MetricsManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req validateCodexKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.APIKey) == "" || strings.TrimSpace(req.BaseURL) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		baseURL := strings.TrimSpace(req.BaseURL)
		apiKey := strings.TrimSpace(req.APIKey)
		summaryOnly := req.SummaryOnly

		isRightCodes := isRightCodesBaseURL(baseURL)
		if summaryOnly && !isRightCodes {
			c.JSON(http.StatusBadRequest, gin.H{"error": "summary_only_requires_right_codes"})
			return
		}

		var rightCodesSummary *rightCodesAccountSummary
		if isRightCodes {
			summary, statusCode, upstreamErr, err := fetchRightCodesAccountSummary(c.Request.Context(), baseURL, apiKey)
			if err == nil && statusCode == http.StatusOK && summary != nil {
				rightCodesSummary = summary

				if summary.IsActive != nil && !*summary.IsActive {
					c.JSON(http.StatusOK, validateCodexKeyResponse{
						Success:       false,
						StatusCode:    http.StatusForbidden,
						UpstreamError: "账号已停用",
						RightCodes:    rightCodesSummary,
					})
					return
				}

				// 若能从 summary 明确判断“额度为 0”，直接失败（并硬熔断到本地 0 点自动恢复），避免无意义探测。
				if summary.Subscription != nil && summary.Subscription.RemainingQuota != nil && *summary.Subscription.RemainingQuota <= 0 {
					if metricsManager != nil {
						metricsManager.SuspendKeyUntil(baseURL, apiKey, utils.NextLocalMidnight(time.Now()), "insufficient_balance")
					}
					c.JSON(http.StatusOK, validateCodexKeyResponse{
						Success:       false,
						StatusCode:    http.StatusPaymentRequired,
						UpstreamError: "额度不足",
						RightCodes:    rightCodesSummary,
					})
					return
				}
			} else if err == nil && statusCode == http.StatusUnauthorized {
				// summary 明确返回 401 时，直接判定 Key 无效（无需再发起实际 /responses 探测）。
				c.JSON(http.StatusOK, validateCodexKeyResponse{
					Success:       false,
					StatusCode:    statusCode,
					UpstreamError: upstreamErr,
				})
				return
			} else if summaryOnly {
				// summaryOnly 模式：summary 拉取失败直接返回，不做 /responses 探测。
				if err != nil {
					c.JSON(http.StatusOK, validateCodexKeyResponse{
						Success:       false,
						StatusCode:    http.StatusBadGateway,
						UpstreamError: "上游错误: " + err.Error(),
					})
					return
				}
				if statusCode == 0 {
					statusCode = http.StatusBadGateway
				}
				c.JSON(http.StatusOK, validateCodexKeyResponse{
					Success:       false,
					StatusCode:    statusCode,
					UpstreamError: upstreamErr,
				})
				return
			}

			if summaryOnly {
				// summaryOnly 模式：仅返回 Right.Codes 信息，不做 /responses 探测。
				c.JSON(http.StatusOK, validateCodexKeyResponse{Success: true, RightCodes: rightCodesSummary})
				return
			}
		}

		ok, statusCode, upstreamErr, isInsufficientBalance, err := validateCodexKeyWithBaseURL(
			c.Request.Context(),
			baseURL,
			apiKey,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "validate_failed"})
			return
		}

		if ok {
			c.JSON(http.StatusOK, validateCodexKeyResponse{Success: true, RightCodes: rightCodesSummary})
			return
		}

		// 点击“检测”发现额度不足时，也需要立即进入硬熔断（到本地 0 点自动恢复），避免后续无意义重试。
		if isInsufficientBalance && metricsManager != nil {
			metricsManager.SuspendKeyUntil(baseURL, apiKey, utils.NextLocalMidnight(time.Now()), "insufficient_balance")
		}

		c.JSON(http.StatusOK, validateCodexKeyResponse{
			Success:       false,
			StatusCode:    statusCode,
			UpstreamError: upstreamErr,
			RightCodes:    rightCodesSummary,
		})
	}
}

func isRightCodesBaseURL(inputBaseURL string) bool {
	baseURL := strings.TrimSpace(inputBaseURL)
	if baseURL == "" {
		return false
	}
	if strings.HasSuffix(baseURL, "#") {
		baseURL = strings.TrimSuffix(baseURL, "#")
	}

	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		// 兜底：允许未带 scheme 的输入
		u2, err2 := url.Parse("https://" + baseURL)
		if err2 != nil || u2.Host == "" {
			return false
		}
		u = u2
	}

	host := strings.ToLower(u.Hostname())
	return host == "right.codes" || host == "www.right.codes" || strings.HasSuffix(host, ".right.codes")
}

func buildRightCodesAccountSummaryURL(inputBaseURL string) string {
	baseURL := strings.TrimSpace(inputBaseURL)
	if baseURL == "" {
		return ""
	}
	if strings.HasSuffix(baseURL, "#") {
		baseURL = strings.TrimSuffix(baseURL, "#")
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		u2, err2 := url.Parse("https://" + baseURL)
		if err2 != nil || u2.Scheme == "" || u2.Host == "" {
			return ""
		}
		u = u2
	}
	return u.Scheme + "://" + u.Host + "/account/summary"
}

func fetchRightCodesAccountSummary(ctx context.Context, baseURL string, apiKey string) (*rightCodesAccountSummary, int, string, error) {
	targetURL := buildRightCodesAccountSummaryURL(baseURL)
	if targetURL == "" {
		return nil, 0, "", nil
	}

	client := httpclient.GetManager().GetStandardClient(10*time.Second, false)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("User-Agent", "proxy-gateway/1.0")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()

	const maxRead = 32 * 1024
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxRead))
	bodyText := strings.TrimSpace(string(bodyBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, summarizeUpstreamError(resp.StatusCode, bodyText), nil
	}

	var upstream rightCodesAccountSummaryUpstream
	if err := json.Unmarshal(bodyBytes, &upstream); err != nil {
		return nil, resp.StatusCode, summarizeUpstreamError(resp.StatusCode, bodyText), nil
	}

	summary := &rightCodesAccountSummary{
		Balance:  upstream.Balance,
		IsActive: upstream.IsActive,
	}
	if upstream.Subscription != nil {
		summary.Subscription = &rightCodesSubscriptionSummary{
			UsedQuota:      upstream.Subscription.UsedQuota,
			TotalQuota:     upstream.Subscription.TotalQuota,
			RemainingQuota: upstream.Subscription.RemainingQuota,
		}
	}
	return summary, resp.StatusCode, "", nil
}

func validateCodexKeyWithBaseURL(ctx context.Context, baseURL string, apiKey string) (bool, int, string, bool, error) {
	payload := map[string]any{
		"model": "gpt-5.2",
		"input": []any{
			map[string]any{
				"type": "message",
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "input_text",
						"text": "你好",
					},
				},
			},
		},
		"stream": true,
	}
	body, _ := json.Marshal(payload)

	client := httpclient.GetManager().GetStandardClient(10*time.Second, false)
	targetURL := buildCodexResponsesURL(baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return false, 0, "", false, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return false, 0, "", false, err
	}
	defer resp.Body.Close()

	const maxRead = 8 * 1024
	peekBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxRead))
	peekText := strings.TrimSpace(string(peekBytes))
	isInsufficientBalance := isInsufficientBalanceBody(peekBytes, peekText)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, resp.StatusCode, summarizeUpstreamError(resp.StatusCode, peekText), isInsufficientBalance, nil
	}

	if looksLikeWrappedUpstreamError(peekText) {
		// 上游返回 2xx 但 body 呈现错误封装结构：视为失败。statusCode 取封装体内的 statusCode（若可解析），否则用 502。
		wrappedStatus, wrappedSummary := parseWrappedUpstreamError(peekText)
		if wrappedStatus == 0 {
			wrappedStatus = http.StatusBadGateway
		}
		return false, wrappedStatus, summarizeUpstreamError(wrappedStatus, wrappedSummary), isInsufficientBalance, nil
	}

	return true, 0, "", false, nil
}

func buildCodexResponsesURL(inputBaseURL string) string {
	baseURL := strings.TrimSpace(inputBaseURL)
	if baseURL == "" {
		return ""
	}

	skipVersionPrefix := strings.HasSuffix(baseURL, "#")
	if skipVersionPrefix {
		baseURL = strings.TrimSuffix(baseURL, "#")
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	hasVersionSuffix := false
	if strings.HasSuffix(baseURL, "/v1") || strings.HasSuffix(baseURL, "/v2") || strings.HasSuffix(baseURL, "/v3") ||
		strings.HasSuffix(baseURL, "/v4") || strings.HasSuffix(baseURL, "/v5") || strings.HasSuffix(baseURL, "/v6") ||
		strings.HasSuffix(baseURL, "/v7") || strings.HasSuffix(baseURL, "/v8") || strings.HasSuffix(baseURL, "/v9") {
		hasVersionSuffix = true
	} else if strings.HasPrefix(baseURL, "http") && strings.Contains(baseURL, "/v") {
		// 兜底：若末尾形如 /v1beta /v2alpha
		last := baseURL[strings.LastIndex(baseURL, "/")+1:]
		if len(last) >= 2 && last[0] == 'v' {
			digits := 0
			for i := 1; i < len(last); i++ {
				if last[i] < '0' || last[i] > '9' {
					break
				}
				digits++
			}
			if digits > 0 {
				hasVersionSuffix = true
			}
		}
	}

	if hasVersionSuffix || skipVersionPrefix {
		return baseURL + "/responses"
	}
	return baseURL + "/v1/responses"
}

func looksLikeWrappedUpstreamError(body string) bool {
	if body == "" {
		return false
	}
	if !strings.HasPrefix(body, "{") {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return false
	}
	if _, ok := m["statusCode"]; ok {
		return true
	}
	if _, ok := m["errorMessage"]; ok {
		return true
	}
	if _, ok := m["responseBody"]; ok {
		return true
	}
	return false
}

func extractWrappedErrorSummary(body string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return body
	}
	if v, ok := m["errorMessage"].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	if v, ok := m["responseBody"].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	if v, ok := m["error"].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return body
}

func parseWrappedUpstreamError(body string) (int, string) {
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return 0, body
	}
	code := 0
	if v, ok := m["statusCode"]; ok {
		switch t := v.(type) {
		case float64:
			code = int(t)
		case int:
			code = t
		}
	}
	return code, extractWrappedErrorSummary(body)
}

func isInsufficientBalanceBody(bodyBytes []byte, bodyText string) bool {
	if common.IsInsufficientBalanceResponse(bodyBytes) {
		return true
	}

	if bodyText == "" {
		return false
	}

	// 支持“2xx 但 body 是错误封装”的场景：优先从 responseBody / errorMessage 中提取。
	if looksLikeWrappedUpstreamError(bodyText) {
		if rb := extractWrappedResponseBody(bodyText); rb != "" {
			if common.IsInsufficientBalanceResponse([]byte(rb)) {
				return true
			}
			if looksLikeInsufficientBalanceText(rb) {
				return true
			}
		}
		if em := extractWrappedErrorMessage(bodyText); looksLikeInsufficientBalanceText(em) {
			return true
		}
	}

	// 兜底：body 非 JSON 或被截断时，仍能命中关键字。
	return looksLikeInsufficientBalanceText(bodyText)
}

func extractWrappedResponseBody(body string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return ""
	}
	if v, ok := m["responseBody"].(string); ok {
		return v
	}
	return ""
}

func extractWrappedErrorMessage(body string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return ""
	}
	if v, ok := m["errorMessage"].(string); ok {
		return v
	}
	return ""
}

func looksLikeInsufficientBalanceText(msg string) bool {
	m := strings.ToLower(msg)
	if strings.Contains(m, "余额不足") || strings.Contains(m, "额度不足") || strings.Contains(m, "积分不足") {
		return true
	}
	if strings.Contains(m, "insufficient") && strings.Contains(m, "balance") {
		return true
	}
	if strings.Contains(m, "insufficient") && (strings.Contains(m, "quota") || strings.Contains(m, "credit")) {
		return true
	}
	return false
}

func summarizeUpstreamError(statusCode int, raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "上游错误"
	}

	const maxChars = 512
	if len([]rune(s)) > maxChars {
		r := []rune(s)
		s = string(r[:maxChars]) + "…"
	}
	return "上游错误: " + statusCodeToText(statusCode) + " - " + s
}

func statusCodeToText(code int) string {
	if s := http.StatusText(code); s != "" {
		return s
	}
	return "HTTP " + strconv.Itoa(code)
}
