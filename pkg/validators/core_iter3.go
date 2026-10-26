//go:build !osspublic

package validators

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// --- anti-LLM-failure (12) ---

type NoRefusalPhrases struct{ patternValidator }

func (NoRefusalPhrases) Name() string { return "no_refusal_phrases" }

func (v NoRefusalPhrases) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	p["patterns"] = coalescePatterns(p, "phrases", []string{
		"I cannot", "I'm sorry, I can't", "I'm unable to", "As an AI", "As a language model",
	})
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

type NoMetaCommentary struct{ patternValidator }

func (NoMetaCommentary) Name() string { return "no_meta_commentary" }

func (v NoMetaCommentary) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{`^Sure[!,]`, `^Here(?:'s| is) your`, `^I'll (?:help|provide|create)`, `^Let me (?:help|provide)`}
	}
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

type NoSelfReference struct{ patternValidator }

func (NoSelfReference) Name() string { return "no_self_reference" }

func (v NoSelfReference) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{"as an AI", "as a language model", "I'm just an AI", "I am an artificial"}
	}
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

type NoApologies struct{ patternValidator }

func (NoApologies) Name() string { return "no_apologies" }

func (v NoApologies) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{`^I'm sorry`, `^I apologize`, "sorry for the inconvenience"}
	}
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

type NoHedging struct{}

func (NoHedging) Name() string { return "no_hedging" }

func (NoHedging) Run(ctx context.Context, output string, params map[string]any) Result {
	max := intFromParam(params["max_hedge_count"], 3)
	patterns := stringSliceParam(params["hedge_patterns"])
	if len(patterns) == 0 {
		patterns = []string{"might", "perhaps", "possibly", "I'm not sure", "it seems"}
	}
	count := 0
	lower := outputLowerFrom(ctx, output)
	for _, p := range patterns {
		count += strings.Count(lower, strings.ToLower(p))
	}
	if count > max {
		return failResult("warn", fmt.Sprintf("hedging count %d exceeds %d", count, max), map[string]any{"count": count})
	}
	return passResult("warn", "hedging ok")
}


type NoLoremIpsum struct{ patternValidator }

func (NoLoremIpsum) Name() string { return "no_lorem_ipsum" }

func (v NoLoremIpsum) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{"lorem ipsum", "dolor sit amet", "consectetur adipiscing"}
	}
	p["case_insensitive"] = true
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

type CompleteSentenceEnding struct{}

func (CompleteSentenceEnding) Name() string { return "complete_sentence_ending" }

func (CompleteSentenceEnding) Run(ctx context.Context, output string, params map[string]any) Result {
	trim := strings.TrimSpace(output)
	if trim == "" {
		return skipResult("empty output")
	}
	if skip, _ := params["skip_if_json"].(bool); skip && strings.HasPrefix(trim, "{") {
		return passResult("warn", "json skipped")
	}
	if skip, _ := params["skip_if_code_block"].(bool); skip && strings.Contains(trim, "```") {
		return passResult("warn", "code skipped")
	}
	endings := stringSliceParam(params["allowed_endings"])
	if len(endings) == 0 {
		endings = []string{".", "!", "?", "\"", "'"}
	}
	last := trim[len(trim)-1:]
	for _, e := range endings {
		if strings.HasSuffix(trim, e) {
			return passResult("warn", "ending ok")
		}
	}
	return failResult("warn", "output does not end with sentence punctuation", map[string]any{"last_char": last})
}

// --- repetition & markdown ---

type MarkdownLinkWellFormed struct{}

func (MarkdownLinkWellFormed) Name() string { return "markdown_link_well_formed" }

var (
	mdLinkWellRe = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`)
	mdLinkBadRe  = regexp.MustCompile(`\[[^\]]*\]\([^)]*$|\[[^\]]*$`)
	currencyFmtRe = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)
)

func (MarkdownLinkWellFormed) Run(ctx context.Context, output string, _ map[string]any) Result {
	bad := mdLinkBadRe
	if bad.MatchString(output) {
		return failResult("error", "malformed markdown link", nil)
	}
	_ = mdLinkWellRe
	return passResult("error", "links ok")
}

type BracketBalanced struct{}

func (BracketBalanced) Name() string { return "bracket_balanced" }

func (BracketBalanced) Run(ctx context.Context, output string, params map[string]any) Result {
	text := output
	if ignore, _ := params["ignore_in_code_blocks"].(bool); ignore {
		text = stripCodeBlocks(text)
	}
	for _, pair := range []struct{ open, close rune }{{'(', ')'}, {'[', ']'}, {'{', '}'}} {
		if !bracketsBalanced(text, pair.open, pair.close) {
			return failResult("error", "unbalanced brackets", map[string]any{"pair": string(pair.open) + string(pair.close)})
		}
	}
	return passResult("error", "brackets balanced")
}

func bracketsBalanced(text string, open, close rune) bool {
	depth := 0
	for _, ch := range text {
		if ch == open {
			depth++
		}
		if ch == close {
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

func stripCodeBlocks(s string) string {
	parts := strings.Split(s, "```")
	var b strings.Builder
	for i := 0; i < len(parts); i += 2 {
		if i < len(parts) {
			b.WriteString(parts[i])
		}
	}
	return b.String()
}

