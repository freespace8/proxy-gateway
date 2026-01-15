package responses

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/BenedictKing/claude-proxy/internal/metrics"
	"github.com/BenedictKing/claude-proxy/internal/monitor"
	"github.com/BenedictKing/claude-proxy/internal/session"
	"github.com/gin-gonic/gin"
)

func TestResponsesHandler_WritesReasoningEffortToRequestLog_UpstreamWins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var upstreamBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		upstreamBody, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "id":"resp_1",
  "model":"gpt-5.2",
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
				APIKeys:     []string{"rk1"},
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

	dbPath := filepath.Join(t.TempDir(), "metrics.db")
	sqliteStore, err := metrics.NewSQLiteStore(&metrics.SQLiteStoreConfig{
		DBPath:        dbPath,
		RetentionDays: 3,
	})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer sqliteStore.Close()

	requestLogs := metrics.NewMemoryRequestLogStore(10)
	live := monitor.NewLiveRequestManager(10)

	envCfg := &config.EnvConfig{
		ProxyAccessKey:     "secret",
		MaxRequestBodySize: 1024 * 1024,
	}
	h := NewHandler(envCfg, cfgManager, sessionManager, sch, nil, nil, live, sqliteStore, requestLogs)

	r := gin.New()
	r.POST("/v1/responses", h)

	reqBody := `{"model":"gpt-5.2","input":"hi","reasoning":{"effort":"minimal"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	logs, total, err := requestLogs.QueryRequestLogs("responses", 10, 0)
	if err != nil {
		t.Fatalf("QueryRequestLogs: %v", err)
	}
	if total != 1 || len(logs) != 1 {
		t.Fatalf("logs total=%d len=%d, want 1", total, len(logs))
	}
	if logs[0].ReasoningEffort != "low" {
		t.Fatalf("ReasoningEffort=%q, want %q", logs[0].ReasoningEffort, "low")
	}
	if !bytes.Contains(upstreamBody, []byte(`"effort":"low"`)) {
		t.Fatalf("upstream body missing normalized effort=low: %s", string(upstreamBody))
	}
}
