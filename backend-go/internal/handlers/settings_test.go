package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/gin-gonic/gin"
)

func TestFuzzyModeHandlers_GetAndSet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cm, _ := newTestConfigManager(t, config.Config{
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		GeminiLoadBalance:    "failover",
		FuzzyModeEnabled:     false,
	})

	r := gin.New()
	r.GET("/api/settings/fuzzy-mode", GetFuzzyMode(cm))
	r.PUT("/api/settings/fuzzy-mode", SetFuzzyMode(cm))

	// GET initial
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/settings/fuzzy-mode", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET status=%d", w.Code)
		}
		var resp struct {
			Enabled bool `json:"fuzzyModeEnabled"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Enabled {
			t.Fatalf("expected fuzzyModeEnabled=false")
		}
	}

	// PUT invalid body
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/settings/fuzzy-mode", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("PUT invalid status=%d", w.Code)
		}
	}

	// PUT enable
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/settings/fuzzy-mode", bytes.NewBufferString(`{"enabled":true}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("PUT status=%d body=%s", w.Code, w.Body.String())
		}
	}

	// GET updated
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/settings/fuzzy-mode", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET2 status=%d", w.Code)
		}
		var resp struct {
			Enabled bool `json:"fuzzyModeEnabled"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !resp.Enabled {
			t.Fatalf("expected fuzzyModeEnabled=true")
		}
	}
}

func TestGlobalMappingHandlers_GetAndSet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cm, _ := newTestConfigManager(t, config.Config{
		LoadBalance:            "failover",
		ResponsesLoadBalance:   "failover",
		GeminiLoadBalance:      "failover",
		GlobalModelMapping:     map[string]string{"codex": "gpt-5.1-codex"},
		GlobalReasoningMapping: map[string]string{"low": "high"},
	})

	r := gin.New()
	r.GET("/api/settings/model-mapping", GetGlobalModelMapping(cm))
	r.PUT("/api/settings/model-mapping", SetGlobalModelMapping(cm))
	r.GET("/api/settings/reasoning-mapping", GetGlobalReasoningMapping(cm))
	r.PUT("/api/settings/reasoning-mapping", SetGlobalReasoningMapping(cm))

	// GET model mapping
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/settings/model-mapping", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET model mapping status=%d", w.Code)
		}
		var resp struct {
			Mapping map[string]string `json:"globalModelMapping"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal model mapping: %v", err)
		}
		expected := map[string]string{"codex": "gpt-5.1-codex"}
		if !reflect.DeepEqual(resp.Mapping, expected) {
			t.Fatalf("model mapping=%v want=%v", resp.Mapping, expected)
		}
	}

	// PUT model mapping
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/settings/model-mapping", bytes.NewBufferString(`{"globalModelMapping":{" codex ":" gpt-5.2 ","":"ignored","noop":" "}}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("PUT model mapping status=%d body=%s", w.Code, w.Body.String())
		}
	}

	// GET model mapping after normalization
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/settings/model-mapping", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET model mapping2 status=%d", w.Code)
		}
		var resp struct {
			Mapping map[string]string `json:"globalModelMapping"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal model mapping2: %v", err)
		}
		expected := map[string]string{"codex": "gpt-5.2"}
		if !reflect.DeepEqual(resp.Mapping, expected) {
			t.Fatalf("model mapping2=%v want=%v", resp.Mapping, expected)
		}
	}

	// PUT reasoning mapping
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/settings/reasoning-mapping", bytes.NewBufferString(`{"globalReasoningMapping":{" LOW ":" XHIGH ","":"high","medium":" "}}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("PUT reasoning mapping status=%d body=%s", w.Code, w.Body.String())
		}
	}

	// GET reasoning mapping after normalization
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/settings/reasoning-mapping", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET reasoning mapping status=%d", w.Code)
		}
		var resp struct {
			Mapping map[string]string `json:"globalReasoningMapping"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal reasoning mapping: %v", err)
		}
		expected := map[string]string{"low": "xhigh"}
		if !reflect.DeepEqual(resp.Mapping, expected) {
			t.Fatalf("reasoning mapping=%v want=%v", resp.Mapping, expected)
		}
	}
}

func TestSetFuzzyMode_SaveFailReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cm, path := newTestConfigManager(t, config.Config{
		LoadBalance:          "failover",
		ResponsesLoadBalance: "failover",
		GeminiLoadBalance:    "failover",
		FuzzyModeEnabled:     false,
	})

	if err := os.Chmod(path, 0400); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	r := gin.New()
	r.PUT("/api/settings/fuzzy-mode", SetFuzzyMode(cm))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/settings/fuzzy-mode", bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
