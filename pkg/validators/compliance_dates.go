//go:build !osspublic

package validators

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func parseDateField(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02T15:04:05",
		"02.01.2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

type DateInRange struct{}

func (DateInRange) Name() string { return "date_in_range" }

func (DateInRange) Run(ctx context.Context, output string, params map[string]any) Result {
	fields := stringSliceParam(params["fields"])
	if len(fields) == 0 {
		return skipResult("fields required")
	}
	minT := timeFromParam(params["min"])
	maxT := timeFromParam(params["max"])
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
		t, ok := parseDateField(strings.TrimSpace(fmt.Sprint(val)))
		if !ok {
			bad = append(bad, f+": unparseable")
			continue
		}
		if !minT.IsZero() && t.Before(minT) {
			bad = append(bad, f+": before min")
		}
		if !maxT.IsZero() && t.After(maxT) {
			bad = append(bad, f+": after max")
		}
	}
	if len(bad) > 0 {
		return failResult("error", "date out of range", map[string]any{"fields": bad})
	}
	return passResult("error", "dates in range")
}

func timeFromParam(v any) time.Time {
	s, _ := v.(string)
	if s == "" || s == "null" {
		return time.Time{}
	}
	t, ok := parseDateField(s)
	if ok {
		return t
	}
	return time.Time{}
}

type DateFutureOnly struct{}

func (DateFutureOnly) Name() string { return "date_future_only" }

func (DateFutureOnly) Run(ctx context.Context, output string, params map[string]any) Result {
	now := time.Now()
	return validateDateFields(ctx, output, stringSliceParam(params["fields"]), func(t time.Time) bool {
		return t.After(now)
	}, "date must be in the future")
}

type DatePastOnly struct{}

func (DatePastOnly) Name() string { return "date_past_only" }

func (DatePastOnly) Run(ctx context.Context, output string, params map[string]any) Result {
	now := time.Now()
	return validateDateFields(ctx, output, stringSliceParam(params["fields"]), func(t time.Time) bool {
		return t.Before(now)
	}, "date must be in the past")
}

type BusinessDayOnly struct{}

func (BusinessDayOnly) Name() string { return "business_day_only" }

func (BusinessDayOnly) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateDateFields(ctx, output, stringSliceParam(params["fields"]), func(t time.Time) bool {
		wd := t.Weekday()
		return wd != time.Saturday && wd != time.Sunday
	}, "date must be a business day")
}

type TimezonePresent struct{}

func (TimezonePresent) Name() string { return "timezone_present" }

func (TimezonePresent) Run(ctx context.Context, output string, params map[string]any) Result {
	fields := stringSliceParam(params["fields"])
	if len(fields) == 0 {
		return skipResult("fields required")
	}
	data, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, f := range fields {
		s := strings.TrimSpace(fmt.Sprint(data[f]))
		if s == "" {
			continue
		}
		if !hasTimezone(s) {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return failResult("warn", "missing timezone", map[string]any{"fields": bad})
	}
	return passResult("warn", "timezone ok")
}

func hasTimezone(s string) bool {
	if strings.HasSuffix(s, "Z") {
		return true
	}
	if strings.Contains(s, "+") || strings.Count(s, "-") > 2 {
		return true
	}
	if strings.Contains(s, "/") { // IANA e.g. Europe/Warsaw
		return true
	}
	return false
}

type AgeFromBirthdateInRange struct{}

func (AgeFromBirthdateInRange) Name() string { return "age_from_birthdate_in_range" }

func (AgeFromBirthdateInRange) Run(ctx context.Context, output string, params map[string]any) Result {
	field, _ := params["birthdate_field"].(string)
	if field == "" {
		return skipResult("birthdate_field required")
	}
	minAge := int(floatFromParam(params["min_age"], 0))
	maxAge := int(floatFromParam(params["max_age"], 120))
	data, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	t, ok := parseDateField(strings.TrimSpace(fmt.Sprint(data[field])))
	if !ok {
		return failResult("error", "invalid birthdate", nil)
	}
	age := int(time.Since(t).Hours() / 24 / 365)
	if age < minAge || age > maxAge {
		return failResult("error", "age out of range", map[string]any{"age": age, "min": minAge, "max": maxAge})
	}
	return passResult("error", "age ok")
}

func validateDateFields(ctx context.Context, output string, fields []string, fn func(time.Time) bool, label string) Result {
	if len(fields) == 0 {
		return skipResult("fields required")
	}
	data, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, f := range fields {
		t, ok := parseDateField(strings.TrimSpace(fmt.Sprint(data[f])))
		if !ok {
			bad = append(bad, f+": unparseable")
			continue
		}
		if !fn(t) {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return failResult("error", label, map[string]any{"fields": bad})
	}
	return passResult("error", "ok")
}
