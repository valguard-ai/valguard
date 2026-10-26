//go:build !osspublic

package validators

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Package-level regexes used in Run() hot path. Compiling once at init
// avoids ~80 KB / many allocs per invocation. Each was previously created
// per Run via regexp.MustCompile or regexp.MatchString.
var (
	phaseBStepRefRe       = regexp.MustCompile(`step\[(\d+)\]\.(\w+)`)
	phaseBBracketCiteRe   = regexp.MustCompile(`\[(\d+)\]`)
	phaseBParenYearCiteRe = regexp.MustCompile(`\([A-Za-z][^,]+,\s*\d{4}\)`)
	phaseBHTMLOnAttrRe    = regexp.MustCompile(`on[a-z]+\s*=`)
)

// wordTokenReCache memoises `[A-Za-z]{N,}` regexes keyed on N. The token
// length parameter has a small bounded range in practice (3-12).
var wordTokenReCache sync.Map // int -> *regexp.Regexp

func wordTokenRegexp(minLen int) *regexp.Regexp {
	if v, ok := wordTokenReCache.Load(minLen); ok {
		return v.(*regexp.Regexp)
	}
	re := regexp.MustCompile(fmt.Sprintf(`[A-Za-z]{%d,}`, minLen))
	actual, _ := wordTokenReCache.LoadOrStore(minLen, re)
	return actual.(*regexp.Regexp)
}

// --- input / output fidelity ---

type FactMatch struct{}

func (FactMatch) Name() string { return "fact_match" }

func (FactMatch) Run(ctx context.Context, output string, params map[string]any) Result {
	meta := RunMetaFrom(ctx)
	fields := stringSliceParam(params["fields"])
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
		lowerOut := outputLowerFrom(ctx, output)
		if !strings.Contains(output, valStr) && !strings.Contains(lowerOut, strings.ToLower(valStr)) {
			return failResult("error", "input fact value not reflected in output", map[string]any{"field": f, "value": inVal})
		}
	}
	return passResult("error", "facts match input")
}

type InputCoverage struct{}

func (InputCoverage) Name() string { return "input_coverage" }

func (InputCoverage) Run(ctx context.Context, output string, params map[string]any) Result {
	meta := RunMetaFrom(ctx)
	required := stringSliceParam(params["required_inputs"])
	if len(required) == 0 && meta.InputText != "" {
		for _, line := range strings.Split(meta.InputText, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				required = append(required, line)
			}
		}
	}
	if len(required) == 0 {
		return skipResult("required_inputs or input_text required")
	}
	lowerOut := outputLowerFrom(ctx, output)
	for _, req := range required {
		if req == "" {
			continue
		}
		if !strings.Contains(lowerOut, strings.ToLower(req)) {
			return failResult("warn", "required input not covered in output", map[string]any{"missing": req})
		}
	}
	return passResult("warn", "input covered")
}

type InputOutputDiff struct{}

func (InputOutputDiff) Name() string { return "input_output_diff" }

func (InputOutputDiff) Run(ctx context.Context, output string, params map[string]any) Result {
	cd := corpusDataFrom(ctx)
	if cd.text == "" {
		return skipResult("input_text or rag sources required")
	}
	for _, n := range extractNumbers(output) {
		if _, ok := cd.numSet[n]; !ok {
			return failResult("error", "number in output not found in input/sources", map[string]any{"number": n})
		}
	}
	minLen := 5
	if v, ok := params["min_token_len"].(float64); ok && v > 0 {
		minLen = int(v)
	}
	wordRe := wordTokenRegexp(minLen)
	for _, tok := range wordRe.FindAllString(output, -1) {
		key := strings.ToLower(tok)
		if _, ok := cd.tokenSet[key]; !ok {
			return failResult("error", "term in output not found in input/sources", map[string]any{"term": tok})
		}
	}
	return passResult("error", "no novel facts detected")
}

// --- multi-step ---

type StepConsistency struct{}

func (StepConsistency) Name() string { return "step_consistency" }

