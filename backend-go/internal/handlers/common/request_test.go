package common

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/BenedictKing/claude-proxy/internal/metrics"
	"github.com/gin-gonic/gin"
)

func TestReadRequestBody_SuccessRestoresBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString("hello"))

	body, err := ReadRequestBody(c, 10)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(body) != "hello" {
		t.Fatalf("body = %q", string(body))
	}

	again, _ := io.ReadAll(c.Request.Body)
	if string(again) != "hello" {
		t.Fatalf("restored body = %q", string(again))
	}
}

func TestReadRequestBody_TooLargeReturns413(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString("123456"))

	_, err := ReadRequestBody(c, 5)
	if err == nil {
		t.Fatalf("expected error")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestRestoreRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString("x"))

	RestoreRequestBody(c, []byte("abc"))
	got, _ := io.ReadAll(c.Request.Body)
	if string(got) != "abc" {
		t.Fatalf("got %q", string(got))
	}
}

func TestSendRequest_StandardAndStream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	upstream := &config.UpstreamConfig{InsecureSkipVerify: true}

	t.Run("standard client", func(t *testing.T) {
		envCfg := &config.EnvConfig{
			Env:               "development",
			RequestTimeout:    1000,
			EnableRequestLogs: true,
			RawLogOutput:      false,
		}

		req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewBufferString(`{"a":1}`))
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := SendRequest(req, upstream, envCfg, false)
		if err != nil {
			t.Fatalf("SendRequest: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("stream client", func(t *testing.T) {
		envCfg := &config.EnvConfig{
			Env:               "development",
			RequestTimeout:    1000,
			EnableRequestLogs: true,
			RawLogOutput:      true,
		}

		req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewBufferString(`{"a":1}`))
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := SendRequest(req, upstream, envCfg, true)
		if err != nil {
			t.Fatalf("SendRequest: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})
}

func TestLogOriginalRequest_CoversBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"a":1}`))
	c.Request.Header.Set("Authorization", "Bearer secret")

	LogOriginalRequest(c, []byte(`{"a":1}`), &config.EnvConfig{EnableRequestLogs: false}, "Messages")
	LogOriginalRequest(c, []byte(`{"a":1}`), &config.EnvConfig{EnableRequestLogs: true, Env: "development"}, "Messages")
	LogOriginalRequest(c, []byte(`{"a":1}`), &config.EnvConfig{EnableRequestLogs: true, Env: "development", RawLogOutput: true}, "Messages")
}

func TestAreAllKeysSuspended(t *testing.T) {
	m := metrics.NewMetricsManagerWithConfig(3, 0.5)
	defer m.Stop()

	baseURL := "https://example.com"
	keys := []string{"k1", "k2"}

	for i := 0; i < 3; i++ {
		m.RecordFailure(baseURL, "k1")
		m.RecordFailure(baseURL, "k2")
	}

	if !AreAllKeysSuspended(m, baseURL, keys) {
		t.Fatalf("expected all keys suspended")
	}
	if AreAllKeysSuspended(m, baseURL, nil) {
		t.Fatalf("expected false for empty keys")
	}

	m2 := metrics.NewMetricsManagerWithConfig(3, 0.5)
	defer m2.Stop()
	for i := 0; i < 3; i++ {
		m2.RecordFailure(baseURL, "k1")
	}
	if AreAllKeysSuspended(m2, baseURL, keys) {
		t.Fatalf("expected false when one key is not suspended")
	}
}

func TestAreAllKeysSoftSuspended_IgnoresHardSuspend(t *testing.T) {
	m := metrics.NewMetricsManagerWithConfig(3, 0.5)
	defer m.Stop()

	baseURL := "https://example.com"
	keys := []string{"k1", "k2"}

	// k1 进入硬熔断，k2 未进入任何熔断
	m.SuspendKeyUntil(baseURL, "k1", time.Now().Add(1*time.Hour), "insufficient_balance")

	if AreAllKeysSuspended(m, baseURL, keys) {
		t.Fatalf("expected false: only one key hard-suspended")
	}
	if AreAllKeysSoftSuspended(m, baseURL, keys) {
		t.Fatalf("expected false: hard suspend should not count as soft suspend")
	}

	// 模拟两把 key 都进入软熔断
	for i := 0; i < 3; i++ {
		m.RecordFailure(baseURL, "k1")
		m.RecordFailure(baseURL, "k2")
	}
	if !AreAllKeysSoftSuspended(m, baseURL, keys) {
		t.Fatalf("expected all keys soft suspended")
	}
}

func TestExtractUserIDAndConversationID(t *testing.T) {
	body := []byte(`{"metadata":{"user_id":"u1"},"prompt_cache_key":"pc"}`)
	if got := ExtractUserID(body); got != "u1" {
		t.Fatalf("user_id = %q", got)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Request.Header.Set("Conversation_id", "c1")
	if got := ExtractConversationID(c, body); got != "c1" {
		t.Fatalf("conversation_id = %q", got)
	}

	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c2.Request.Header.Set("Session_id", "s1")
	if got := ExtractConversationID(c2, body); got != "s1" {
		t.Fatalf("session_id = %q", got)
	}

	c3, _ := gin.CreateTestContext(httptest.NewRecorder())
	c3.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c3.Request.Header.Set("X-Gemini-Api-Privileged-User-Id", "g1")
	if got := ExtractConversationID(c3, body); got != "g1" {
		t.Fatalf("gemini privileged user id = %q", got)
	}

	c4, _ := gin.CreateTestContext(httptest.NewRecorder())
	c4.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	if got := ExtractConversationID(c4, body); got != "pc" {
		t.Fatalf("prompt_cache_key = %q", got)
	}

	body2 := []byte(`{"metadata":{"user_id":"u2"}}`)
	c5, _ := gin.CreateTestContext(httptest.NewRecorder())
	c5.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	if got := ExtractConversationID(c5, body2); got != "u2" {
		t.Fatalf("metadata.user_id = %q", got)
	}
}

func TestRemoveEmptySignatures(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantModified bool
		wantRemoved  bool
	}{
		{
			name:         "no signature",
			body:         `{"messages":[{"content":[{"type":"text","text":"hi"}]}]}`,
			wantModified: false,
			wantRemoved:  false,
		},
		{
			name:         "empty string signature removed",
			body:         `{"messages":[{"content":[{"type":"tool_use","signature":"","text":"x"}]}]}`,
			wantModified: true,
			wantRemoved:  true,
		},
		{
			name:         "null signature removed",
			body:         `{"messages":[{"content":[{"type":"tool_use","signature":null,"text":"x"}]}]}`,
			wantModified: true,
			wantRemoved:  true,
		},
		{
			name:         "non-empty signature kept",
			body:         `{"messages":[{"content":[{"type":"tool_use","signature":"abc","text":"x"}]}]}`,
			wantModified: false,
			wantRemoved:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, modified := RemoveEmptySignatures([]byte(tt.body), false)
			if modified != tt.wantModified {
				t.Fatalf("modified=%v, want %v, out=%s", modified, tt.wantModified, string(out))
			}

			if tt.wantRemoved {
				if bytes.Contains(out, []byte(`"signature"`)) {
					t.Fatalf("signature should be removed, out=%s", string(out))
				}
				return
			}

			if !bytes.Equal(out, []byte(tt.body)) {
				t.Fatalf("body should be unchanged, out=%s", string(out))
			}
		})
	}
}

func TestCaptureUpstreamRequestSnapshot_FullURLHeadersAndBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/v1/messages?trace=1", bytes.NewBufferString(`{"a":1}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer sk-test-123")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Anthropic-Beta", "prompt-caching-2024-07-31")
	req.Header.Add("Anthropic-Beta", "extended-thinking-2025-01-01")

	snapshot := CaptureUpstreamRequestSnapshot(req)
	if snapshot.Method != http.MethodPost {
		t.Fatalf("method=%q", snapshot.Method)
	}
	if snapshot.URL != "https://api.example.com/v1/messages?trace=1" {
		t.Fatalf("url=%q", snapshot.URL)
	}
	if got := snapshot.Headers["Authorization"]; got != "Bearer sk-test-123" {
		t.Fatalf("authorization=%q", got)
	}
	if got := snapshot.Headers["Anthropic-Beta"]; got != "prompt-caching-2024-07-31, extended-thinking-2025-01-01" {
		t.Fatalf("anthropic-beta=%q", got)
	}
	if snapshot.Body != `{"a":1}` {
		t.Fatalf("body=%q", snapshot.Body)
	}

	restored, _ := io.ReadAll(req.Body)
	if string(restored) != `{"a":1}` {
		t.Fatalf("restored body=%q", string(restored))
	}
}

