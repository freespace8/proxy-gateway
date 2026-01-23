package responses

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/httpclient"
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

	t.Run("right_codes_account_summary_success_should_return_fields", func(t *testing.T) {
		summaryCount := 0
		responsesCount := 0

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer rk-test" {
				t.Fatalf("authorization=%q", got)
			}
			switch r.URL.Path {
			case "/account/summary":
				summaryCount++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"balance":12.34,"subscription":{"used_quota":1,"total_quota":10,"remaining_quota":9},"is_active":true}`))
				return
			case "/codex/v1/responses":
				responsesCount++
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"你\"}\n\n"))
				return
			default:
				t.Fatalf("path=%s", r.URL.Path)
			}
		}))
		defer srv.Close()

		client := httpclient.GetManager().GetStandardClient(10*time.Second, false)
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("transport_not_http_transport")
		}
		origDial := transport.DialContext
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err == nil && strings.HasSuffix(strings.ToLower(host), "right.codes") {
				return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
			}
			if origDial != nil {
				return origDial(ctx, network, addr)
			}
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		}
		defer func() { transport.DialContext = origDial }()

		mm := metrics.NewMetricsManagerWithConfig(10, 0.5)
		r := gin.New()
		r.POST("/validate", ValidateCodexRightKey(mm))

		baseURL := "http://www.right.codes/codex"
		reqBody := []byte(`{"apiKey":"rk-test","baseUrl":"` + baseURL + `"}`)
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
		if !bytes.Contains(w.Body.Bytes(), []byte(`"rightCodes"`)) {
			t.Fatalf("resp=%s", w.Body.String())
		}
		if summaryCount != 1 {
			t.Fatalf("summaryCount=%d", summaryCount)
		}
		if responsesCount != 1 {
			t.Fatalf("responsesCount=%d", responsesCount)
		}
	})

	t.Run("right_codes_account_summary_summary_only_should_skip_probe", func(t *testing.T) {
		responsesCalled := false
		summaryCount := 0

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer rk-test" {
				t.Fatalf("authorization=%q", got)
			}
			switch r.URL.Path {
			case "/account/summary":
				summaryCount++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"balance":21.054373,"subscription":{"used_quota":72.966156,"total_quota":90,"remaining_quota":17.033844},"is_active":true}`))
				return
			case "/codex/v1/responses":
				responsesCalled = true
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("data: {}\n\n"))
				return
			default:
				t.Fatalf("path=%s", r.URL.Path)
			}
		}))
		defer srv.Close()

		client := httpclient.GetManager().GetStandardClient(10*time.Second, false)
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("transport_not_http_transport")
		}
		origDial := transport.DialContext
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err == nil && strings.HasSuffix(strings.ToLower(host), "right.codes") {
				return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
			}
			if origDial != nil {
				return origDial(ctx, network, addr)
			}
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		}
		defer func() { transport.DialContext = origDial }()

		mm := metrics.NewMetricsManagerWithConfig(10, 0.5)
		r := gin.New()
		r.POST("/validate", ValidateCodexRightKey(mm))

		baseURL := "http://www.right.codes/codex"
		reqBody := []byte(`{"apiKey":"rk-test","baseUrl":"` + baseURL + `","summaryOnly":true}`)
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
		if !bytes.Contains(w.Body.Bytes(), []byte(`"rightCodes"`)) {
			t.Fatalf("resp=%s", w.Body.String())
		}
		if responsesCalled {
			t.Fatalf("should_not_call_responses")
		}
		if summaryCount != 1 {
			t.Fatalf("summaryCount=%d", summaryCount)
		}
	})

	t.Run("right_codes_account_summary_remaining_zero_should_suspend_and_skip_probe", func(t *testing.T) {
		responsesCalled := false

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer rk-test" {
				t.Fatalf("authorization=%q", got)
			}
			switch r.URL.Path {
			case "/account/summary":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"balance":0,"subscription":{"remaining_quota":0},"is_active":true}`))
				return
			case "/codex/v1/responses":
				responsesCalled = true
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("data: {}\n\n"))
				return
			default:
				t.Fatalf("path=%s", r.URL.Path)
			}
		}))
		defer srv.Close()

		client := httpclient.GetManager().GetStandardClient(10*time.Second, false)
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("transport_not_http_transport")
		}
		origDial := transport.DialContext
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err == nil && strings.HasSuffix(strings.ToLower(host), "right.codes") {
				return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
			}
			if origDial != nil {
				return origDial(ctx, network, addr)
			}
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		}
		defer func() { transport.DialContext = origDial }()

		mm := metrics.NewMetricsManagerWithConfig(10, 0.5)
		r := gin.New()
		r.POST("/validate", ValidateCodexRightKey(mm))

		baseURL := "http://www.right.codes/codex"
		reqBody := []byte(`{"apiKey":"rk-test","baseUrl":"` + baseURL + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"success":false`)) {
			t.Fatalf("resp=%s", w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"statusCode":402`)) {
			t.Fatalf("resp=%s", w.Body.String())
		}
		if responsesCalled {
			t.Fatalf("should_not_call_responses")
		}
		if !mm.IsKeyHardSuspended(baseURL, "rk-test") {
			t.Fatalf("should_suspend")
		}
	})

	t.Run("right_codes_account_summary_inactive_should_fail", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer rk-test" {
				t.Fatalf("authorization=%q", got)
			}
			switch r.URL.Path {
			case "/account/summary":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"balance":1,"subscription":{"remaining_quota":1},"is_active":false}`))
				return
			case "/codex/v1/responses":
				t.Fatalf("should_not_call_responses")
			default:
				t.Fatalf("path=%s", r.URL.Path)
			}
		}))
		defer srv.Close()

		client := httpclient.GetManager().GetStandardClient(10*time.Second, false)
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("transport_not_http_transport")
		}
		origDial := transport.DialContext
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err == nil && strings.HasSuffix(strings.ToLower(host), "right.codes") {
				return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
			}
			if origDial != nil {
				return origDial(ctx, network, addr)
			}
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		}
		defer func() { transport.DialContext = origDial }()

		mm := metrics.NewMetricsManagerWithConfig(10, 0.5)
		r := gin.New()
		r.POST("/validate", ValidateCodexRightKey(mm))

		baseURL := "http://www.right.codes/codex"
		reqBody := []byte(`{"apiKey":"rk-test","baseUrl":"` + baseURL + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(reqBody))
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

		if mm.IsKeyHardSuspended(baseURL, "rk-test") {
			t.Fatalf("should_not_suspend")
		}
	})
}
