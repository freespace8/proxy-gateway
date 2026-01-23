package metrics

import (
	"net/http"
	"testing"
)

func TestRecordFailureWithStatus_502DoesNotTriggerSoftSuspend(t *testing.T) {
	m := NewMetricsManagerWithConfig(3, 0.5)

	baseURL := "https://u.example.com"
	apiKey := "k-test"

	m.RecordFailureWithStatus(baseURL, apiKey, http.StatusBadGateway)
	m.RecordFailureWithStatus(baseURL, apiKey, http.StatusBadGateway)
	m.RecordFailureWithStatus(baseURL, apiKey, http.StatusBadGateway)

	km := m.GetKeyMetrics(baseURL, apiKey)
	if km == nil {
		t.Fatal("expected key metrics to exist")
	}
	if km.FailureCount != 3 {
		t.Fatalf("FailureCount=%d, want %d", km.FailureCount, 3)
	}
	if km.ConsecutiveFailures != 3 {
		t.Fatalf("ConsecutiveFailures=%d, want %d", km.ConsecutiveFailures, 3)
	}
	if km.CircuitBrokenAt != nil {
		t.Fatalf("CircuitBrokenAt=%v, want nil", km.CircuitBrokenAt)
	}
	if m.ShouldSuspendKeySoft(baseURL, apiKey) {
		t.Fatal("expected ShouldSuspendKeySoft=false for repeated 502")
	}
}

func TestRecordSuccess_ClearsSoftSuspendWindowImmediately(t *testing.T) {
	m := NewMetricsManagerWithConfig(3, 0.5)

	baseURL := "https://u.example.com"
	apiKey := "k-test"

	m.RecordFailureWithStatus(baseURL, apiKey, http.StatusInternalServerError)
	m.RecordFailureWithStatus(baseURL, apiKey, http.StatusInternalServerError)
	m.RecordFailureWithStatus(baseURL, apiKey, http.StatusInternalServerError)

	if !m.ShouldSuspendKeySoft(baseURL, apiKey) {
		t.Fatal("expected key to be soft suspended after repeated 5xx failures")
	}

	m.RecordSuccess(baseURL, apiKey)

	if m.ShouldSuspendKeySoft(baseURL, apiKey) {
		t.Fatal("expected key to recover immediately after success")
	}

	km := m.GetKeyMetrics(baseURL, apiKey)
	if km == nil {
		t.Fatal("expected key metrics to exist")
	}
	if km.ConsecutiveFailures != 0 {
		t.Fatalf("ConsecutiveFailures=%d, want 0", km.ConsecutiveFailures)
	}
}
