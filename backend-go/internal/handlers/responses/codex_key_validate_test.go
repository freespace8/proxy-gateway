package responses

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BenedictKing/claude-proxy/internal/metrics"
	"github.com/gin-gonic/gin"
)

func TestValidateCodexRightKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success_2xx_stream_like_body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method=%s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("path=%s", r.URL.Path)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer rk-test" {
				t.Fatalf("authorization=%q", got)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"你\"}\n\n"))
		}))
		defer srv.Close()

		mm := metrics.NewMetricsManagerWithConfig(10, 0.5)
		r := gin.New()
		r.POST("/validate", ValidateCodexRightKey(mm))

		reqBody := []byte(`{"apiKey":"rk-test","baseUrl":"` + srv.URL + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"success":true`)) {
			t.Fatalf("resp=%s", w.Body.String())
		}
	})

	t.Run("fail_non_2xx", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("path=%s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"denied"}`))
		}))
		defer srv.Close()

		mm := metrics.NewMetricsManagerWithConfig(10, 0.5)
		r := gin.New()
		r.POST("/validate", ValidateCodexRightKey(mm))

		req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader([]byte(`{"apiKey":"rk-test","baseUrl":"`+srv.URL+`"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"success":false`)) {
			t.Fatalf("resp=%s", w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"statusCode":403`)) {
			t.Fatalf("resp=%s", w.Body.String())
		}

		if mm.IsKeyHardSuspended(srv.URL, "rk-test") {
			t.Fatalf("should_not_suspend")
		}
	})

	t.Run("fail_2xx_but_wrapped_error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("path=%s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"timestamp":"2026-01-20T15:18:52.900945658+08:00","baseUrl":"https://www.right.codes/codex","statusCode":403,"errorMessage":"上游错误: 403","responseBody":"{\"error\":\"API Key额度不足\"}"}`))
		}))
		defer srv.Close()

		mm := metrics.NewMetricsManagerWithConfig(10, 0.5)
		r := gin.New()
		r.POST("/validate", ValidateCodexRightKey(mm))

		req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader([]byte(`{"apiKey":"rk-test","baseUrl":"`+srv.URL+`"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"success":false`)) {
			t.Fatalf("resp=%s", w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"statusCode":403`)) {
			t.Fatalf("resp=%s", w.Body.String())
		}

		if !mm.IsKeyHardSuspended(srv.URL, "rk-test") {
			t.Fatalf("should_suspend")
		}
	})

	t.Run("fail_non_2xx_insufficient_balance_should_suspend", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("path=%s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"message":"API Key额度不足"}}`))
		}))
		defer srv.Close()

		mm := metrics.NewMetricsManagerWithConfig(10, 0.5)
		r := gin.New()
		r.POST("/validate", ValidateCodexRightKey(mm))

		req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader([]byte(`{"apiKey":"rk-test","baseUrl":"`+srv.URL+`"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"success":false`)) {
			t.Fatalf("resp=%s", w.Body.String())
		}
		if !mm.IsKeyHardSuspended(srv.URL, "rk-test") {
			t.Fatalf("should_suspend")
		}
	})
}
