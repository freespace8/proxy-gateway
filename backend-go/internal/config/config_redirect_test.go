package config

import "testing"

func TestRedirectModelWithGlobal_ChannelMappingHasPriority(t *testing.T) {
	upstream := &UpstreamConfig{ModelMapping: map[string]string{"gpt-5": "channel-model"}}
	global := map[string]string{"gpt-5": "global-model"}

	got := RedirectModelWithGlobal("gpt-5", upstream, global)
	if got != "channel-model" {
		t.Fatalf("got=%q want=%q", got, "channel-model")
	}
}

func TestRedirectModelWithGlobal_FallbackToGlobalMapping(t *testing.T) {
	upstream := &UpstreamConfig{ModelMapping: map[string]string{"other": "x"}}
	global := map[string]string{"gpt-5": "global-model"}

	got := RedirectModelWithGlobal("gpt-5", upstream, global)
	if got != "global-model" {
		t.Fatalf("got=%q want=%q", got, "global-model")
	}
}

func TestRedirectModelWithGlobal_LongestPatternWins(t *testing.T) {
	upstream := &UpstreamConfig{ModelMapping: map[string]string{
		"codex":         "short",
		"gpt-5.1-codex": "long",
	}}

	got := RedirectModelWithGlobal("gpt-5.1-codex", upstream, nil)
	if got != "long" {
		t.Fatalf("got=%q want=%q", got, "long")
	}
}

func TestRedirectReasoningEffort(t *testing.T) {
	mapping := map[string]string{"low": "xhigh"}

	if got := RedirectReasoningEffort("low", mapping); got != "xhigh" {
		t.Fatalf("got=%q want=%q", got, "xhigh")
	}
	if got := RedirectReasoningEffort("medium", mapping); got != "medium" {
		t.Fatalf("got=%q want=%q", got, "medium")
	}
}

func TestRedirectModelWithGlobal_ChannelFuzzyOverridesGlobalExact(t *testing.T) {
	upstream := &UpstreamConfig{ModelMapping: map[string]string{"gpt-5": "channel-model"}}
	global := map[string]string{"gpt-5-mini": "global-model"}

	got := RedirectModelWithGlobal("gpt-5-mini", upstream, global)
	if got != "channel-model" {
		t.Fatalf("got=%q want=%q", got, "channel-model")
	}
}

func TestRedirectModel_SameLengthPatternDeterministic(t *testing.T) {
	upstream := &UpstreamConfig{ModelMapping: map[string]string{
		"gpt-5.1": "first",
		"gpt-5.2": "second",
	}}

	got := RedirectModel("gpt-5", upstream)
	if got != "first" {
		t.Fatalf("got=%q want=%q", got, "first")
	}
}

func TestRedirectReasoningEffort_EmptyInputNoRedirect(t *testing.T) {
	mapping := map[string]string{"low": "xhigh"}

	if got := RedirectReasoningEffort("", mapping); got != "" {
		t.Fatalf("got=%q want=%q", got, "")
	}
}
