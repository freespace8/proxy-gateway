package providers

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/BenedictKing/claude-proxy/internal/types"
)

func TestExtractSystemText_SupportsStringObjectAndArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		system interface{}
		want   string
	}{
		{
			name:   "string",
			system: "sys",
			want:   "sys",
		},
		{
			name: "single_object_text",
			system: map[string]interface{}{
				"type": "text",
				"text": "a",
			},
			want: "a",
		},
		{
			name: "array_text_and_nontext",
			system: []interface{}{
				map[string]interface{}{"type": "text", "text": "a"},
				map[string]interface{}{"type": "image", "text": "ignored"},
				map[string]interface{}{"type": "text", "text": "b"},
			},
			want: "a\nb",
		},
		{
			name: "unknown_type",
			system: map[string]interface{}{
				"type": "image",
				"text": "ignored",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractSystemText(tt.system); got != tt.want {
				t.Fatalf("extractSystemText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenAIProvider_ConvertMessages_ToolResultsPrecedeTextAndToolCalls(t *testing.T) {
	t.Parallel()

	p := &OpenAIProvider{}
	claudeReq := &types.ClaudeRequest{
		Model: "m",
		Messages: []types.ClaudeMessage{
			{
				Role: "assistant",
				Content: []interface{}{
					map[string]interface{}{
						"type":  "tool_use",
						"id":    "toolu_1",
						"name":  "get_weather",
						"input": map[string]interface{}{"location": "Boston"},
					},
				},
			},
			{
				Role: "user",
				Content: []interface{}{
					map[string]interface{}{"type": "text", "text": "hi"},
				},
			},
			{
				Role: "tool",
				Content: []interface{}{
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": "toolu_1",
						"content":     "ok",
					},
					map[string]interface{}{"type": "text", "text": "tool-says-hi"},
				},
			},
		},
	}

	msgs := p.convertMessages(claudeReq)

	if len(msgs) < 3 {
		t.Fatalf("expected >=3 messages, got %d", len(msgs))
	}

	if msgs[0].Role != "assistant" || len(msgs[0].ToolCalls) != 1 || msgs[0].ToolCalls[0].ID != "toolu_1" {
		t.Fatalf("expected first message assistant with tool_calls toolu_1, got %+v", msgs[0])
	}

	if msgs[1].Role != "user" {
		t.Fatalf("expected second message user, got %+v", msgs[1])
	}

	if msgs[2].Role != "tool" || msgs[2].ToolCallID != "toolu_1" {
		t.Fatalf("expected third message tool with tool_call_id=toolu_1, got %+v", msgs[2])
	}
}

func TestOpenAIProvider_HandleStreamResponse_EmitsToolUseStopOnlyOnFinishReason(t *testing.T) {
	t.Parallel()

	marshal := func(v interface{}) string {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		return string(b)
	}

	// tool_calls arguments 分片 + finish_reason=tool_calls，确保 stop_reason=tool_use 只在 finish_reason 到达后发送
	sse := strings.Join([]string{
		"data: " + marshal(map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"delta": map[string]interface{}{
						"tool_calls": []interface{}{
							map[string]interface{}{
								"index": 0,
								"id":    "call_1",
								"function": map[string]interface{}{
									"name":      "get_weather",
									"arguments": "{\"location\": ",
								},
							},
						},
					},
				},
			},
		}),
		"data: " + marshal(map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"delta": map[string]interface{}{
						"tool_calls": []interface{}{
							map[string]interface{}{
								"index": 0,
								"function": map[string]interface{}{
									"arguments": "\"Boston\"}",
								},
							},
						},
					},
				},
			},
		}),
		"data: " + marshal(map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"delta":         map[string]interface{}{},
					"finish_reason": "tool_calls",
				},
			},
		}),
		`data: [DONE]`,
	}, "\n")

	body := io.NopCloser(strings.NewReader(sse))

	eventCh, errCh, err := (&OpenAIProvider{}).HandleStreamResponse(body)
	if err != nil {
		t.Fatalf("HandleStreamResponse() err: %v", err)
	}

	var events []string
	for e := range eventCh {
		events = append(events, e)
	}
	select {
	case e := <-errCh:
		if e != nil {
			t.Fatalf("unexpected stream error: %v", e)
		}
	default:
	}

	stopIdx := -1
	toolStopIdx := -1
	stopCount := 0
	for i, e := range events {
		if strings.Contains(e, `"type":"content_block_stop"`) && strings.Contains(e, `"index":0`) {
			toolStopIdx = i
		}
		if strings.Contains(e, `"stop_reason":"tool_use"`) {
			stopIdx = i
			stopCount++
		}
	}

	if stopCount != 1 {
		t.Fatalf("expected exactly 1 tool_use stop, got %d; events=%v", stopCount, events)
	}
	if toolStopIdx == -1 || stopIdx == -1 || stopIdx < toolStopIdx {
		t.Fatalf("expected stop_reason after tool_use content_block_stop; toolStopIdx=%d stopIdx=%d; events=%v", toolStopIdx, stopIdx, events)
	}
}