func (StepConsistency) Run(ctx context.Context, output string, params map[string]any) Result {
	field := "steps"
	if s, ok := params["steps_field"].(string); ok && s != "" {
		field = s
	}
	steps, err := parseJSONArrayField(ctx, output, field)
	if err != nil {
		return skipResult("steps array required: " + err.Error())
	}
	if len(steps) == 0 {
		return failResult("error", "empty steps array", nil)
	}
	var keys []string
	keySet := make(map[string]struct{})
	for k := range steps[0] {
		keys = append(keys, k)
		keySet[k] = struct{}{}
	}
	for i, step := range steps {
		for _, k := range keys {
			if _, ok := step[k]; !ok {
				return failResult("error", "inconsistent step keys", map[string]any{"step": i, "missing_key": k})
			}
		}
		for k := range step {
			if _, ok := keySet[k]; !ok {
				return failResult("error", "inconsistent step keys", map[string]any{"step": i, "extra_key": k})
			}
		}
	}
	return passResult("error", "steps structurally consistent")
}

type RequiredStepOrder struct{}

func (RequiredStepOrder) Name() string { return "required_step_order" }

func (RequiredStepOrder) Run(ctx context.Context, output string, params map[string]any) Result {
	order := stringSliceParam(params["order"])
	if len(order) == 0 {
		return skipResult("order required")
	}
	field := "name"
	if s, ok := params["step_name_field"].(string); ok && s != "" {
		field = s
	}
	stepsField := "steps"
	if s, ok := params["steps_field"].(string); ok && s != "" {
		stepsField = s
	}
	steps, err := parseJSONArrayField(ctx, output, stepsField)
	if err != nil {
		return failResult("error", "invalid steps for order check", map[string]any{"err": err.Error()})
	}
	if len(steps) != len(order) {
		return failResult("error", "step count mismatch", map[string]any{"expected": len(order), "got": len(steps)})
	}
	for i, want := range order {
		got, ok := jsonFieldString(steps[i], field)
		if !ok || got != want {
			return failResult("error", "step order mismatch", map[string]any{"index": i, "expected": want, "got": got})
		}
	}
	return passResult("error", "step order ok")
}

type StateTransitionRules struct{}

func (StateTransitionRules) Name() string { return "state_transition_rules" }

func (StateTransitionRules) Run(ctx context.Context, output string, params map[string]any) Result {
	fromField := "from_state"
	toField := "to_state"
	if s, ok := params["from_field"].(string); ok && s != "" {
		fromField = s
	}
	if s, ok := params["to_field"].(string); ok && s != "" {
		toField = s
	}
	obj, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return skipResult("JSON output required for state transitions")
	}
	from, okFrom := jsonFieldString(obj, fromField)
	to, okTo := jsonFieldString(obj, toField)
	if !okFrom || !okTo {
		return skipResult("from/to state fields missing")
	}
	allowed := parseTransitionRules(params["transitions"])
	if len(allowed) == 0 {
		return skipResult("transitions required")
	}
	key := from + "->" + to
	if _, ok := allowed[key]; !ok {
		return failResult("error", "disallowed state transition", map[string]any{"from": from, "to": to})
	}
	return passResult("error", "state transition allowed")
}

func parseTransitionRules(v any) map[string]struct{} {
	out := make(map[string]struct{})
	arr, ok := v.([]any)
	if !ok {
		return out
	}
	for _, item := range arr {
		switch t := item.(type) {
		case string:
			out[t] = struct{}{}
		case map[string]any:
			from, _ := t["from"].(string)
			to, _ := t["to"].(string)
			if from != "" && to != "" {
				out[from+"->"+to] = struct{}{}
			}
		}
	}
	return out
}

type MultiStepNumericLogic struct{}

func (MultiStepNumericLogic) Name() string { return "multi_step_numeric_logic" }

func (MultiStepNumericLogic) Run(ctx context.Context, output string, params map[string]any) Result {
	rules := stringSliceParam(params["rules"])
	if len(rules) == 0 {
		return skipResult("rules required")
	}
	stepsField := "steps"
	if s, ok := params["steps_field"].(string); ok && s != "" {
		stepsField = s
	}
	steps, err := parseJSONArrayField(ctx, output, stepsField)
	if err != nil {
		return skipResult("steps array required")
	}
	for _, rule := range rules {
		if err := evalStepNumericRule(rule, steps); err != nil {
			return failResult("error", err.Error(), map[string]any{"rule": rule})
		}
	}
	return passResult("error", "multi-step numeric rules ok")
}

