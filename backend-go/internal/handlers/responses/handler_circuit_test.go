package responses

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/BenedictKing/claude-proxy/internal/metrics"
	"github.com/BenedictKing/claude-proxy/internal/scheduler"
	"github.com/BenedictKing/claude-proxy/internal/session"
	"github.com/BenedictKing/claude-proxy/internal/warmup"
	"github.com/gin-gonic/gin"
)

func createTestSchedulerWithMetricsConfig(t *testing.T, cfgManager *config.ConfigManager) (*scheduler.ChannelScheduler, func()) {
	t.Helper()

	messagesMetrics := metrics.NewMetricsManagerWithConfig(3, 0.5)
	responsesMetrics := metrics.NewMetricsManagerWithConfig(3, 0.5)
	geminiMetrics := metrics.NewMetricsManagerWithConfig(3, 0.5)
	traceAffinity := session.NewTraceAffinityManagerWithTTL(2 * time.Minute)
	urlManager := warmup.NewURLManager(30*time.Second, 3)

	sch := scheduler.NewChannelScheduler(cfgManager, messagesMetrics, responsesMetrics, geminiMetrics, traceAffinity, urlManager)
	cleanup := func() {
		messagesMetrics.Stop()
		responsesMetrics.Stop()
		geminiMetrics.Stop()
		traceAffinity.Stop()
	}
	return sch, cleanup
}

func TestResponsesHandler_SingleChannel_SkipsSuspendedKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var calls atomic.Int64
	var usedSuspended atomic.Bool
	var usedGood atomic.Bool

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		calls.Add(1)
		auth := r.Header.Get("Authorization")
		if strings.Contains(auth, "rk-suspended") {
			usedSuspended.Store(true)
		}
		if strings.Contains(auth, "rk-good") {
			usedGood.Store(true)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "id":"resp_ok",
  "model":"gpt-4o",
  "status":"completed",
  "output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],
  "usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}
}`))
	}))
	defer upstream.Close()

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{},
		ResponsesUpstream: []config.UpstreamConfig{
			{
				Name:        "r0",
				BaseURL:     upstream.URL,
				APIKeys:     []string{"rk-suspended", "rk-good"},
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

	sch, cleanupSch := createTestSchedulerWithMetricsConfig(t, cfgManager)
	defer cleanupSch()

	// 让第一个 key 进入熔断状态，触发 ShouldSuspendKey 跳过逻辑。
	rm := sch.GetResponsesMetricsManager()
	for i := 0; i < 3; i++ {
		rm.RecordFailure(upstream.URL, "rk-suspended")
	}

	sessionManager := session.NewSessionManager(time.Hour, 10, 1000)
	envCfg := &config.EnvConfig{
		ProxyAccessKey:     "secret",
		MaxRequestBodySize: 1024 * 1024,
	}

	r := gin.New()
	r.POST("/v1/responses", NewHandler(envCfg, cfgManager, sessionManager, sch, nil, nil, nil, nil, nil))

	reqBody := `{"model":"gpt-4o","input":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d, want 1", calls.Load())
	}
	if usedSuspended.Load() {
		t.Fatalf("expected suspended key skipped")
	}
	if !usedGood.Load() {
		t.Fatalf("expected good key used")
	}
}

func TestResponsesHandler_MultiChannel_BaseURLFailoverWithinChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var badCalls atomic.Int64
	upstreamBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		badCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"quota exceeded"}}`))
	}))
	defer upstreamBad.Close()

	var goodCalls atomic.Int64
	upstreamGood := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		goodCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "id":"resp_ok",
  "model":"gpt-4o",
  "status":"completed",
  "output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],
  "usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}
}`))
	}))
	defer upstreamGood.Close()

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{},
		ResponsesUpstream: []config.UpstreamConfig{
			{
				Name:        "r0",
				BaseURL:     upstreamBad.URL,
				BaseURLs:    []string{upstreamBad.URL, upstreamGood.URL},
				APIKeys:     []string{"rk1"},
				ServiceType: "responses",
				Status:      "active",
				Priority:    1,
			},
			{
				Name:        "unused",
				BaseURL:     upstreamGood.URL,
				APIKeys:     []string{"rk2"},
				ServiceType: "responses",
				Status:      "active",
				Priority:    2,
			},
		},
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		GeminiLoadBalance:    "failover",
		FuzzyModeEnabled:     true,
	}

	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestSchedulerWithMetricsConfig(t, cfgManager)
	defer cleanupSch()

	sessionManager := session.NewSessionManager(time.Hour, 10, 1000)
	envCfg := &config.EnvConfig{
		ProxyAccessKey:     "secret",
		MaxRequestBodySize: 1024 * 1024,
		Env:                "development",
		EnableRequestLogs:  true,
		EnableResponseLogs: true,
		RawLogOutput:       true,
	}

	r := gin.New()
	r.POST("/v1/responses", NewHandler(envCfg, cfgManager, sessionManager, sch, nil, nil, nil, nil, nil))

	// 固定路由 key，确保选择到 r0 槽位并覆盖“同渠道多 BaseURL failover”路径
	routingKey := "conv_baseurl_failover"
	sch.SetTraceAffinitySlot(routingKey, 0, 0)

	reqBody := `{"model":"gpt-4o","input":"hello","store":false}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
	req.Header.Set("Conversation_id", routingKey)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if badCalls.Load() != 1 {
		t.Fatalf("badCalls=%d, want 1", badCalls.Load())
	}
	if goodCalls.Load() != 1 {
		t.Fatalf("goodCalls=%d, want 1", goodCalls.Load())
	}
	if !strings.Contains(w.Body.String(), `"id":"resp_ok"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestResponsesHandler_ClientCanceled_DoesNotRecordFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var calls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{},
		ResponsesUpstream: []config.UpstreamConfig{
			{
				Name:        "r0",
				BaseURL:     upstream.URL,
				APIKeys:     []string{"rk1"},
				ServiceType: "responses",
				Status:      "active",
				Priority:    1,
			},
		},
		ResponsesLoadBalance: "failover",
		FuzzyModeEnabled:     true,
	}

	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestSchedulerWithMetricsConfig(t, cfgManager)
	defer cleanupSch()

	sessionManager := session.NewSessionManager(time.Hour, 10, 1000)
	envCfg := &config.EnvConfig{
		ProxyAccessKey:     "secret",
		MaxRequestBodySize: 1024 * 1024,
	}

	r := gin.New()
	r.POST("/v1/responses", NewHandler(envCfg, cfgManager, sessionManager, sch, nil, nil, nil, nil, nil))

	reqBody := `{"model":"gpt-4o","input":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", envCfg.ProxyAccessKey)

	ctx, cancel := context.WithCancel(req.Context())
	cancel() // 立即取消，模拟请求方断开
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if calls.Load() != 0 {
		t.Fatalf("upstream calls=%d, want 0 (canceled before request)", calls.Load())
	}

	rm := sch.GetResponsesMetricsManager()
	if km := rm.GetKeyMetrics(upstream.URL, "rk1"); km != nil && km.RequestCount != 0 {
		t.Fatalf("metrics requestCount=%d, want 0", km.RequestCount)
	}
}
