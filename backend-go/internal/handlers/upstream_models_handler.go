package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/httpclient"
	"github.com/gin-gonic/gin"
)

type probeUpstreamModelsRequest struct {
	BaseURL            string `json:"baseUrl"`
	APIKey             string `json:"apiKey"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
}

type probeUpstreamModelsResponse struct {
	Success       bool           `json:"success"`
	StatusCode    int            `json:"statusCode,omitempty"`
	UpstreamError string         `json:"upstreamError,omitempty"`
	Models        *modelsResponse `json:"models,omitempty"`
}

type modelsResponse struct {
	Object string       `json:"object"`
	Data   []modelEntry `json:"data"`
}

type modelEntry struct {
	ID      string `json:"id"`
	Object  string `json:"object,omitempty"`
	Created int64  `json:"created,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
}

var modelsVersionSuffixPattern = regexp.MustCompile(`/v\d+[a-z]*$`)

func ProbeUpstreamModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req probeUpstreamModelsRequest
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.BaseURL) == "" || strings.TrimSpace(req.APIKey) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		targetURL := buildModelsURL(strings.TrimSpace(req.BaseURL))
		if targetURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid baseUrl"})
			return
		}

		client := httpclient.GetManager().GetStandardClient(10*time.Second, req.InsecureSkipVerify)
		httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, targetURL, nil)
		if err != nil {
			c.JSON(http.StatusOK, probeUpstreamModelsResponse{
				Success:       false,
				StatusCode:    http.StatusBadGateway,
				UpstreamError: "上游错误: " + err.Error(),
			})
			return
		}
		httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(req.APIKey))

		resp, err := client.Do(httpReq)
		if err != nil {
			c.JSON(http.StatusOK, probeUpstreamModelsResponse{
				Success:       false,
				StatusCode:    http.StatusBadGateway,
				UpstreamError: "上游错误: " + err.Error(),
			})
			return
		}
		defer resp.Body.Close()

		const maxRead = 32 * 1024
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxRead))

		bodyText := strings.TrimSpace(string(bodyBytes))
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			c.JSON(http.StatusOK, probeUpstreamModelsResponse{
				Success:       false,
				StatusCode:    resp.StatusCode,
				UpstreamError: summarizeUpstreamError(resp.StatusCode, bodyText),
			})
			return
		}

		var models modelsResponse
		if err := json.Unmarshal(bodyBytes, &models); err != nil {
			c.JSON(http.StatusOK, probeUpstreamModelsResponse{
				Success:       false,
				StatusCode:    http.StatusBadGateway,
				UpstreamError: summarizeUpstreamError(http.StatusBadGateway, bodyText),
			})
			return
		}

		c.JSON(http.StatusOK, probeUpstreamModelsResponse{
			Success:    true,
			StatusCode: http.StatusOK,
			Models:     &models,
		})
	}
}

func buildModelsURL(inputBaseURL string) string {
	baseURL := strings.TrimSpace(inputBaseURL)
	if baseURL == "" {
		return ""
	}

	skipVersionPrefix := strings.HasSuffix(baseURL, "#")
	if skipVersionPrefix {
		baseURL = strings.TrimSuffix(baseURL, "#")
	}

	baseURL = strings.TrimSuffix(baseURL, "/")

	hasVersionSuffix := modelsVersionSuffixPattern.MatchString(baseURL)

	endpoint := "/models"
	if !hasVersionSuffix && !skipVersionPrefix {
		endpoint = "/v1" + endpoint
	}

	return baseURL + endpoint
}

func summarizeUpstreamError(statusCode int, raw string) string {
	msg := extractUpstreamErrorMessage(raw)
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "上游错误"
	}

	const maxChars = 512
	if len([]rune(msg)) > maxChars {
		r := []rune(msg)
		msg = string(r[:maxChars]) + "…"
	}

	return "上游错误: " + statusCodeToText(statusCode) + " - " + msg
}

func extractUpstreamErrorMessage(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}

	var parsed struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		return s
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return parsed.Error.Message
	}
	if strings.TrimSpace(parsed.Message) != "" {
		return parsed.Message
	}
	return s
}

func statusCodeToText(code int) string {
	if s := http.StatusText(code); s != "" {
		return s
	}
	return "HTTP " + strconv.Itoa(code)
}
