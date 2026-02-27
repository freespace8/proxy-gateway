package providers

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/gin-gonic/gin"
)

func TestResponsesProvider_ForceSU8CodexResponsesUserAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/responses", bytes.NewBufferString(`{"model":"gpt-5.3-codex","input":"hi"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("User-Agent", "CustomClient/1.0")

	upstream := &config.UpstreamConfig{
		BaseURL:     "https://www.su8.codes/codex/v1",
		ServiceType: "responses",
	}

	req, _, err := (&ResponsesProvider{}).ConvertToProviderRequest(c, upstream, "rk-test")
	if err != nil {
		t.Fatalf("ConvertToProviderRequest err=%v", err)
	}

	if req.URL.String() != "https://www.su8.codes/codex/v1/responses" {
		t.Fatalf("url=%q", req.URL.String())
	}
	if got := req.Header.Get("User-Agent"); got != "codex_cli_rs/0.105.0 (Mac OS 26.3.0; arm64) vscode/1.110.0-insider" {
		t.Fatalf("User-Agent=%q", got)
	}
}

func TestResponsesProvider_KeepUserAgentForOtherTargets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/responses", bytes.NewBufferString(`{"model":"gpt-5.3-codex","input":"hi"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("User-Agent", "CustomClient/1.0")

	upstream := &config.UpstreamConfig{
		BaseURL:     "https://api.example.com/v1",
		ServiceType: "responses",
	}

	req, _, err := (&ResponsesProvider{}).ConvertToProviderRequest(c, upstream, "rk-test")
	if err != nil {
		t.Fatalf("ConvertToProviderRequest err=%v", err)
	}

	if got := req.Header.Get("User-Agent"); got != "CustomClient/1.0" {
		t.Fatalf("User-Agent=%q, want=%q", got, "CustomClient/1.0")
	}
}
