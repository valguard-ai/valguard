//go:build !osspublic

package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type ToolCallWhitelist struct{}

func (ToolCallWhitelist) Name() string { return "tool_call_whitelist" }

func (ToolCallWhitelist) Run(ctx context.Context, output string, params map[string]any) Result {
	allowed := stringSliceParam(params["allowed_tools"])
	if len(allowed) == 0 {
		return skipResult("no allowed_tools configured")
	}
	calls, err := parseToolCalls(output)
	if err != nil {
		return failResult("error", err.Error(), nil)
	}
	if len(calls) == 0 {
		return passResult("error", "no tool calls in output")
	}
	allowSet := make(map[string]struct{})
	for _, a := range allowed {
		allowSet[strings.ToLower(a)] = struct{}{}
	}
	var denied []string
	for _, c := range calls {
		if _, ok := allowSet[strings.ToLower(c)]; !ok {
			denied = append(denied, c)
		}
	}
	if len(denied) > 0 {
		return failResult("error", "tool not in whitelist", map[string]any{"denied": denied})
	}
	return passResult("error", "all tool calls whitelisted")
}

type ToolCallArgsSchema struct{}

func (ToolCallArgsSchema) Name() string { return "tool_call_args_schema" }

func (ToolCallArgsSchema) Run(ctx context.Context, output string, params map[string]any) Result {
	schemas, _ := params["schemas"].(map[string]any)
	if len(schemas) == 0 {
		return skipResult("no schemas configured")
	}
	calls, err := parseToolCallsWithArgs(output)
	if err != nil {
		return failResult("error", err.Error(), nil)
	}
	for name, args := range calls {
		rawSchema, ok := schemas[name]
		if !ok {
			continue
		}
		schema, ok := rawSchema.(map[string]any)
		if !ok {
			continue
		}
		if req, ok := schema["required"].([]any); ok {
			for _, f := range req {
				field, _ := f.(string)
				if field == "" {
					continue
				}
				if _, present := args[field]; !present {
					return failResult("error", fmt.Sprintf("tool %s missing required arg %s", name, field), nil)
				}
			}
		}
	}
	return passResult("error", "tool call args match schemas")
}

type FunctionSignatureMatch struct{}

func (FunctionSignatureMatch) Name() string { return "function_signature_match" }

func (FunctionSignatureMatch) Run(ctx context.Context, output string, params map[string]any) Result {
	sigs, _ := params["signatures"].(map[string]any)
	if len(sigs) == 0 {
		return skipResult("no signatures configured")
	}
	for fn, sigVal := range sigs {
		pattern, ok := sigVal.(string)
		if !ok || pattern == "" {
			continue
		}
		re, err := compileCached(pattern)
		if err != nil {
			return failResult("error", "invalid signature regex for "+fn, map[string]any{"error": err.Error()})
		}
		if strings.Contains(output, fn+"(") && !re.MatchString(output) {
			return failResult("error", "function call does not match signature pattern", map[string]any{"function": fn})
		}
	}
	return passResult("error", "function signatures match")
}

type StepCountMax struct{}

func (StepCountMax) Name() string { return "step_count_max" }

func (StepCountMax) Run(ctx context.Context, output string, params map[string]any) Result {
	max := 10
	if v, ok := params["max_steps"].(float64); ok && v > 0 {
		max = int(v)
	}
	count := countWorkflowSteps(output)
	if count > max {
		return failResult("error", fmt.Sprintf("too many steps (%d > %d)", count, max), map[string]any{"count": count, "max": max})
	}
	return passResult("error", fmt.Sprintf("step count %d within limit", count))
}

var (
	workflowStepHeaderRe = regexp.MustCompile(`(?im)^\s*#{1,3}\s*step\s+\d+`)
	workflowStepNumRe    = regexp.MustCompile(`(?im)^\s*\d+[\.\)]\s+`)
	workflowStepJSONRe   = regexp.MustCompile(`(?i)"step"\s*:`)
	toolCallFallbackRe   = regexp.MustCompile(`(?i)(?:function|tool)[_ ]?call[:\s]+["']?([A-Za-z0-9_\-]+)`)
)

func countWorkflowSteps(output string) int {
	max := 0
	for _, re := range [...]*regexp.Regexp{workflowStepHeaderRe, workflowStepNumRe, workflowStepJSONRe} {
		n := len(re.FindAllStringIndex(output, -1))
		if n > max {
			max = n
		}
	}
	if max == 0 && strings.Contains(strings.ToLower(output), "step") {
		return 1
	}
	return max
}

func parseToolCalls(output string) ([]string, error) {
	m, err := parseToolCallsWithArgs(output)
	if err != nil {
		return nil, err
	}
	var names []string
	for k := range m {
		names = append(names, k)
	}
	return names, nil
}

func parseToolCallsWithArgs(output string) (map[string]map[string]any, error) {
	trim := strings.TrimSpace(output)
	if trim == "" {
		return nil, nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(trim), &data); err == nil {
		return toolCallsFromJSON(data)
	}
	// OpenAI-style tool_calls in assistant message JSON
	var arr []map[string]any
	if err := json.Unmarshal([]byte(trim), &arr); err == nil {
		out := make(map[string]map[string]any)
		for _, item := range arr {
			if tc, ok := item["tool_calls"].([]any); ok {
				mergeToolCalls(out, tc)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	// Fallback: function_name( in text
	matches := toolCallFallbackRe.FindAllStringSubmatch(output, -1)
	out := make(map[string]map[string]any)
	for _, m := range matches {
		if len(m) > 1 {
			out[m[1]] = map[string]any{}
		}
	}
	return out, nil
}

func toolCallsFromJSON(data map[string]any) (map[string]map[string]any, error) {
	out := make(map[string]map[string]any)
	if tc, ok := data["tool_calls"].([]any); ok {
		mergeToolCalls(out, tc)
	}
	if choices, ok := data["choices"].([]any); ok {
		for _, c := range choices {
			choice, _ := c.(map[string]any)
			msg, _ := choice["message"].(map[string]any)
			if tc, ok := msg["tool_calls"].([]any); ok {
				mergeToolCalls(out, tc)
			}
		}
	}
	return out, nil
}

func mergeToolCalls(out map[string]map[string]any, tc []any) {
	for _, item := range tc {
		entry, _ := item.(map[string]any)
		fn, _ := entry["function"].(map[string]any)
		name, _ := fn["name"].(string)
		if name == "" {
			name, _ = entry["name"].(string)
		}
		if name == "" {
			continue
		}
		args := map[string]any{}
		if raw, ok := fn["arguments"].(string); ok {
			_ = json.Unmarshal([]byte(raw), &args)
		}
		out[name] = args
	}
}
