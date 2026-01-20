package responses

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/httpclient"
	"github.com/gin-gonic/gin"
)

type validateCodexKeyRequest struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseUrl"`
}

type validateCodexKeyResponse struct {
	Success       bool   `json:"success"`
	StatusCode    int    `json:"statusCode,omitempty"`
	UpstreamError string `json:"upstreamError,omitempty"`
}

func ValidateCodexRightKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req validateCodexKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.APIKey) == "" || strings.TrimSpace(req.BaseURL) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		ok, statusCode, upstreamErr, err := validateCodexKeyWithBaseURL(
			c.Request.Context(),
			strings.TrimSpace(req.BaseURL),
			strings.TrimSpace(req.APIKey),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "validate_failed"})
			return
		}

		if ok {
			c.JSON(http.StatusOK, validateCodexKeyResponse{Success: true})
			return
		}

		c.JSON(http.StatusOK, validateCodexKeyResponse{
			Success:       false,
			StatusCode:    statusCode,
			UpstreamError: upstreamErr,
		})
	}
}

func validateCodexKeyWithBaseURL(ctx context.Context, baseURL string, apiKey string) (bool, int, string, error) {
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
		return false, 0, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return false, 0, "", err
	}
	defer resp.Body.Close()

	const maxRead = 8 * 1024
	peekBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxRead))
	peekText := strings.TrimSpace(string(peekBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, resp.StatusCode, summarizeUpstreamError(resp.StatusCode, peekText), nil
	}

	if looksLikeWrappedUpstreamError(peekText) {
		// 上游返回 2xx 但 body 呈现错误封装结构：视为失败。statusCode 取封装体内的 statusCode（若可解析），否则用 502。
		wrappedStatus, wrappedSummary := parseWrappedUpstreamError(peekText)
		if wrappedStatus == 0 {
			wrappedStatus = http.StatusBadGateway
		}
		return false, wrappedStatus, summarizeUpstreamError(wrappedStatus, wrappedSummary), nil
	}

	return true, 0, "", nil
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
