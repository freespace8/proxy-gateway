package metrics

import (
	"testing"
	"time"
)

func TestToResponse_CircuitBrokenIncludesHardSuspend(t *testing.T) {
	m := NewMetricsManagerWithConfig(10, 0.5)
	defer m.Stop()

	baseURL := "https://example.com"
	apiKey := "k1"

	m.SuspendKeyUntil(baseURL, apiKey, time.Now().Add(1*time.Hour), "insufficient_balance")

	resp := m.ToResponse(0, baseURL, []string{apiKey}, 0)
	if resp == nil || len(resp.KeyMetrics) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if !resp.KeyMetrics[0].CircuitBroken {
		t.Fatalf("expected circuitBroken=true when hard-suspended")
	}

	// 过期但未触发后台清理时，也应视为未熔断
	m.SuspendKeyUntil(baseURL, apiKey, time.Now().Add(-1*time.Second), "insufficient_balance")
	resp2 := m.ToResponse(0, baseURL, []string{apiKey}, 0)
	if resp2 == nil || len(resp2.KeyMetrics) != 1 {
		t.Fatalf("unexpected response2: %+v", resp2)
	}
	if resp2.KeyMetrics[0].CircuitBroken {
		t.Fatalf("expected circuitBroken=false when hard-suspend expired")
	}
}

func TestToResponseMultiURL_CircuitBrokenIncludesHardSuspend(t *testing.T) {
	m := NewMetricsManagerWithConfig(10, 0.5)
	defer m.Stop()

	baseURL := "https://example.com"
	apiKey := "k1"

	m.SuspendKeyUntil(baseURL, apiKey, time.Now().Add(1*time.Hour), "insufficient_balance")

	resp := m.ToResponseMultiURL(0, []string{baseURL}, []string{apiKey}, 0)
	if resp == nil || len(resp.KeyMetrics) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if !resp.KeyMetrics[0].CircuitBroken {
		t.Fatalf("expected circuitBroken=true when hard-suspended")
	}

	// 过期但未触发后台清理时，也应视为未熔断
	m.SuspendKeyUntil(baseURL, apiKey, time.Now().Add(-1*time.Second), "insufficient_balance")
	resp2 := m.ToResponseMultiURL(0, []string{baseURL}, []string{apiKey}, 0)
	if resp2 == nil || len(resp2.KeyMetrics) != 1 {
		t.Fatalf("unexpected response2: %+v", resp2)
	}
	if resp2.KeyMetrics[0].CircuitBroken {
		t.Fatalf("expected circuitBroken=false when hard-suspend expired")
	}
}
