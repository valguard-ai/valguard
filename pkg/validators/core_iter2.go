//go:build !osspublic

package validators

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

type FieldPresenceRules struct{}

func (FieldPresenceRules) Name() string { return "field_presence_rules" }

func (FieldPresenceRules) Run(ctx context.Context, output string, params map[string]any) Result {
	rules, _ := params["rules"].([]any)
	if len(rules) == 0 {
		return skipResult("no rules")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	for _, r := range rules {
		rule, _ := r.(map[string]any)
		if rule == nil {
			continue
		}
		ifField, _ := rule["if_field"].(string)
		thenField, _ := rule["then_field"].(string)
		if ifField == "" || thenField == "" {
			continue
		}
		if _, hasIf := data[ifField]; hasIf {
			if _, hasThen := data[thenField]; !hasThen {
				return failResult("error", "conditional field missing", map[string]any{"if": ifField, "then": thenField})
			}
		}
	}
	return passResult("error", "presence rules ok")
}

type RequiredSections struct{}

func (RequiredSections) Name() string { return "required_sections" }

func (RequiredSections) Run(ctx context.Context, output string, params map[string]any) Result {
	sections := stringSliceParam(params["sections"])
	if len(sections) == 0 {
		return skipResult("no sections configured")
	}
	lower := outputLowerFrom(ctx, output)
	var missing []string
	for _, s := range sections {
		if !strings.Contains(lower, strings.ToLower(s)) {
			missing = append(missing, s)
		}
	}
	if len(missing) > 0 {
		return failResult("error", "missing sections", map[string]any{"missing": missing})
	}
	return passResult("error", "sections present")
}

type MarkdownFormat struct{}

func (MarkdownFormat) Name() string { return "markdown_format" }

func (MarkdownFormat) Run(ctx context.Context, output string, _ map[string]any) Result {
	if strings.TrimSpace(output) == "" {
		return failResult("warn", "empty output", nil)
	}
	hasStructure := strings.Contains(output, "#") || strings.Contains(output, "- ") || strings.Contains(output, "* ")
	if !hasStructure && len(output) > 200 {
		return failResult("warn", "long text without markdown structure", nil)
	}
	return passResult("warn", "markdown ok")
}

var (
	numericMatchRe = regexp.MustCompile(`\d+(?:\.\d+)?`)
	urlSafetyRe    = regexp.MustCompile(`https?://([^\s/]+)`)
	sqlInjectionPatterns = []string{
		`(?i)(\bOR\b\s+\d+\s*=\s*\d+)`,
		`(?i)(DROP\s+TABLE)`,
		`(?i)(UNION\s+SELECT)`,
		`(?i)(--\s*$)`,
	}
	sqlInjectionCompiled = mustCompileAll(sqlInjectionPatterns)
)

type DependencyRules struct{}

func (DependencyRules) Name() string { return "dependency_rules" }

func (DependencyRules) Run(ctx context.Context, output string, params map[string]any) Result {
	return (FieldPresenceRules{}).Run(ctx, output, map[string]any{"rules": params["rules"]})
}

type NumericMatch struct{}

func (NumericMatch) Name() string { return "numeric_match" }

func (NumericMatch) Run(ctx context.Context, output string, params map[string]any) Result {
	meta := RunMetaFrom(ctx)
	if meta.InputText == "" {
		return skipResult("no input text for numeric_match")
	}
	inNums := numericMatchRe.FindAllString(meta.InputText, -1)
	outNums := numericMatchRe.FindAllString(output, -1)
	outSet := make(map[string]struct{}, len(outNums))
	for _, o := range outNums {
		outSet[o] = struct{}{}
	}
	for _, n := range inNums {
		if _, found := outSet[n]; !found {
			return failResult("error", "number from input missing in output", map[string]any{"missing": n})
		}
	}
	return passResult("error", "numbers preserved")
}

type SQLInjection struct{}

func (SQLInjection) Name() string { return "sql_injection" }

func (SQLInjection) Run(ctx context.Context, output string, _ map[string]any) Result {
	if m := matchFirstCompiled(output, sqlInjectionCompiled, sqlInjectionPatterns); m != "" {
		return failResult("error", "sql injection pattern", map[string]any{"matched": m})
	}
	return passResult("error", "no sql injection patterns")
}

func (SQLInjection) Compile(_ map[string]any) CompiledRunner {
	return func(_ context.Context, output string) Result {
		if m := matchFirstCompiled(output, sqlInjectionCompiled, sqlInjectionPatterns); m != "" {
			return failResult("error", "sql injection pattern", map[string]any{"matched": m})
		}
		return passResult("error", "no sql injection patterns")
	}
}

type URLSafety struct{}

func (URLSafety) Name() string { return "url_safety" }

func (URLSafety) Run(ctx context.Context, output string, params map[string]any) Result {
	allowed := stringSliceParam(params["allowed_domains"])
	if len(allowed) == 0 {
		return skipResult("allowed_domains required")
	}
	matches := urlSafetyRe.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		host := strings.ToLower(m[1])
		ok := false
		for _, d := range allowed {
			if host == strings.ToLower(d) || strings.HasSuffix(host, "."+strings.ToLower(d)) {
				ok = true
				break
			}
		}
		if !ok {
			return failResult("error", "url domain not allowed", map[string]any{"host": host})
		}
	}
	return passResult("error", "urls ok")
}

// --- meta ---

type KeywordPreservation struct{}

func (KeywordPreservation) Name() string { return "keyword_preservation" }

func (KeywordPreservation) Run(ctx context.Context, output string, params map[string]any) Result {
	keywords := stringSliceParam(params["keywords"])
	if len(keywords) == 0 {
		meta := RunMetaFrom(ctx)
		if meta.InputText != "" {
			for _, w := range strings.Fields(meta.InputText) {
				if len(w) > 5 {
					keywords = append(keywords, w)
				}
			}
		}
	}
	if len(keywords) == 0 {
		return skipResult("no keywords")
	}
	lower := outputLowerFrom(ctx, output)
	var missing []string
	for _, k := range keywords {
		if !strings.Contains(lower, strings.ToLower(k)) {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return failResult("warn", "keywords missing", map[string]any{"missing": missing})
	}
	return passResult("warn", "keywords preserved")
}

type LatencyValidator struct{}

func (LatencyValidator) Name() string { return "latency_validator" }

func (LatencyValidator) Run(ctx context.Context, _ string, params map[string]any) Result {
	maxMs := intFromParam(params["max_ms"], 30000)
	meta := RunMetaFrom(ctx)
	if meta.LatencyMs == 0 {
		return skipResult("latency not measured")
	}
	if meta.LatencyMs > maxMs {
		return failResult("warn", "latency exceeded", map[string]any{"ms": meta.LatencyMs, "max": maxMs})
	}
	return passResult("warn", "latency ok")
}

type TokenCountValidator struct{}

func (TokenCountValidator) Name() string { return "token_count_validator" }

func (TokenCountValidator) Run(ctx context.Context, output string, params map[string]any) Result {
	maxTok := intFromParam(params["max_tokens"], 8000)
	meta := RunMetaFrom(ctx)
	tokens := meta.CompletionTokens
	if tokens == 0 {
		tokens = len(strings.Fields(output))
	}
	if tokens > maxTok {
		return failResult("warn", "token count high", map[string]any{"tokens": tokens, "max": maxTok})
	}
	return passResult("warn", "token count ok")
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		return 0
	}
}
