//go:build !osspublic

package validators

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// --- pattern compile helpers (from compile_helpers) ---

type compiledPatternPlan struct {
	severity   string
	regs       []*regexp.Regexp
	sources    []string
	ciFallback []string
	errNoValid bool
	errMsg     string
}

func compilePatternPlan(patterns []string, caseInsensitive bool, severity string) compiledPatternPlan {
	plan := compiledPatternPlan{severity: severity}
	valid := 0
	for _, p := range patterns {
		if p == "" {
			continue
		}
		pat := p
		if caseInsensitive {
			pat = "(?i)" + p
		}
		re, err := compileCached(pat)
		if err != nil {
			if caseInsensitive {
				plan.ciFallback = append(plan.ciFallback, p)
			}
			continue
		}
		valid++
		plan.regs = append(plan.regs, re)
		plan.sources = append(plan.sources, p)
	}
	if len(patterns) > 0 && valid == 0 {
		plan.errNoValid = true
		plan.errMsg = errNoValidPatterns.Error()
	}
	return plan
}

func (plan compiledPatternPlan) run(output string) Result {
	if plan.errNoValid {
		return failResult(plan.severity, "no valid patterns in params", map[string]any{"error": plan.errMsg})
	}
	var matched []string
	for i, re := range plan.regs {
		if re.MatchString(output) {
			matched = append(matched, plan.sources[i])
		}
	}
	if len(plan.ciFallback) > 0 {
		lower := strings.ToLower(output)
		for _, p := range plan.ciFallback {
			if strings.Contains(lower, strings.ToLower(p)) {
				matched = append(matched, p)
			}
		}
	}
	if len(matched) > 0 {
		return failResult(plan.severity, "pattern matched", map[string]any{"matched": matched})
	}
	return passResult(plan.severity, "ok")
}

func compilePatternValidator(defaults []string, defaultCI bool, params map[string]any, keys ...string) CompiledRunner {
	patterns := resolvePatternStrings(params, keys, defaults)
	ci := defaultCI
	if v, ok := params["case_insensitive"].(bool); ok {
		ci = v
	}
	sev := "error"
	if s, _ := params["_severity"].(string); s != "" {
		sev = s
	}
	plan := compilePatternPlan(patterns, ci, sev)
	return func(_ context.Context, output string) Result {
		return plan.run(output)
	}
}

func resolvePatternStrings(params map[string]any, keys []string, defaults []string) []string {
	for _, key := range keys {
		if p := stringSliceParam(params[key]); len(p) > 0 {
			return p
		}
	}
	if p := stringSliceParam(params["patterns"]); len(p) > 0 {
		return p
	}
	return defaults
}

// --- iter2 ---

func (FieldPresenceRules) Compile(params map[string]any) CompiledRunner {
	rules, ok := fieldPresenceRulesFromParams(params)
	if !ok {
		return func(_ context.Context, _ string) Result {
			return skipResult("no rules")
		}
	}
	return func(ctx context.Context, output string) Result {
		return fieldPresenceRulesRun(ctx, output, rules)
	}
}

type fieldPresenceRule struct {
	ifField   string
	thenField string
}

func fieldPresenceRulesFromParams(params map[string]any) ([]fieldPresenceRule, bool) {
	raw, _ := params["rules"].([]any)
	if len(raw) == 0 {
		return nil, false
	}
	rules := make([]fieldPresenceRule, 0, len(raw))
	for _, r := range raw {
		rule, _ := r.(map[string]any)
		if rule == nil {
			continue
		}
		ifField, _ := rule["if_field"].(string)
		thenField, _ := rule["then_field"].(string)
		if ifField == "" || thenField == "" {
			continue
		}
		rules = append(rules, fieldPresenceRule{ifField: ifField, thenField: thenField})
	}
	if len(rules) == 0 {
		return nil, false
	}
	return rules, true
}

func fieldPresenceRulesRun(ctx context.Context, output string, rules []fieldPresenceRule) Result {
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	for _, rule := range rules {
		if _, hasIf := data[rule.ifField]; hasIf {
			if _, hasThen := data[rule.thenField]; !hasThen {
				return failResult("error", "conditional field missing", map[string]any{"if": rule.ifField, "then": rule.thenField})
			}
		}
	}
	return passResult("error", "presence rules ok")
}