func evalStepNumericRule(rule string, steps []map[string]any) error {
	// e.g. "step[0].total + step[1].total == step[2].total" simplified: sum(field) across steps
	rule = strings.TrimSpace(rule)
	if !strings.Contains(rule, "==") {
		return fmt.Errorf("rule must contain equality (==)")
	}
	parts := strings.Split(rule, "==")
	if len(parts) != 2 {
		return fmt.Errorf("invalid rule")
	}
	left, err := evalStepExpr(strings.TrimSpace(parts[0]), steps)
	if err != nil {
		return err
	}
	right, err := evalStepExpr(strings.TrimSpace(parts[1]), steps)
	if err != nil {
		return err
	}
	if left != right {
		return fmt.Errorf("numeric rule failed: %s != %s", parts[0], parts[1])
	}
	return nil
}

func evalStepExpr(expr string, steps []map[string]any) (float64, error) {
	if strings.HasPrefix(expr, "sum(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(expr, "sum("), ")")
		idx := strings.Index(inner, ".")
		if idx < 0 {
			return 0, fmt.Errorf("sum needs step.field")
		}
		field := inner[idx+1:]
		var total float64
		for i, step := range steps {
			v, ok := step[field]
			if !ok {
				continue
			}
			f, ok := v.(float64)
			if !ok {
				return 0, fmt.Errorf("sum(%s): step[%d].%s must be numeric", inner, i, field)
			}
			total += f
		}
		return total, nil
	}
	if strings.HasPrefix(expr, "step[") {
		m := phaseBStepRefRe.FindStringSubmatch(expr)
		if len(m) != 3 {
			return 0, fmt.Errorf("unsupported expr")
		}
		i, err := strconv.Atoi(m[1])
		if err != nil || i >= len(steps) {
			return 0, fmt.Errorf("step index out of range")
		}
		v, ok := steps[i][m[2]]
		if !ok {
			return 0, fmt.Errorf("field missing")
		}
		if f, ok := v.(float64); ok {
			return f, nil
		}
	}
	return 0, fmt.Errorf("unsupported expr: %s", expr)
}

// --- RAG ---

type RAGSourcePresence struct{}

func (RAGSourcePresence) Name() string { return "rag_source_presence" }

func (RAGSourcePresence) Run(ctx context.Context, output string, _ map[string]any) Result {
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

type RAGSourceCoverage struct{}

func (RAGSourceCoverage) Name() string { return "rag_source_coverage" }

func (RAGSourceCoverage) Run(ctx context.Context, output string, params map[string]any) Result {
	meta := RunMetaFrom(ctx)
	required := stringSliceParam(params["required_source_ids"])
	if len(required) == 0 {
		required = ragSourceIDs(meta)
	}
	if len(required) == 0 {
		return skipResult("required_source_ids or rag sources required")
	}
	for _, id := range required {
		if !citationMentionsID(output, id) {
			return failResult("warn", "required source not cited", map[string]any{"source_id": id})
		}
	}
	return passResult("warn", "source coverage ok")
}

type RAGSourceMatch struct{}

func (RAGSourceMatch) Name() string { return "rag_source_match" }

func (RAGSourceMatch) Run(ctx context.Context, output string, params map[string]any) Result {
	cd := corpusDataFrom(ctx)
	if cd.text == "" {
		return skipResult("input_text or rag sources required")
	}
	minLen := 5
	if v, ok := params["min_token_len"].(float64); ok && v > 0 {
		minLen = int(v)
	}
	wordRe := wordTokenRegexp(minLen)
	for _, tok := range wordRe.FindAllString(output, -1) {
		key := strings.ToLower(tok)
		if _, ok := cd.tokenSet[key]; !ok {
			return failResult("error", "output term not grounded in sources", map[string]any{"term": tok})
		}
	}
	return passResult("error", "output grounded in sources")
}

// citationMentionsID checks whether `output` cites a given source id using
// any of three accepted citation forms:
//   - bracket: "[id]"
//   - source word: "source <id>" or "Source <id>" (case-insensitive, with
//     a word boundary after the id)
//   - parenthetical: "(source <id>)" (case-insensitive)
//
// Previously implemented with three regexp.MatchString calls per id; that
// compiled three regexes per invocation and dominated the RAG benchmark's
// allocation profile. The byte-level scanner below is allocation-free and
// roughly an order of magnitude cheaper.
func citationMentionsID(output, id string) bool {
	if id == "" {
		return false
	}
	if mentionsBracketID(output, id) {
		return true
	}
	return mentionsSourceID(output, id, false) || mentionsSourceID(output, id, true)
}

// mentionsBracketID matches "[id]".
func mentionsBracketID(output, id string) bool {
	for i := 0; i < len(output)-len(id)-1; i++ {
		if output[i] != '[' {
			continue
		}
		end := i + 1 + len(id)
		if end+1 > len(output) {
			return false
		}
		if output[end] != ']' {
			continue
		}
		if output[i+1:end] == id {
			return true
		}
	}
	return false
}

// mentionsSourceID matches "source <id>" with word-boundary after id.
// When paren is true the form is "(source <id>)".
// ASCII case-insensitive.
func mentionsSourceID(output, id string, paren bool) bool {
	prefix := "source"
	idLen := len(id)
	for i := 0; i+len(prefix)+1+idLen <= len(output); i++ {
		if paren {
			if i == 0 || output[i-1] != '(' {
				continue
			}
		}
		if !asciiEqualFold(output[i:i+len(prefix)], prefix) {
			continue
		}
		j := i + len(prefix)
		if j >= len(output) || (output[j] != ' ' && output[j] != '\t') {
			continue
		}
		// Allow multiple spaces between "source" and the id.
		for j < len(output) && (output[j] == ' ' || output[j] == '\t') {
			j++
		}
		if j+idLen > len(output) {
			return false
		}
		if output[j:j+idLen] != id {
			continue
		}
		after := j + idLen
		if paren {
			if after >= len(output) || output[after] != ')' {
				continue
			}
			return true
		}
		// Word boundary: end-of-string or non-word ASCII char.
		if after == len(output) || !isWordByte(output[after]) {
			return true
		}
	}
	return false
}

func asciiEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if asciiToLower(a[i]) != asciiToLower(b[i]) {
			return false
		}
	}
	return true
}

func isWordByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

type CitationFormatValid struct{}

func (CitationFormatValid) Name() string { return "citation_format_valid" }

// citationFormatPlan binds the list of accepted citation formats at load
// time so the hot path can iterate a small fixed slice and skip
// `stringSliceParam` per Run.
type citationFormatPlan struct {
	wantBracket   bool
	wantParenYear bool
}

func planCitationFormats(formats []string) citationFormatPlan {
	if len(formats) == 0 {
		return citationFormatPlan{wantBracket: true}
	}
	var p citationFormatPlan
	for _, f := range formats {
		switch f {
		case "bracket":
			p.wantBracket = true
		case "paren_year":
			p.wantParenYear = true
		}
	}
	if !p.wantBracket && !p.wantParenYear {
		p.wantBracket = true
	}
	return p
}

func (CitationFormatValid) Run(ctx context.Context, output string, params map[string]any) Result {
	return citationFormatValidRun(ctx, output, planCitationFormats(stringSliceParam(params["formats"])))
}

func (CitationFormatValid) Compile(params map[string]any) CompiledRunner {
	plan := planCitationFormats(stringSliceParam(params["formats"]))
	return func(ctx context.Context, output string) Result {
		return citationFormatValidRun(ctx, output, plan)
	}
}

func citationFormatValidRun(ctx context.Context, output string, plan citationFormatPlan) Result {
	if plan.wantBracket {
		meta := RunMetaFrom(ctx)
		ids := meta.RAGSources
		// scanBracketCitationIDs walks output once and reports each
		// numeric [N] occurrence via callback; zero allocations.
		var badID string
		var foundBad bool
		scanBracketCitationIDs(output, func(id string) bool {
			if len(ids) == 0 {
				badID = id
				foundBad = true
				return false
			}
			for i := range ids {
				if ids[i].ID == id {
					return true
				}
			}
			badID = id
			foundBad = true
			return false
		})
		if foundBad {
			if len(ids) == 0 {
				return failResult("warn", "citation present but no RAG sources", map[string]any{"id": badID})
			}
			return failResult("warn", "unknown citation id", map[string]any{"id": badID})
		}
	}
	if plan.wantParenYear {
		if !phaseBParenYearCiteRe.MatchString(output) {
			return failResult("warn", "invalid parenthetical citation format", nil)
		}
	}
	return passResult("warn", "citation format ok")
}

