package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/BenedictKing/claude-proxy/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestProbeUpstreamModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"unauthorized"}}`))
			return
		}

		switch r.URL.Path {
		case "/v1/models", "/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt-4o","object":"model","created":0,"owned_by":"openai"}]}`))
		case "/forbidden/v1/models":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"message":"forbidden"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"message":"not found"}}`))
		}
	}))
	defer upstream.Close()

	envCfg := &config.EnvConfig{ProxyAccessKey: "secret-key", EnableWebUI: true}
	r := gin.New()
	r.Use(middleware.WebAuthMiddleware(envCfg, nil))
	r.POST("/api/admin/upstream/models", ProbeUpstreamModels())

	t.Run("missing key returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/admin/upstream/models", bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("invalid request returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/admin/upstream/models", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("success returns models", func(t *testing.T) {
		payload := map[string]any{"baseUrl": upstream.URL, "apiKey": "test-key"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/upstream/models", bytes.NewReader(body))
		req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp probeUpstreamModelsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !resp.Success {
			t.Fatalf("success = false, want true (resp=%s)", w.Body.String())
		}
		if resp.Models == nil || len(resp.Models.Data) != 1 || resp.Models.Data[0].ID != "gpt-4o" {
			t.Fatalf("models unexpected: %s", w.Body.String())
		}
	})

	t.Run("baseUrl ending with # skips /v1 prefix", func(t *testing.T) {
		payload := map[string]any{"baseUrl": upstream.URL + "#", "apiKey": "test-key"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/upstream/models", bytes.NewReader(body))
		req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp probeUpstreamModelsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !resp.Success || resp.Models == nil || len(resp.Models.Data) != 1 {
			t.Fatalf("response unexpected: %s", w.Body.String())
		}
	})

	t.Run("upstream error returns success=false with statusCode", func(t *testing.T) {
		payload := map[string]any{"baseUrl": upstream.URL + "/forbidden", "apiKey": "test-key"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/upstream/models", bytes.NewReader(body))
		req.Header.Set("x-api-key", envCfg.ProxyAccessKey)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp probeUpstreamModelsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Success {
			t.Fatalf("success = true, want false")
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("statusCode = %d, want %d", resp.StatusCode, http.StatusForbidden)
		}
		if strings.TrimSpace(resp.UpstreamError) == "" {
			t.Fatalf("upstreamError should not be empty")
		}
	})
}
