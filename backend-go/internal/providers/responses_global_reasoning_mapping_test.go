package providers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/BenedictKing/claude-proxy/internal/session"
	"github.com/gin-gonic/gin"
)

func TestResponsesProvider_ConvertModeAppliesGlobalReasoningMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/responses", bytes.NewBufferString(`{"model":"gpt-5.1","input":"hi","reasoning":{"effort":"low"}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(ContextKeyResponsesGlobalReasoningMapping, map[string]string{"low": "xhigh"})

	provider := &ResponsesProvider{
		SessionManager: session.NewSessionManager(time.Hour, 100, 100000),
	}
	upstream := &config.UpstreamConfig{
		BaseURL:     "https://api.example.com/v1",
		ServiceType: "openai",
	}

	req, _, err := provider.ConvertToProviderRequest(c, upstream, "rk-test")
	if err != nil {
		t.Fatalf("ConvertToProviderRequest err=%v", err)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}

	if got := payload["reasoning_effort"]; got != "xhigh" {
		t.Fatalf("reasoning_effort=%v, want %q", got, "xhigh")
	}
}

func TestResponsesProvider_ConvertModeAppliesGlobalReasoningEffortFieldMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/responses", bytes.NewBufferString(`{"model":"gpt-5.1","input":"hi","reasoning_effort":"medium"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(ContextKeyResponsesGlobalReasoningMapping, map[string]string{"medium": "high"})

	provider := &ResponsesProvider{
		SessionManager: session.NewSessionManager(time.Hour, 100, 100000),
	}
	upstream := &config.UpstreamConfig{
		BaseURL:     "https://api.example.com/v1",
		ServiceType: "openai",
	}

	req, _, err := provider.ConvertToProviderRequest(c, upstream, "rk-test")
	if err != nil {
		t.Fatalf("ConvertToProviderRequest err=%v", err)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}

	if got := payload["reasoning_effort"]; got != "high" {
		t.Fatalf("reasoning_effort=%v, want %q", got, "high")
	}
}