// --- iter3 ---

func (ValidUUID) Compile(params map[string]any) CompiledRunner {
	return compileFieldStringCheck(params, func(s string) bool {
		return uuidRe.MatchString(s)
	}, "invalid uuid")
}

func (IntegerOnly) Compile(params map[string]any) CompiledRunner {
	fields := stringSliceParam(params["fields"])
	return func(ctx context.Context, output string) Result {
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
}

func (CurrencyFormat) Compile(params map[string]any) CompiledRunner {
	fields := stringSliceParam(params["fields"])
	places := intFromParam(params["decimal_places"], 2)
	return func(ctx context.Context, output string) Result {
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
}

func (PercentageFormat) Compile(params map[string]any) CompiledRunner {
	fields := stringSliceParam(params["fields"])
	scale, _ := params["scale"].(string)
	if scale == "" {
		scale = "0-100"
	}
	return func(ctx context.Context, output string) Result {
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
}

func (NoRefusalPhrases) Compile(params map[string]any) CompiledRunner {
	return compilePatternValidator([]string{
		"I cannot", "I'm sorry, I can't", "I'm unable to", "As an AI", "As a language model",
	}, false, params, "phrases", "patterns")
}

func (NoMetaCommentary) Compile(params map[string]any) CompiledRunner {
	return compilePatternValidator([]string{
		`^Sure[!,]`, `^Here(?:'s| is) your`, `^I'll (?:help|provide|create)`, `^Let me (?:help|provide)`,
	}, false, params, "patterns")
}

func (NoSelfReference) Compile(params map[string]any) CompiledRunner {
	return compilePatternValidator([]string{
		"as an AI", "as a language model", "I'm just an AI", "I am an artificial",
	}, false, params, "patterns")
}

func (NoApologies) Compile(params map[string]any) CompiledRunner {
	return compilePatternValidator([]string{
		`^I'm sorry`, `^I apologize`, "sorry for the inconvenience",
	}, false, params, "patterns")
}


func (NoLoremIpsum) Compile(params map[string]any) CompiledRunner {
	return compilePatternValidator([]string{
		"lorem ipsum", "dolor sit amet", "consectetur adipiscing",
	}, true, params, "patterns")
}

func (XSSPattern) Compile(params map[string]any) CompiledRunner {
	return compilePatternValidator([]string{
		"<script", "javascript:", "onerror=", "onload=",
	}, true, params, "patterns")
}

func (PathTraversal) Compile(params map[string]any) CompiledRunner {
	return compilePatternValidator([]string{`\.\./`, `\.\.\\`, `%2e%2e`}, false, params, "patterns")
}

// --- phase_a ---

func compileJSONStringFieldsCheck(params map[string]any, check func(string) bool, label string) CompiledRunner {
	fields := jsonFieldsFromParams(params)
	if len(fields) == 0 {
		return func(_ context.Context, _ string) Result {
			return skipResult("no fields configured")
		}
	}
	return func(ctx context.Context, output string) Result {
		return validateJSONStringFieldsWith(ctx, output, fields, check, label)
	}
}

func validateJSONStringFieldsWith(ctx context.Context, output string, fields []string, check func(string) bool, label string) Result {
	if len(fields) == 0 {
		return skipResult("no fields configured")
	}
	data, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, f := range fields {
		val, ok := data[f]
		if !ok {
			continue
		}
		s, ok := val.(string)
		if !ok {
			bad = append(bad, f+": not a string")
			continue
		}
		if !check(s) {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return failResult("error", label, map[string]any{"invalid": bad})
	}
	return passResult("error", label+" ok")
}

func (ValidBase64) Compile(params map[string]any) CompiledRunner {
	return compileJSONStringFieldsCheck(params, func(s string) bool {
		s = strings.TrimSpace(s)
		if s == "" {
			return false
		}
		_, err := base64.StdEncoding.DecodeString(s)
		return err == nil
	}, "invalid base64")
}

func (ValidJWTStructure) Compile(params map[string]any) CompiledRunner {
	return compileJSONStringFieldsCheck(params, func(s string) bool {
		return jwtStructRe.MatchString(strings.TrimSpace(s))
	}, "invalid JWT structure")
}

func (ValidSemver) Compile(params map[string]any) CompiledRunner {
	return compileJSONStringFieldsCheck(params, func(s string) bool {
		return semverRe.MatchString(strings.TrimSpace(s))
	}, "invalid semver")
}

func (ValidColorHex) Compile(params map[string]any) CompiledRunner {
	return compileJSONStringFieldsCheck(params, func(s string) bool {
		return colorHexRe.MatchString(strings.TrimSpace(s))
	}, "invalid color hex")
}

func validIPv4OrIPv6Check(s string) bool {
	s = strings.TrimSpace(s)
	if ipv4Re.MatchString(s) {
		return true
	}
	if strings.Contains(s, ":") && ipv6SimpleRe.MatchString(s) {
		return strings.Count(s, ":") >= 2
	}
	return false
}

func (ValidIPv4OrIPv6) Compile(params map[string]any) CompiledRunner {
	return compileJSONStringFieldsCheck(params, validIPv4OrIPv6Check, "invalid IP address")
}

func (CodeBlockLanguageWhitelist) Compile(params map[string]any) CompiledRunner {
	allowed := stringSliceParam(params["allowed_languages"])
	return func(_ context.Context, output string) Result {
		if len(allowed) == 0 {
			return skipResult("no allowed_languages configured")
		}
		allow := map[string]bool{}
		for _, lang := range allowed {
			allow[strings.ToLower(lang)] = true
		}
		var bad []string
		for _, m := range phaseACodeFenceRe.FindAllStringSubmatch(output, -1) {
			lang := strings.ToLower(strings.TrimSpace(m[1]))
			if lang == "" {
				continue
			}
			if !allow[lang] {
				bad = append(bad, lang)
			}
		}
		if len(bad) > 0 {
			return failResult("warn", "code block language not allowed", map[string]any{"languages": bad})
		}
		return passResult("warn", "languages ok")
	}
}

func (InstructionCompliance) Compile(params map[string]any) CompiledRunner {
	required := stringSliceParam(params["required_phrases"])
	ci, _ := params["case_insensitive"].(bool)
	return func(_ context.Context, output string) Result {
		if len(required) == 0 {
			return skipResult("no required_phrases configured")
		}
		text := output
		if ci {
			text = strings.ToLower(output)
		}
		var missing []string
		for _, phrase := range required {
			p := phrase
			if ci {
				p = strings.ToLower(phrase)
			}
			if !strings.Contains(text, p) {
				missing = append(missing, phrase)
			}
		}
		if len(missing) > 0 {
			return failResult("error", "missing required phrases", map[string]any{"missing": missing})
		}
		return passResult("error", "instructions followed")
	}
}

func (ForbiddenContent) Compile(params map[string]any) CompiledRunner {
	forbidden := stringSliceParam(params["forbidden"])
	ci, _ := params["case_insensitive"].(bool)
	plan := compilePatternPlan(forbidden, ci, "error")
	return func(_ context.Context, output string) Result {
		if len(forbidden) == 0 {
			return skipResult("no forbidden content configured")
		}
		if plan.errNoValid {
			return failResult("error", "no valid patterns in params", map[string]any{"error": plan.errMsg})
		}
		r := plan.run(output)
		if r.Status == "fail" && r.Message == "pattern matched" {
			return failResult("error", "forbidden content matched", r.Details)
		}
		return passResult("error", "no forbidden content")
	}
}

func (SensitiveContent) Compile(params map[string]any) CompiledRunner {
	topics := stringSliceParam(params["forbidden_topics"])
	ci, _ := params["case_insensitive"].(bool)
	plan := compilePatternPlan(topics, ci, "error")
	return func(_ context.Context, output string) Result {
		if len(topics) == 0 {
			return skipResult("no forbidden_topics configured")
		}
		if plan.errNoValid {
			return failResult("error", "no valid patterns in params", map[string]any{"error": plan.errMsg})
		}
		r := plan.run(output)
		if r.Status == "fail" && r.Message == "pattern matched" {
			return failResult("error", "sensitive topic matched", r.Details)
		}
		return passResult("error", "no sensitive topics")
	}
}

func (ToneStyleRules) Compile(params map[string]any) CompiledRunner {
	start, _ := params["must_start_with"].(string)
	end, _ := params["must_end_with"].(string)
	bulletWant := -1
	if bc := params["bullet_count"]; bc != nil {
		bulletWant = intFromParam(bc, -1)
	}
	return func(_ context.Context, output string) Result {
		if start != "" {
			if !strings.HasPrefix(strings.TrimSpace(output), start) {
				return failResult("warn", "missing required opening", map[string]any{"must_start_with": start})
			}
		}
		if end != "" {
			trim := strings.TrimSpace(output)
			if !strings.HasSuffix(trim, end) {
				return failResult("warn", "missing required closing", map[string]any{"must_end_with": end})
			}
		}
		if bulletWant >= 0 {
			got := countMarkdownBullets(output)
			if got != bulletWant {
				return failResult("warn", "bullet count mismatch", map[string]any{"expected": bulletWant, "got": got})
			}
		}
		return passResult("warn", "tone/style ok")
	}
}

// --- phase_b ---

func (FactMatch) Compile(params map[string]any) CompiledRunner {
	fields := stringSliceParam(params["fields"])
	return func(ctx context.Context, output string) Result {
		meta := RunMetaFrom(ctx)
		if meta.InputText == "" {
			return skipResult("input_text required for fact_match")
		}
		if len(fields) == 0 {
			return skipResult("fields required for fact_match")
		}
		inObj, inErr := inputJSONObjectFrom(ctx)
		outObj, outErr := parseJSONObjectCtx(ctx, output)
		for _, f := range fields {
			if inErr == nil && outErr == nil {
				inVal, inOk := inObj[f]
				outVal, outOk := outObj[f]
				if !inOk {
					continue
				}
				if !outOk || fmt.Sprint(inVal) != fmt.Sprint(outVal) {
					return failResult("error", "field mismatch vs input", map[string]any{"field": f, "input": inVal, "output": outVal})
				}
				continue
			}
			if inErr != nil {
				return skipResult("cannot parse input JSON for fact_match")
			}
			inVal, inOk := inObj[f]
			if !inOk {
				continue
			}
			valStr := fmt.Sprint(inVal)
			if valStr == "" {
				continue
			}
			if !strings.Contains(output, valStr) && !strings.Contains(outputLowerFrom(ctx, output), strings.ToLower(valStr)) {
				return failResult("error", "input fact value not reflected in output", map[string]any{"field": f, "value": inVal})
			}
		}
		return passResult("error", "facts match input")
	}
}

type stateTransitionPlan struct {
	fromField string
	toField   string
	allowed   map[string]struct{}
}

func stateTransitionPlanFromParams(params map[string]any) stateTransitionPlan {
	plan := stateTransitionPlan{
		fromField: "from_state",
		toField:   "to_state",
		allowed:   parseTransitionRules(params["transitions"]),
	}
	if s, ok := params["from_field"].(string); ok && s != "" {
		plan.fromField = s
	}
	if s, ok := params["to_field"].(string); ok && s != "" {
		plan.toField = s
	}
	return plan
}

func (StateTransitionRules) Compile(params map[string]any) CompiledRunner {
	plan := stateTransitionPlanFromParams(params)
	return func(ctx context.Context, output string) Result {
		obj, err := parseJSONObjectCtx(ctx, output)
		if err != nil {
			return skipResult("JSON output required for state transitions")
		}
		from, okFrom := jsonFieldString(obj, plan.fromField)
		to, okTo := jsonFieldString(obj, plan.toField)
		if !okFrom || !okTo {
			return skipResult("from/to state fields missing")
		}
		if len(plan.allowed) == 0 {
			return skipResult("transitions required")
		}
		key := from + "->" + to
		if _, ok := plan.allowed[key]; !ok {
			return failResult("error", "disallowed state transition", map[string]any{"from": from, "to": to})
		}
		return passResult("error", "state transition allowed")
	}
}

func (RAGSourcePresence) Compile(_ map[string]any) CompiledRunner {
	return func(ctx context.Context, output string) Result {
		meta := RunMetaFrom(ctx)
		ids := ragSourceIDs(meta)
		if len(ids) == 0 {
			return skipResult("rag sources required (X-VG-Sources)")
		}
		for _, id := range ids {
			if !citationMentionsID(output, id) {
				return failResult("error", "output missing source citation", map[string]any{"source_id": id})
			}
		}
		return passResult("error", "all sources cited")
	}
}
