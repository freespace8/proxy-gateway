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
