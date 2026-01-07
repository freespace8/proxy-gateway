package responses

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/BenedictKing/claude-proxy/internal/session"
	"github.com/gin-gonic/gin"
)

func TestResponsesHandler_SingleChannel_NoUpstreamConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Upstream:             []config.UpstreamConfig{},
		ResponsesUpstream:    []config.UpstreamConfig{},
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		GeminiLoadBalance:    "failover",
		FuzzyModeEnabled:     true,
	}

	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	sessionManager := session.NewSessionManager(time.Hour, 10, 1000)
	envCfg := &config.EnvConfig{ProxyAccessKey: "secret", MaxRequestBodySize: 1024 * 1024}

	r := gin.New()
	r.POST("/v1/responses", NewHandler(envCfg, cfgManager, sessionManager, sch, nil, nil, nil, nil))

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4o","input":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NO_RESPONSES_UPSTREAM") {
		t.Fatalf("expected code NO_RESPONSES_UPSTREAM, got %s", w.Body.String())
	}
}

func TestResponsesHandler_SingleChannel_NoAPIKeysReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{},
		ResponsesUpstream: []config.UpstreamConfig{
			{Name: "r0", BaseURL: "http://example.invalid", APIKeys: nil, ServiceType: "responses", Status: "active", Priority: 1},
		},
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		GeminiLoadBalance:    "failover",
		FuzzyModeEnabled:     true,
	}

	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	sessionManager := session.NewSessionManager(time.Hour, 10, 1000)
	envCfg := &config.EnvConfig{ProxyAccessKey: "secret", MaxRequestBodySize: 1024 * 1024}

	r := gin.New()
	r.POST("/v1/responses", NewHandler(envCfg, cfgManager, sessionManager, sch, nil, nil, nil, nil))

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4o","input":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NO_API_KEYS") {
		t.Fatalf("expected code NO_API_KEYS, got %s", w.Body.String())
	}
}

func TestResponsesHandler_SingleChannel_FailoverKeyThenSuccessAndDeprioritize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var calls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		calls.Add(1)
		if strings.Contains(r.Header.Get("Authorization"), "rk-bad") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"quota exceeded"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "id":"resp_1",
  "model":"gpt-4o",
  "status":"completed",
  "output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],
  "usage":{"input_tokens":10,"output_tokens":10,"total_tokens":20}
}`))
	}))
	defer upstream.Close()

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{},
		ResponsesUpstream: []config.UpstreamConfig{
			{
				Name:        "r0",
				BaseURL:     upstream.URL,
				APIKeys:     []string{"rk-bad", "rk-good"},
				ServiceType: "responses",
				Status:      "active",
				Priority:    1,
			},
		},
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		GeminiLoadBalance:    "failover",
		FuzzyModeEnabled:     true,
	}

	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	sessionManager := session.NewSessionManager(time.Hour, 100, 100000)
	envCfg := &config.EnvConfig{ProxyAccessKey: "secret", MaxRequestBodySize: 1024 * 1024}

	r := gin.New()
	r.POST("/v1/responses", NewHandler(envCfg, cfgManager, sessionManager, sch, nil, nil, nil, nil))

	// 固定路由到 bad key，确保覆盖“key 失败->切到另一个 key”路径（避免 sessionID 随机导致用例不稳定）。
	convID := ""
	for i := 0; i < 1024; i++ {
		candidate := fmt.Sprintf("cid-%d", i)
		selection, err := sch.SelectSlot(context.Background(), candidate, map[string]bool{}, true)
		if err != nil {
			t.Fatalf("SelectSlot err: %v", err)
		}
		if selection.APIKey == "rk-bad" {
			convID = candidate
			break
		}
	}
	if convID == "" {
		t.Fatalf("failed to find Conversation_id that routes to rk-bad")
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4o","input":"hello","store":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
	req.Header.Set("Conversation_id", convID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d, want 2", calls.Load())
	}
	if !strings.Contains(w.Body.String(), `"id":"resp_1"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestResponsesHandler_SingleChannel_NonFailover400ReturnsUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	defer upstream.Close()

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{},
		ResponsesUpstream: []config.UpstreamConfig{
			{Name: "r0", BaseURL: upstream.URL, APIKeys: []string{"rk1"}, ServiceType: "responses", Status: "active", Priority: 1},
		},
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		GeminiLoadBalance:    "failover",
		FuzzyModeEnabled:     false,
	}

	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	sessionManager := session.NewSessionManager(time.Hour, 10, 1000)
	envCfg := &config.EnvConfig{ProxyAccessKey: "secret", MaxRequestBodySize: 1024 * 1024}

	r := gin.New()
	r.POST("/v1/responses", NewHandler(envCfg, cfgManager, sessionManager, sch, nil, nil, nil, nil))

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4o","input":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "bad request") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestResponsesHandler_SingleChannel_InvalidJSONCausesConvertError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{},
		ResponsesUpstream: []config.UpstreamConfig{
			{Name: "r0", BaseURL: "http://127.0.0.1:0", APIKeys: []string{"rk1"}, ServiceType: "responses", Status: "active", Priority: 1},
		},
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		GeminiLoadBalance:    "failover",
		FuzzyModeEnabled:     true,
	}

	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	sessionManager := session.NewSessionManager(time.Hour, 10, 1000)
	envCfg := &config.EnvConfig{ProxyAccessKey: "secret", MaxRequestBodySize: 1024 * 1024}

	r := gin.New()
	r.POST("/v1/responses", NewHandler(envCfg, cfgManager, sessionManager, sch, nil, nil, nil, nil))

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
