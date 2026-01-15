package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/gin-gonic/gin"
)

func TestPatchAPIKeyMeta_ToggleDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{Name: "m0", ServiceType: "claude", BaseURL: "https://m0.example.com", APIKeys: []string{"k1", "k2"}, Status: "active"},
		},
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		GeminiLoadBalance:    "failover",
		FuzzyModeEnabled:     true,
	}
	cm, _ := newTestConfigManager(t, cfg)

	r := gin.New()
	r.PATCH("/channels/:id/keys/index/:keyIndex/meta", PatchAPIKeyMeta(cm, "messages"))

	reqEnable := httptest.NewRequest(http.MethodPatch, "/channels/0/keys/index/0/meta", bytes.NewBufferString(`{"disabled":true}`))
	reqEnable.Header.Set("Content-Type", "application/json")
	wEnable := httptest.NewRecorder()
	r.ServeHTTP(wEnable, reqEnable)
	if wEnable.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", wEnable.Code, http.StatusOK)
	}

	afterDisable := cm.GetConfig()
	if afterDisable.Upstream[0].APIKeyMeta == nil || !afterDisable.Upstream[0].APIKeyMeta["k1"].Disabled {
		t.Fatalf("expected k1 disabled=true, got=%v", afterDisable.Upstream[0].APIKeyMeta)
	}

	reqDisable := httptest.NewRequest(http.MethodPatch, "/channels/0/keys/index/0/meta", bytes.NewBufferString(`{"disabled":false}`))
	reqDisable.Header.Set("Content-Type", "application/json")
	wDisable := httptest.NewRecorder()
	r.ServeHTTP(wDisable, reqDisable)
	if wDisable.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", wDisable.Code, http.StatusOK)
	}

	afterEnable := cm.GetConfig()
	if afterEnable.Upstream[0].APIKeyMeta != nil {
		if _, ok := afterEnable.Upstream[0].APIKeyMeta["k1"]; ok {
			t.Fatalf("expected k1 meta removed when disabled=false and empty description, got=%v", afterEnable.Upstream[0].APIKeyMeta["k1"])
		}
	}
}

func TestPatchAPIKeyMeta_InvalidKeyIndexReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{Name: "m0", ServiceType: "claude", BaseURL: "https://m0.example.com", APIKeys: []string{"k1"}, Status: "active"},
		},
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		GeminiLoadBalance:    "failover",
		FuzzyModeEnabled:     true,
	}
	cm, _ := newTestConfigManager(t, cfg)

	r := gin.New()
	r.PATCH("/channels/:id/keys/index/:keyIndex/meta", PatchAPIKeyMeta(cm, "messages"))

	req := httptest.NewRequest(http.MethodPatch, "/channels/0/keys/index/9/meta", bytes.NewBufferString(`{"disabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want %d", w.Code, http.StatusNotFound)
	}
}