type NoRepeatedPhrases struct{}

func (NoRepeatedPhrases) Name() string { return "no_repeated_phrases" }

func (NoRepeatedPhrases) Run(ctx context.Context, output string, params map[string]any) Result {
	minLen := intFromParam(params["min_phrase_length"], 20)
	maxRep := intFromParam(params["max_repetitions"], 2)
	words := strings.Fields(output)
	if len(words) < 3 {
		return passResult("warn", "too short")
	}
	for size := minLen / 5; size <= len(words)/2; size++ {
		if size < 2 {
			continue
		}
		counts := map[string]int{}
		for i := 0; i+size <= len(words); i++ {
			phrase := strings.Join(words[i:i+size], " ")
			if len(phrase) < minLen {
				continue
			}
			counts[phrase]++
			if counts[phrase] > maxRep {
				return failResult("warn", "repeated phrase", map[string]any{"phrase": phrase, "count": counts[phrase]})
			}
		}
	}
	return passResult("warn", "no excessive repeats")
}

type RepetitionRatioMax struct{}

func (RepetitionRatioMax) Name() string { return "repetition_ratio_max" }

func (RepetitionRatioMax) Run(ctx context.Context, output string, params map[string]any) Result {
	n := intFromParam(params["ngram_size"], 4)
	maxRatio := floatFromParam(params["max_ratio"], 0.35)
	tokens := strings.Fields(strings.ToLower(output))
	if len(tokens) < n*2 {
		return passResult("warn", "too short")
	}
	seen := map[string]int{}
	dup := 0
	total := 0
	for i := 0; i+n <= len(tokens); i++ {
		ng := strings.Join(tokens[i:i+n], " ")
		seen[ng]++
		total++
		if seen[ng] > 1 {
			dup++
		}
	}
	ratio := float64(dup) / float64(total)
	if ratio > maxRatio {
		return failResult("warn", fmt.Sprintf("repetition ratio %.2f exceeds %.2f", ratio, maxRatio), map[string]any{"ratio": ratio})
	}
	return passResult("warn", "repetition ok")
}

// --- format ID ---

