package metrics

import (
	"testing"
	"time"
)

func TestSuspendKeyUntil_HardSuspendAndAutoRecover(t *testing.T) {
	m := NewMetricsManagerWithConfig(10, 0.5)
	defer m.Stop()

	baseURL := "https://example.com"
	apiKey := "k1"

	m.SuspendKeyUntil(baseURL, apiKey, time.Now().Add(1*time.Hour), "insufficient_balance")
	if !m.IsKeyHardSuspended(baseURL, apiKey) {
		t.Fatalf("expected hard suspended")
	}
	if !m.ShouldSuspendKey(baseURL, apiKey) {
		t.Fatalf("expected ShouldSuspendKey=true")
	}
	if m.ShouldSuspendKeySoft(baseURL, apiKey) {
		t.Fatalf("expected ShouldSuspendKeySoft=false without failures")
	}

	// 到期后恢复：通过直接调用内部恢复逻辑验证
	m.SuspendKeyUntil(baseURL, apiKey, time.Now().Add(-1*time.Second), "insufficient_balance")
	m.recoverExpiredCircuitBreakers()
	if m.IsKeyHardSuspended(baseURL, apiKey) {
		t.Fatalf("expected hard suspend cleared")
	}
	if m.ShouldSuspendKey(baseURL, apiKey) {
		t.Fatalf("expected ShouldSuspendKey=false after recovery")
	}
}
