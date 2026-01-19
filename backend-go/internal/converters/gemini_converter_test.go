package converters

import (
	"testing"
)

// TestClaudeResponseToGemini_WithThoughtSignature 测试 Claude 响应转换时 thought_signature 的处理
func TestClaudeResponseToGemini_WithThoughtSignature(t *testing.T) {
	t.Run("保留原有 signature", func(t *testing.T) {
		// 测试场景 1: Claude 响应包含 signature
		claudeResp := map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":      "tool_use",
					"name":      "test_function",
					"input":     map[string]interface{}{"arg": "value"},
					"signature": "original_signature_from_claude",
				},
			},
		}

		geminiResp, err := ClaudeResponseToGemini(claudeResp)
		if err != nil {
			t.Fatalf("ClaudeResponseToGemini: %v", err)
		}
		if geminiResp == nil {
			t.Fatalf("geminiResp is nil")
		}
		if len(geminiResp.Candidates) != 1 {
			t.Fatalf("candidates=%d, want 1", len(geminiResp.Candidates))
		}
		if geminiResp.Candidates[0].Content == nil {
			t.Fatalf("candidate content is nil")
		}
		if len(geminiResp.Candidates[0].Content.Parts) != 1 {
			t.Fatalf("parts=%d, want 1", len(geminiResp.Candidates[0].Content.Parts))
		}
		if geminiResp.Candidates[0].Content.Parts[0].FunctionCall == nil {
			t.Fatalf("function call is nil")
		}
		if got, want := geminiResp.Candidates[0].Content.Parts[0].FunctionCall.ThoughtSignature, "original_signature_from_claude"; got != want {
			t.Fatalf("thought signature=%q, want %q", got, want)
		}
	})

	t.Run("使用 dummy signature", func(t *testing.T) {
		// 测试场景 2: Claude 响应不包含 signature
		claudeResp := map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":  "tool_use",
					"name":  "test_function",
					"input": map[string]interface{}{"arg": "value"},
				},
			},
		}

		geminiResp, err := ClaudeResponseToGemini(claudeResp)
		if err != nil {
			t.Fatalf("ClaudeResponseToGemini: %v", err)
		}
		if geminiResp == nil {
			t.Fatalf("geminiResp is nil")
		}
		if len(geminiResp.Candidates) != 1 {
			t.Fatalf("candidates=%d, want 1", len(geminiResp.Candidates))
		}
		if geminiResp.Candidates[0].Content == nil {
			t.Fatalf("candidate content is nil")
		}
		if len(geminiResp.Candidates[0].Content.Parts) != 1 {
			t.Fatalf("parts=%d, want 1", len(geminiResp.Candidates[0].Content.Parts))
		}
		if geminiResp.Candidates[0].Content.Parts[0].FunctionCall == nil {
			t.Fatalf("function call is nil")
		}
		if got, want := geminiResp.Candidates[0].Content.Parts[0].FunctionCall.ThoughtSignature, dummyThoughtSignature; got != want {
			t.Fatalf("thought signature=%q, want %q", got, want)
		}
	})

	t.Run("空 signature 使用 dummy", func(t *testing.T) {
		// 测试场景 3: Claude 响应包含空 signature
		claudeResp := map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":      "tool_use",
					"name":      "test_function",
					"input":     map[string]interface{}{"arg": "value"},
					"signature": "",
				},
			},
		}

		geminiResp, err := ClaudeResponseToGemini(claudeResp)
		if err != nil {
			t.Fatalf("ClaudeResponseToGemini: %v", err)
		}
		if geminiResp == nil {
			t.Fatalf("geminiResp is nil")
		}
		if len(geminiResp.Candidates) != 1 {
			t.Fatalf("candidates=%d, want 1", len(geminiResp.Candidates))
		}
		if geminiResp.Candidates[0].Content == nil {
			t.Fatalf("candidate content is nil")
		}
		if len(geminiResp.Candidates[0].Content.Parts) != 1 {
			t.Fatalf("parts=%d, want 1", len(geminiResp.Candidates[0].Content.Parts))
		}
		if geminiResp.Candidates[0].Content.Parts[0].FunctionCall == nil {
			t.Fatalf("function call is nil")
		}
		if got, want := geminiResp.Candidates[0].Content.Parts[0].FunctionCall.ThoughtSignature, dummyThoughtSignature; got != want {
			t.Fatalf("thought signature=%q, want %q", got, want)
		}
	})
}

// TestOpenAIResponseToGemini_WithThoughtSignature 测试 OpenAI 响应转换时 thought_signature 的处理
func TestOpenAIResponseToGemini_WithThoughtSignature(t *testing.T) {
	t.Run("统一使用 dummy signature", func(t *testing.T) {
		openaiResp := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"tool_calls": []interface{}{
							map[string]interface{}{
								"function": map[string]interface{}{
									"name":      "test_function",
									"arguments": `{"arg":"value"}`,
								},
							},
						},
					},
				},
			},
		}

		geminiResp, err := OpenAIResponseToGemini(openaiResp)
		if err != nil {
			t.Fatalf("OpenAIResponseToGemini: %v", err)
		}
		if geminiResp == nil {
			t.Fatalf("geminiResp is nil")
		}
		if len(geminiResp.Candidates) != 1 {
			t.Fatalf("candidates=%d, want 1", len(geminiResp.Candidates))
		}
		if geminiResp.Candidates[0].Content == nil {
			t.Fatalf("candidate content is nil")
		}
		if len(geminiResp.Candidates[0].Content.Parts) != 1 {
			t.Fatalf("parts=%d, want 1", len(geminiResp.Candidates[0].Content.Parts))
		}
		if geminiResp.Candidates[0].Content.Parts[0].FunctionCall == nil {
			t.Fatalf("function call is nil")
		}
		if got, want := geminiResp.Candidates[0].Content.Parts[0].FunctionCall.ThoughtSignature, dummyThoughtSignature; got != want {
			t.Fatalf("thought signature=%q, want %q", got, want)
		}
	})

	t.Run("多个工具调用都包含 signature", func(t *testing.T) {
		openaiResp := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"tool_calls": []interface{}{
							map[string]interface{}{
								"function": map[string]interface{}{
									"name":      "function1",
									"arguments": `{"arg1":"value1"}`,
								},
							},
							map[string]interface{}{
								"function": map[string]interface{}{
									"name":      "function2",
									"arguments": `{"arg2":"value2"}`,
								},
							},
						},
					},
				},
			},
		}

		geminiResp, err := OpenAIResponseToGemini(openaiResp)
		if err != nil {
			t.Fatalf("OpenAIResponseToGemini: %v", err)
		}
		if geminiResp == nil {
			t.Fatalf("geminiResp is nil")
		}
		if len(geminiResp.Candidates) != 1 {
			t.Fatalf("candidates=%d, want 1", len(geminiResp.Candidates))
		}
		if geminiResp.Candidates[0].Content == nil {
			t.Fatalf("candidate content is nil")
		}
		if len(geminiResp.Candidates[0].Content.Parts) != 2 {
			t.Fatalf("parts=%d, want 2", len(geminiResp.Candidates[0].Content.Parts))
		}

		// 验证所有工具调用都包含 dummy signature
		for _, part := range geminiResp.Candidates[0].Content.Parts {
			if part.FunctionCall == nil {
				t.Fatalf("function call is nil")
			}
			if got, want := part.FunctionCall.ThoughtSignature, dummyThoughtSignature; got != want {
				t.Fatalf("thought signature=%q, want %q", got, want)
			}
		}
	})
}
