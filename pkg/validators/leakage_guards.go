package validators

import (
	"context"
	"strings"
)

type NoThinkingTagLeakage struct{ patternValidator }

func (NoThinkingTagLeakage) Name() string { return "no_thinking_tag_leakage" }

func (v NoThinkingTagLeakage) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{"<think", "</think>", "<thinking>", "</thinking>", `\[thinking\]`}
	}
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

func (v NoThinkingTagLeakage) Compile(params map[string]any) CompiledRunner {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{"<think", "</think>", "<thinking>", "</thinking>", `\[thinking\]`}
	}
	name := v.Name()
	return func(_ context.Context, output string) Result {
		return patternValidator{name: name}.runPatterns(output, p, "patterns")
	}
}

type NoSystemPromptLeakage struct{}

func (NoSystemPromptLeakage) Name() string { return "no_system_prompt_leakage" }

func (NoSystemPromptLeakage) Run(ctx context.Context, output string, params map[string]any) Result {
	markers := stringSliceParam(params["forbidden_markers"])
	if len(markers) == 0 {
		markers = []string{"SYSTEM:", "You are a helpful", "Your task is to", "IMPORTANT INSTRUCTIONS"}
	}
	var hit []string
	lower := strings.ToLower(output)
	for _, m := range markers {
		if strings.Contains(lower, strings.ToLower(m)) {
			hit = append(hit, m)
		}
	}
	if len(hit) > 0 {
		return failResult("error", "system prompt leakage", map[string]any{"markers": hit})
	}
	return passResult("error", "ok")
}

type NoRoleMarkerLeakage struct{ patternValidator }

func (NoRoleMarkerLeakage) Name() string { return "no_role_marker_leakage" }

func (v NoRoleMarkerLeakage) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{`^User:`, `^Assistant:`, `^Human:`, `^System:`}
	}
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

func (v NoRoleMarkerLeakage) Compile(params map[string]any) CompiledRunner {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{`^User:`, `^Assistant:`, `^Human:`, `^System:`}
	}
	name := v.Name()
	return func(_ context.Context, output string) Result {
		return patternValidator{name: name}.runPatterns(output, p, "patterns")
	}
}

type NoTruncationMarkers struct{ patternValidator }

func (NoTruncationMarkers) Name() string { return "no_truncation_markers" }

func (v NoTruncationMarkers) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{`\[truncated\]`, `\[continued\]`, `\.\.\.\s*$`, `\[cut off\]`}
	}
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

func (v NoTruncationMarkers) Compile(params map[string]any) CompiledRunner {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{`\[truncated\]`, `\[continued\]`, `\.\.\.\s*$`, `\[cut off\]`}
	}
	name := v.Name()
	return func(_ context.Context, output string) Result {
		return patternValidator{name: name}.runPatterns(output, p, "patterns")
	}
}

type NoPlaceholderText struct{ patternValidator }

func (NoPlaceholderText) Name() string { return "no_placeholder_text" }

func (v NoPlaceholderText) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{`\[TBD\]`, `\[TODO\]`, `\[PLACEHOLDER\]`, `^\\?$`, `\bunknown\b`, `\bN/A\b`}
	}
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

func (v NoPlaceholderText) Compile(params map[string]any) CompiledRunner {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{`\[TBD\]`, `\[TODO\]`, `\[PLACEHOLDER\]`, `^\\?$`, `\bunknown\b`, `\bN/A\b`}
	}
	name := v.Name()
	return func(_ context.Context, output string) Result {
		return patternValidator{name: name}.runPatterns(output, p, "patterns")
	}
}
