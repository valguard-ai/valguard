//go:build !osspublic

package validators

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

// Starter v1 validators — common production gaps (JSON limits, finance cross-fields, leaks).

type JSONArrayMaxLength struct{}

func (JSONArrayMaxLength) Name() string { return "json_array_max_length" }

func (JSONArrayMaxLength) Run(ctx context.Context, output string, params map[string]any) Result {
	fields := stringSliceParam(params["fields"])
	maxLen, ok := intParam(params["max_length"], 0)
	if len(fields) == 0 || !ok || maxLen < 1 {
		return skipResult("fields or max_length not configured")
	}
	data, err := parseJSONObject(output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, f := range fields {
		arr, ok := data[f].([]any)
		if !ok {
			continue
		}
		if len(arr) > maxLen {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return failResult("error", "array too long", map[string]any{"fields": bad, "max_length": maxLen})
	}
	return passResult("error", "ok")
}

type DateOrderValid struct{}

func (DateOrderValid) Name() string { return "date_order_valid" }

func (DateOrderValid) Run(ctx context.Context, output string, params map[string]any) Result {
	pairs, ok := dateOrderPairsFromParamsMap(params)
	if !ok {
		return skipResult("pairs not configured")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, p := range pairs {
		before, okB := data[p.before]
		after, okA := data[p.after]
		if !okB || !okA {
			continue
		}
		bStr, ok1 := before.String()
		aStr, ok2 := after.String()
		if !ok1 || !ok2 {
			bad = append(bad, p.before+" / "+p.after+": not strings")
			continue
		}
		bt, err1 := parseFlexibleDate(bStr)
		at, err2 := parseFlexibleDate(aStr)
		if err1 != nil || err2 != nil {
			bad = append(bad, p.before+" / "+p.after+": invalid date")
			continue
		}
		if bt.After(at) {
			bad = append(bad, p.before+" after "+p.after)
		}
	}
	if len(bad) > 0 {
		return failResult("error", "date order violation", map[string]any{"errors": bad})
	}
	return passResult("error", "ok")
}

type dateOrderPair struct {
	before, after string
}

func dateOrderPairsFromParams(v any) ([]dateOrderPair, bool) {
	arr, _ := v.([]any)
	if len(arr) == 0 {
		return nil, false
	}
	out := make([]dateOrderPair, 0, len(arr))
	for _, item := range arr {
		m, _ := item.(map[string]any)
		if m == nil {
			continue
		}
		b, _ := m["before"].(string)
		a, _ := m["after"].(string)
		if b != "" && a != "" {
			out = append(out, dateOrderPair{before: b, after: a})
		}
	}
	return out, len(out) > 0
}

func dateOrderPairsFromParamsMap(params map[string]any) ([]dateOrderPair, bool) {
	if pairs, ok := dateOrderPairsFromParams(params["pairs"]); ok {
		return pairs, true
	}
	before := stringSliceParam(params["before_fields"])
	after := stringSliceParam(params["after_fields"])
	if len(before) == 0 || len(before) != len(after) {
		return nil, false
	}
	out := make([]dateOrderPair, 0, len(before))
	for i := range before {
		out = append(out, dateOrderPair{before: before[i], after: after[i]})
	}
	return out, true
}

func parseFlexibleDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date")
}

type EmailDomainDenylist struct{}

func (EmailDomainDenylist) Name() string { return "email_domain_denylist" }

var emailInTextRe = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@([a-zA-Z0-9.\-]+\.[a-zA-Z]{2,})`)

func (EmailDomainDenylist) Run(_ context.Context, output string, params map[string]any) Result {
	deny := stringSliceParam(params["domains"])
	if len(deny) == 0 {
		return skipResult("no domains configured")
	}
	denySet := make(map[string]struct{}, len(deny))
	for _, d := range deny {
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" {
			denySet[d] = struct{}{}
		}
	}
	var matched []string
	for _, m := range emailInTextRe.FindAllStringSubmatch(output, -1) {
		if len(m) < 2 {
			continue
		}
		domain := strings.ToLower(m[1])
		if _, ok := denySet[domain]; ok {
			matched = append(matched, domain)
		}
	}
	if len(matched) > 0 {
		return failResult("error", "denied email domain", map[string]any{"domains": matched})
	}
	return passResult("error", "ok")
}


func AllStarterV1Core() []Validator {
	return []Validator{
		JSONMaxDepth{},
		JSONArrayMaxLength{},
		AmountRequiresCurrency{},
		DateOrderValid{},
		EmailDomainDenylist{},
		NoStackTraceLeak{},
		LineItemMathValid{},
	}
}