var (
	uuidRe    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	urlRe     = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	phoneRe   = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
	isoDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})?)?$`)
)

func validateJSONFields(ctx context.Context, output string, fields []string, fn func(string) bool, label string) Result {
	if len(fields) == 0 {
		return skipResult("no fields configured")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, f := range fields {
		val, ok := data[f]
		if !ok {
			continue
		}
		s := val.Display()
		if !fn(s) {
			bad = append(bad, f+": "+s)
		}
	}
	if len(bad) > 0 {
		return failResult("error", label, map[string]any{"fields": bad})
	}
	return passResult("error", "ok")
}

// compileFieldStringCheck binds the field list and label at load time so
// the hot path skips the params/stringSliceParam dance.
func compileFieldStringCheck(params map[string]any, fn func(string) bool, label string) CompiledRunner {
	fields := stringSliceParam(params["fields"])
	if len(fields) == 0 {
		return func(_ context.Context, _ string) Result {
			return skipResult("no fields configured")
		}
	}
	return func(ctx context.Context, output string) Result {
		return validateJSONFields(ctx, output, fields, fn, label)
	}
}

type ValidUUID struct{}

func (ValidUUID) Name() string { return "valid_uuid" }

func (ValidUUID) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), func(s string) bool {
		return uuidRe.MatchString(s)
	}, "invalid uuid")
}

type ValidURLRFC3986 struct{}

func (ValidURLRFC3986) Name() string { return "valid_url_rfc3986" }

func (ValidURLRFC3986) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), func(s string) bool {
		return urlRe.MatchString(s)
	}, "invalid url")
}

type ValidPhoneE164 struct{}

func (ValidPhoneE164) Name() string { return "valid_phone_e164" }

func (ValidPhoneE164) Run(ctx context.Context, output string, params map[string]any) Result {
	allowed := normalizeCountryCodes(stringSliceParam(params["allowed_countries"]))
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), func(s string) bool {
		if !phoneRe.MatchString(s) {
			return false
		}
		if len(allowed) == 0 {
			return true
		}
		return phoneMatchesAllowedCountries(s, allowed)
	}, "invalid phone or country not allowed")
}

type ValidISODate struct{}

func (ValidISODate) Name() string { return "valid_iso_date" }

func (ValidISODate) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), isoDateMatches, "invalid iso date")
}

func (ValidISODate) Compile(params map[string]any) CompiledRunner {
	return compileFieldStringCheck(params, isoDateMatches, "invalid iso date")
}

func isoDateMatches(s string) bool { return isoDateRe.MatchString(s) }

// --- numeric ---


type IntegerOnly struct{}

func (IntegerOnly) Name() string { return "integer_only" }

func (IntegerOnly) Run(ctx context.Context, output string, params map[string]any) Result {
	fields := stringSliceParam(params["fields"])
	if len(fields) == 0 {
		return skipResult("no fields")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, f := range fields {
		val, ok := data[f]
		if !ok || val.Kind() != 'n' {
			continue
		}
		n, _ := val.Number()
		if math.Mod(n, 1) != 0 {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return failResult("error", "non-integer values", map[string]any{"fields": bad})
	}
	return passResult("error", "ok")
}


type CurrencyFormat struct{}

func (CurrencyFormat) Name() string { return "currency_format" }

func (CurrencyFormat) Run(ctx context.Context, output string, params map[string]any) Result {
	fields := stringSliceParam(params["fields"])
	places := intFromParam(params["decimal_places"], 2)
	if len(fields) == 0 {
		return skipResult("no fields")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	for _, f := range fields {
		val, ok := data[f]
		if !ok {
			continue
		}
		s := val.Display()
		if !currencyFmtRe.MatchString(s) {
			return failResult("error", "invalid currency format for "+f, map[string]any{"decimal_places": places})
		}
	}
	return passResult("error", "ok")
}

type PercentageFormat struct{}

func (PercentageFormat) Name() string { return "percentage_format" }

func (PercentageFormat) Run(ctx context.Context, output string, params map[string]any) Result {
	fields := stringSliceParam(params["fields"])
	scale, _ := params["scale"].(string)
	if scale == "" {
		scale = "0-100"
	}
	if len(fields) == 0 {
		return skipResult("no fields")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	for _, f := range fields {
		val, ok := data[f]
		if !ok {
			continue
		}
		n := shallowToFloat(val)
		if scale == "0-100" && (n < 0 || n > 100) {
			return failResult("error", "percentage out of range", map[string]any{"field": f, "value": n})
		}
		if scale == "0-1" && (n < 0 || n > 1) {
			return failResult("error", "percentage out of range", map[string]any{"field": f, "value": n})
		}
	}
	return passResult("error", "ok")
}


type XSSPattern struct{ patternValidator }

func (XSSPattern) Name() string { return "xss_pattern" }

func (v XSSPattern) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{"<script", "javascript:", "onerror=", "onload="}
	}
	p["case_insensitive"] = true
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

type PathTraversal struct{ patternValidator }

func (PathTraversal) Name() string { return "path_traversal" }

func (v PathTraversal) Run(ctx context.Context, output string, params map[string]any) Result {
	p := cloneParams(params)
	if len(stringSliceParam(p["patterns"])) == 0 {
		p["patterns"] = []any{`\.\./`, `\.\.\\`, `%2e%2e`}
	}
	return patternValidator{name: v.Name()}.runPatterns(output, p, "patterns")
}

// --- meta ---

type CostPerRequestMaxUSD struct{}

func (CostPerRequestMaxUSD) Name() string { return "cost_per_request_max_usd" }

func (CostPerRequestMaxUSD) Run(ctx context.Context, output string, params map[string]any) Result {
	_ = output
	max := floatFromParam(params["max_usd"], 0.10)
	meta := RunMetaFrom(ctx)
	// rough estimate: $0.002 per 1k completion tokens
	cost := float64(meta.CompletionTokens) * 0.000002
	if cost > max {
		return failResult("warn", fmt.Sprintf("estimated cost $%.4f exceeds $%.2f", cost, max), map[string]any{"cost_usd": cost})
	}
	return passResult("warn", "cost ok")
}

type ModelWhitelistPerAgent struct{}

func (ModelWhitelistPerAgent) Name() string { return "model_whitelist_per_agent" }

func (ModelWhitelistPerAgent) Run(ctx context.Context, _ string, params map[string]any) Result {
	allowed := stringSliceParam(params["allowed_models"])
	if len(allowed) == 0 {
		return skipResult("no allowed_models configured")
	}
	meta := RunMetaFrom(ctx)
	if meta.Model == "" {
		return skipResult("no model in context")
	}
	for _, a := range allowed {
		if a == meta.Model {
			return passResult("error", "model allowed")
		}
	}
	return failResult("error", "model not in whitelist", map[string]any{"model": meta.Model})
}

func coalescePatterns(params map[string]any, key string, defaults []string) []any {
	if existing := stringSliceParam(params[key]); len(existing) > 0 {
		out := make([]any, len(existing))
		for i, s := range existing {
			out[i] = s
		}
		return out
	}
	out := make([]any, len(defaults))
	for i, s := range defaults {
		out[i] = s
	}
	return out
}
