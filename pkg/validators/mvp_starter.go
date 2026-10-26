package validators

import (
	"context"
	"encoding/json"
	"math"
	"regexp"
	"strings"
)

type JSONMaxDepth struct{}

func (JSONMaxDepth) Name() string { return "json_max_depth" }

func (JSONMaxDepth) Run(_ context.Context, output string, params map[string]any) Result {
	maxDepth := 32
	if v, ok := intParam(params["max_depth"], 0); ok && v > 0 {
		maxDepth = v
	}
	var data any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &data); err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	if depth := jsonValueDepth(data); depth > maxDepth {
		return failResult("error", "JSON nesting too deep", map[string]any{"depth": depth, "max_depth": maxDepth})
	}
	return passResult("error", "ok")
}

func jsonValueDepth(v any) int {
	switch x := v.(type) {
	case map[string]any:
		max := 0
		for _, child := range x {
			if d := jsonValueDepth(child); d > max {
				max = d
			}
		}
		return max + 1
	case []any:
		max := 0
		for _, child := range x {
			if d := jsonValueDepth(child); d > max {
				max = d
			}
		}
		return max + 1
	default:
		return 1
	}
}

type AmountRequiresCurrency struct{}

func (AmountRequiresCurrency) Name() string { return "amount_requires_currency" }

func (AmountRequiresCurrency) Run(ctx context.Context, output string, params map[string]any) Result {
	amountFields := stringSliceParam(params["amount_fields"])
	if len(amountFields) == 0 {
		amountFields = []string{"amount", "total", "net_amount", "gross_amount", "tax_amount"}
	}
	currencyField, _ := params["currency_field"].(string)
	if currencyField == "" {
		currencyField = "currency"
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	hasAmount := false
	for _, f := range amountFields {
		if v, ok := data[f]; ok && v.Kind() == 'n' {
			hasAmount = true
			break
		}
	}
	if !hasAmount {
		return passResult("error", "no amount fields present")
	}
	cur, ok := data[currencyField]
	if !ok {
		return failResult("error", "amount present but currency missing", map[string]any{"currency_field": currencyField})
	}
	s, ok := cur.String()
	if !ok || !isISO4217(s) {
		return failResult("error", "invalid or missing currency code", map[string]any{"currency_field": currencyField})
	}
	return passResult("error", "ok")
}

var defaultStackTracePatterns = []string{
	`(?i)traceback \(most recent call last\)`,
	`(?i)exception in thread`,
	`(?i)\bpanic:`,
	`(?i)goroutine \d+`,
	`\bat com\.[a-zA-Z0-9_.]+`,
	`\bat org\.[a-zA-Z0-9_.]+`,
	`(?i)stack trace:`,
	`(?i)caused by:`,
	`/var/www/`,
	`C:\\Users\\`,
}

type NoStackTraceLeak struct{}

func (NoStackTraceLeak) Name() string { return "no_stack_trace_leak" }

func (NoStackTraceLeak) Run(_ context.Context, output string, params map[string]any) Result {
	custom := stringSliceParam(params["patterns"])
	patterns := append([]string{}, defaultStackTracePatterns...)
	patterns = append(patterns, custom...)
	plan := compiledPatternPlan{severity: "error"}
	for _, p := range patterns {
		if p == "" {
			continue
		}
		re, err := compileCached(p)
		if err != nil {
			continue
		}
		plan.regs = append(plan.regs, re)
		plan.sources = append(plan.sources, p)
	}
	if len(plan.regs) == 0 {
		return skipResult("no patterns configured")
	}
	return plan.run(output)
}

type LineItemMathValid struct{}

func (LineItemMathValid) Name() string { return "line_item_math_valid" }

func (LineItemMathValid) Run(_ context.Context, output string, params map[string]any) Result {
	field, _ := params["line_items_field"].(string)
	if field == "" {
		field = "line_items"
	}
	qtyKey, _ := params["quantity_field"].(string)
	if qtyKey == "" {
		qtyKey = "quantity"
	}
	unitKey, _ := params["unit_price_field"].(string)
	if unitKey == "" {
		unitKey = "unit_price"
	}
	totalKey, _ := params["line_total_field"].(string)
	if totalKey == "" {
		totalKey = "line_total"
	}
	tol := 0.02
	if v, ok := params["tolerance"].(float64); ok && v >= 0 {
		tol = v
	}
	data, err := parseJSONObject(output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	raw, ok := data[field].([]any)
	if !ok || len(raw) == 0 {
		return passResult("error", "no line items")
	}
	var bad []int
	for i, item := range raw {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		qty, okQ := numberFromAny(row[qtyKey])
		unit, okU := numberFromAny(row[unitKey])
		total, okT := numberFromAny(row[totalKey])
		if !okQ || !okU || !okT {
			continue
		}
		expected := qty * unit
		if math.Abs(expected-total) > tol {
			bad = append(bad, i)
		}
	}
	if len(bad) > 0 {
		return failResult("error", "line item math mismatch", map[string]any{"rows": bad})
	}
	return passResult("error", "ok")
}

func numberFromAny(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case json.Number:
		n, err := x.Float64()
		return n, err == nil
	default:
		return 0, false
	}
}

func intParam(v any, def int) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			return def, false
		}
		return int(n), true
	default:
		if def > 0 {
			return def, true
		}
		return 0, false
	}
}

var maskedCardPattern = regexp.MustCompile(`(?i)(\*{4}|x{4}|#{4}|•{4})[\s\-]*(\*{4}|x{4}|#{4}|•{4})[\s\-]*(\*{4}|x{4}|#{4}|•{4})[\s\-]*(\*{4}|x{4}|#{4}|•{4})[\s\-]*\d{4}`)
var fullCardDigits = regexp.MustCompile(`\b\d{13,19}\b`)
var spacedPAN = regexp.MustCompile(`\b\d{4}(?:[ -]\d{4}){3}\b`)

type MaskedCardOnly struct{}

func (MaskedCardOnly) Name() string { return "masked_card_only" }

func (MaskedCardOnly) Run(_ context.Context, output string, _ map[string]any) Result {
	stripped := output
	for _, m := range maskedCardPattern.FindAllString(output, -1) {
		stripped = strings.ReplaceAll(stripped, m, strings.Repeat(" ", len(m)))
	}
	for _, m := range fullCardDigits.FindAllString(stripped, -1) {
		if luhnValid(m) {
			return failResult("error", "full card number not allowed; use masked last-4 only", map[string]any{"match": m[:6] + "…"})
		}
	}
	for _, m := range spacedPAN.FindAllString(output, -1) {
		d := digits(m)
		if luhnValid(d) {
			return failResult("error", "full card number not allowed; use masked last-4 only", map[string]any{"match": d[:6] + "…"})
		}
	}
	for _, m := range fullCardDigits.FindAllString(stripped, -1) {
		if len(m) >= 15 {
			return failResult("error", "full card number not allowed; use masked last-4 only", nil)
		}
	}
	return passResult("error", "ok")
}