func TestCaptureRequestSnapshot_KeepsSensitiveHeadersForDebug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"x":1}`))
	c.Request.Header.Set("Authorization", "Bearer client-token")
	c.Request.Header.Set("x-api-key", "proxy-token")

	snapshot := CaptureRequestSnapshot(c, []byte(`{"x":1}`))
	if snapshot.Headers["Authorization"] != "Bearer client-token" {
		t.Fatalf("authorization=%q", snapshot.Headers["Authorization"])
	}
	if snapshot.Headers["x-api-key"] != "proxy-token" {
		t.Fatalf("x-api-key=%q", snapshot.Headers["x-api-key"])
	}
}

func TestCaptureUpstreamRequestSnapshot_BodyLimitRaisedFrom64KB(t *testing.T) {
	body := strings.Repeat("x", 128*1024)
	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/v1/messages", strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	snapshot := CaptureUpstreamRequestSnapshot(req)
	if snapshot.BodyTruncated {
		t.Fatalf("body should not be truncated at 128KB")
	}
	if len(snapshot.Body) != len(body) {
		t.Fatalf("body len=%d want=%d", len(snapshot.Body), len(body))
	}
}

func TestResolveRequestSnapshot_PrefersUpstreamSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"x":1}`))
	fallback := CaptureRequestSnapshot(c, []byte(`{"x":1}`))

	upstreamReq, err := http.NewRequest(http.MethodPost, "https://api.example.com/v1/messages", bytes.NewBufferString(`{"y":2}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	upstreamReq.Header.Set("x-api-key", "sk-real-456")
	SetUpstreamRequestSnapshot(c, upstreamReq)

	got := ResolveRequestSnapshot(c, fallback)
	if got.URL != "https://api.example.com/v1/messages" {
		t.Fatalf("url=%q", got.URL)
	}
	if got.Body != `{"y":2}` {
		t.Fatalf("body=%q", got.Body)
	}
	if got.Headers["x-api-key"] != "sk-real-456" {
		t.Fatalf("x-api-key=%q", got.Headers["x-api-key"])
	}
}
