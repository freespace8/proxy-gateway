package gemini

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/gin-gonic/gin"
)

func TestGetDashboard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		GeminiUpstream: []config.UpstreamConfig{
			{
				Name:        "g0",
				ServiceType: "gemini",
				BaseURL:     "https://g0.example.com",
				APIKeys:     []string{"gk0"},
				Status:      "active",
				Priority:    1,
			},
		},
		GeminiLoadBalance:    "failover",
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		FuzzyModeEnabled:     true,
	}

	cfgManager, cleanupCfg := createTestConfigManager(t, cfg)
	defer cleanupCfg()

	sch, cleanupSch := createTestScheduler(t, cfgManager)
	defer cleanupSch()

	// Seed metrics to cover non-empty response paths
	gm := sch.GetGeminiMetricsManager()
	gm.RecordSuccess("https://g0.example.com", "gk0")

	r := gin.New()
	r.GET("/dash", GetDashboard(cfgManager, sch))

	req := httptest.NewRequest(http.MethodGet, "/dash", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := payload["channels"].([]any); !ok {
		t.Fatalf("missing channels: %+v", payload)
	}
	if _, ok := payload["metrics"].([]any); !ok {
		t.Fatalf("missing metrics: %+v", payload)
	}
	if _, ok := payload["stats"].(map[string]any); !ok {
		t.Fatalf("missing stats: %+v", payload)
	}
	if recent, ok := payload["recentActivity"].([]any); !ok || len(recent) != 1 {
		t.Fatalf("recentActivity invalid: %+v", payload["recentActivity"])
	}
}
