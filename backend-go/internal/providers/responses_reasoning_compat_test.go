package providers

import "testing"

func TestNormalizeResponsesReasoningEffort_Gpt52MinimalToLow(t *testing.T) {
	req := map[string]interface{}{
		"model": "gpt-5.2",
		"reasoning": map[string]interface{}{
			"effort": "minimal",
		},
	}

	normalizeResponsesReasoningEffort(req)

	reasoning := req["reasoning"].(map[string]interface{})
	if got := reasoning["effort"]; got != "low" {
		t.Fatalf("effort=%v, want %q", got, "low")
	}
}

func TestNormalizeResponsesReasoningEffort_Gpt53CodexMinimalToLow(t *testing.T) {
	req := map[string]interface{}{
		"model": "gpt-5.3-codex",
		"reasoning": map[string]interface{}{
			"effort": "minimal",
		},
	}

	normalizeResponsesReasoningEffort(req)

	reasoning := req["reasoning"].(map[string]interface{})
	if got := reasoning["effort"]; got != "low" {
		t.Fatalf("effort=%v, want %q", got, "low")
	}
}

func TestNormalizeResponsesReasoningEffort_OtherModelsUnchanged(t *testing.T) {
	req := map[string]interface{}{
		"model": "gpt-5.1",
		"reasoning": map[string]interface{}{
			"effort": "minimal",
		},
	}

	normalizeResponsesReasoningEffort(req)

	reasoning := req["reasoning"].(map[string]interface{})
	if got := reasoning["effort"]; got != "minimal" {
		t.Fatalf("effort=%v, want %q", got, "minimal")
	}
}

func TestRedirectResponsesReasoningEffort(t *testing.T) {
	req := map[string]interface{}{
		"reasoning": map[string]interface{}{
			"effort": "low",
		},
		"reasoning_effort": "medium",
	}

	redirectResponsesReasoningEffort(req, map[string]string{
		"low":    "xhigh",
		"medium": "high",
	})

	reasoning := req["reasoning"].(map[string]interface{})
	if got := reasoning["effort"]; got != "xhigh" {
		t.Fatalf("reasoning.effort=%v, want %q", got, "xhigh")
	}
	if got := req["reasoning_effort"]; got != "high" {
		t.Fatalf("reasoning_effort=%v, want %q", got, "high")
	}
}
