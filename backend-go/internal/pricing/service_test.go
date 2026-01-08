package pricing

import (
	"encoding/json"
	"testing"
)

func TestJSONInt_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want jsonInt
	}{
		{"number", `123`, 123},
		{"string-number", `"456"`, 456},
		{"null", `null`, 0},
		{"empty-string", `""`, 0},
		{"float", `12.9`, 12},
		{"garbage-string", `"max input tokens, if the provider specifies it. if not default to max_tokens"`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got jsonInt
			if err := json.Unmarshal([]byte(tt.in), &got); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d want %d", got, tt.want)
			}
		})
	}
}
