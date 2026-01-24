package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/BenedictKing/claude-proxy/internal/metrics"
	"github.com/BenedictKing/claude-proxy/internal/scheduler"
	"github.com/BenedictKing/claude-proxy/internal/session"
	"github.com/BenedictKing/claude-proxy/internal/warmup"
	"github.com/gin-gonic/gin"
)

type recordingRequestLogStats struct {
	resetKeyCalls []struct {
		apiType      string
		channelIndex int
		keyID        string
	}
	resetChannelCalls []struct {
		apiType      string
		channelIndex int
	}
}

func (r *recordingRequestLogStats) GetTotalRequestCount(string) int64 { return 0 }
func (r *recordingRequestLogStats) GetKeyRequestCount(string, int, string) int64 {
	return 0
}
func (r *recordingRequestLogStats) ResetKey(apiType string, channelIndex int, keyID string) {
	r.resetKeyCalls = append(r.resetKeyCalls, struct {
		apiType      string
		channelIndex int
		keyID        string
	}{apiType: apiType, channelIndex: channelIndex, keyID: keyID})
}
func (r *recordingRequestLogStats) ResetChannel(apiType string, channelIndex int) {
	r.resetChannelCalls = append(r.resetChannelCalls, struct {
		apiType      string
		channelIndex int
	}{apiType: apiType, channelIndex: channelIndex})
}

func createTestConfigManager(t *testing.T, cfg config.Config) (*config.ConfigManager, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfgManager, err := config.NewConfigManager(configFile)
	if err != nil {
		t.Fatalf("NewConfigManager: %v", err)
	}
	return cfgManager, func() { cfgManager.Close() }
}

func createTestScheduler(t *testing.T, cfgManager *config.ConfigManager) (*scheduler.ChannelScheduler, func()) {
	t.Helper()

	messagesMetrics := metrics.NewMetricsManager()
	responsesMetrics := metrics.NewMetricsManager()
	geminiMetrics := metrics.NewMetricsManager()
	traceAffinity := session.NewTraceAffinityManager()
	urlManager := warmup.NewURLManager(30*time.Second, 3)

	sch := scheduler.NewChannelScheduler(cfgManager, messagesMetrics, responsesMetrics, geminiMetrics, traceAffinity, urlManager)
	return sch, func() {
		messagesMetrics.Stop()
		responsesMetrics.Stop()
		geminiMetrics.Stop()
		traceAffinity.Stop()
	}
}

func TestResetAllKeysCircuitStatus_Messages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:    "ch0",
				BaseURL: "https://example.com",
				APIKeys: []string{"k1", "k2"},
				Status:  "active",
			},
		},
	}
	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	r := gin.New()
	r.POST("/messages/channels/:id/keys/reset-state", ResetAllKeysCircuitStatus(sch, cfgManager, "messages"))

	req := httptest.NewRequest(http.MethodPost, "/messages/channels/0/keys/reset-state", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Count   int  `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal resp: %v body=%s", err, w.Body.String())
	}
	if !resp.Success || resp.Count != 2 {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestResetAllKeysCircuitStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:    "ch0",
				BaseURL: "https://example.com",
				APIKeys: []string{"k1"},
				Status:  "active",
			},
		},
	}
	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	r := gin.New()
	r.POST("/messages/channels/:id/keys/reset-state", ResetAllKeysCircuitStatus(sch, cfgManager, "messages"))

	req := httptest.NewRequest(http.MethodPost, "/messages/channels/99/keys/reset-state", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestResetAllKeysCircuitState_Messages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:    "ch0",
				BaseURL: "https://example.com",
				APIKeys: []string{"k1", "k2"},
				Status:  "active",
			},
		},
	}
	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	r := gin.New()
	r.POST("/messages/channels/:id/keys/reset", ResetAllKeysCircuitState(sch, cfgManager, "messages", nil))

	req := httptest.NewRequest(http.MethodPost, "/messages/channels/0/keys/reset", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Count   int  `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal resp: %v body=%s", err, w.Body.String())
	}
	if !resp.Success || resp.Count != 2 {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestResetKeyCircuitState_InvokesRequestLogReset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:    "ch0",
				BaseURL: "https://example.com",
				APIKeys: []string{"k1", "k2"},
				Status:  "active",
			},
		},
	}
	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	stats := &recordingRequestLogStats{}

	r := gin.New()
	r.POST("/messages/channels/:id/keys/index/:keyIndex/reset", ResetKeyCircuitState(sch, cfgManager, "messages", stats))

	req := httptest.NewRequest(http.MethodPost, "/messages/channels/0/keys/index/1/reset", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	if len(stats.resetKeyCalls) != 1 {
		t.Fatalf("resetKeyCalls=%d, want %d", len(stats.resetKeyCalls), 1)
	}
	call := stats.resetKeyCalls[0]
	if call.apiType != "messages" || call.channelIndex != 0 || call.keyID != metrics.HashAPIKey("k2") {
		t.Fatalf("resetKeyCalls[0]=%+v", call)
	}
}

func TestResetAllKeysCircuitState_InvokesRequestLogReset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:    "ch0",
				BaseURL: "https://example.com",
				APIKeys: []string{"k1", "k2"},
				Status:  "active",
			},
		},
	}
	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	stats := &recordingRequestLogStats{}

	r := gin.New()
	r.POST("/messages/channels/:id/keys/reset", ResetAllKeysCircuitState(sch, cfgManager, "messages", stats))

	req := httptest.NewRequest(http.MethodPost, "/messages/channels/0/keys/reset", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	if len(stats.resetChannelCalls) != 1 {
		t.Fatalf("resetChannelCalls=%d, want %d", len(stats.resetChannelCalls), 1)
	}
	call := stats.resetChannelCalls[0]
	if call.apiType != "messages" || call.channelIndex != 0 {
		t.Fatalf("resetChannelCalls[0]=%+v", call)
	}
}