// scanBracketCitationIDs walks output and invokes accept(id) for every
// `[N]` token where N is a non-empty run of digits. It stops early when
// accept returns false. The id substring aliases output (no copy).
func scanBracketCitationIDs(output string, accept func(id string) bool) {
	for i := 0; i < len(output); i++ {
		if output[i] != '[' {
			continue
		}
		j := i + 1
		start := j
		for j < len(output) && output[j] >= '0' && output[j] <= '9' {
			j++
		}
		if j == start || j >= len(output) || output[j] != ']' {
			continue
		}
		id := output[start:j]
		if !accept(id) {
			return
		}
		i = j
	}
}

// --- ops / format / compliance ---

type RetryCountValidator struct{}

func (RetryCountValidator) Name() string { return "retry_count_validator" }

func (RetryCountValidator) Run(ctx context.Context, _ string, params map[string]any) Result {
	meta := RunMetaFrom(ctx)
	maxRetries := 2
	if v, ok := params["max_retries"].(float64); ok {
		maxRetries = int(v)
		if maxRetries < 0 {
			maxRetries = 0
		}
	}
	if meta.ReaskAttempt > maxRetries {
		return failResult("error", "max retries exceeded", map[string]any{"attempt": meta.ReaskAttempt, "max": maxRetries})
	}
	return passResult("error", "retry count ok")
}

type HomoglyphDetection struct{}

func (HomoglyphDetection) Name() string { return "homoglyph_detection" }

func (HomoglyphDetection) Run(ctx context.Context, output string, params map[string]any) Result {
	fields := stringSliceParam(params["fields"])
	if len(fields) == 0 {
		if hasHomoglyphURL(output) {
			return failResult("error", "homoglyph URL detected", nil)
		}
		return passResult("error", "no homoglyphs")
	}
	obj, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return skipResult("JSON required for field homoglyph check")
	}
	for _, f := range fields {
		s, ok := jsonFieldString(obj, f)
		if !ok {
			continue
		}
		if hasHomoglyphURL(s) || mixedScriptHost(s) {
			return failResult("error", "homoglyph in field", map[string]any{"field": f})
		}
	}
	return passResult("error", "no homoglyphs in fields")
}

type HTMLNoScript struct{}

func (HTMLNoScript) Name() string { return "html_no_script" }

func (HTMLNoScript) Run(ctx context.Context, output string, _ map[string]any) Result {
	lower := outputLowerFrom(ctx, output)
	if strings.Contains(lower, "<script") || phaseBHTMLOnAttrRe.MatchString(lower) {
		return failResult("error", "unsafe HTML detected", nil)
	}
	return passResult("error", "html safe")
}

type ValidVATEU struct{}

func (ValidVATEU) Name() string { return "valid_vat_eu" }

func (ValidVATEU) Run(ctx context.Context, output string, params map[string]any) Result {
	fields := stringSliceParam(params["fields"])
	if len(fields) == 0 {
		fields = []string{"vat_number", "vat", "vat_id"}
	}
	obj, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return skipResult("JSON required for VAT validation")
	}
	for _, f := range fields {
		v, ok := jsonFieldString(obj, f)
		if !ok || strings.TrimSpace(v) == "" {
			continue
		}
		if !validEUVATNumber(v) {
			return failResult("error", "invalid EU VAT number", map[string]any{"field": f, "value": v})
		}
	}
	return passResult("error", "VAT ok")
}

type AIDisclosureRequired struct{}

func (AIDisclosureRequired) Name() string { return "ai_disclosure_required" }

func (AIDisclosureRequired) Run(ctx context.Context, output string, params map[string]any) Result {
	phrases := stringSliceParam(params["required_phrases"])
	if len(phrases) == 0 {
		phrases = []string{"AI", "automated", "bot", "assistant"}
	}
	ci := true
	if v, ok := params["case_insensitive"].(bool); ok {
		ci = v
	}
	out := output
	if ci {
		out = outputLowerFrom(ctx, output)
	}
	for _, p := range phrases {
		needle := p
		if ci {
			needle = strings.ToLower(p)
		}
		if !strings.Contains(out, needle) {
			return failResult("warn", "missing AI disclosure phrase", map[string]any{"phrase": p})
		}
	}
	return passResult("warn", "AI disclosure present")
}
