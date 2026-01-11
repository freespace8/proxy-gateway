package utils

import (
	"strings"
	"testing"
)

func TestStreamSynthesizer_MergeSplitToolCalls(t *testing.T) {
	s := NewStreamSynthesizer("claude")
	s.toolCallAccumulator = map[int]*ToolCall{
		0: {ID: "tool_1", Name: "do_thing", Arguments: "{}"},
		1: {ID: "tool_1", Name: "", Arguments: `{"a":1}`},
	}

	out := s.GetSynthesizedContent()
	if strings.Count(out, "Tool Call:") != 1 {
		t.Fatalf("expected 1 tool call, got: %s", out)
	}
	if !strings.Contains(out, "do_thing") {
		t.Fatalf("expected tool name in output, got: %s", out)
	}
	if strings.Contains(out, "unknown_function") {
		t.Fatalf("did not expect unknown_function in output, got: %s", out)
	}
	if !strings.Contains(out, `{"a":1}`) {
		t.Fatalf("expected merged arguments in output, got: %s", out)
	}
}
