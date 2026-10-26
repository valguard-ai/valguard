package validators

import (
	"context"
	"strings"
)

type decimalPrecisionEntry struct {
	field string
	max   int
}

func decimalPrecisionEntriesFromParams(params map[string]any) []decimalPrecisionEntry {
	fields, _ := params["fields"].(map[string]any)
	if len(fields) == 0 {
		return nil
	}
	out := make([]decimalPrecisionEntry, 0, len(fields))
	for field, maxAny := range fields {
		out = append(out, decimalPrecisionEntry{field: field, max: intFromParam(maxAny, 2)})
	}
	return out
}

type DecimalPrecisionMax struct{}

func (DecimalPrecisionMax) Name() string { return "decimal_precision_max" }

func (DecimalPrecisionMax) Run(ctx context.Context, output string, params map[string]any) Result {
	entries := decimalPrecisionEntriesFromParams(params)
	return decimalPrecisionRun(ctx, output, entries)
}

func decimalPrecisionRun(ctx context.Context, output string, entries []decimalPrecisionEntry) Result {
	if len(entries) == 0 {
		return skipResult("no fields")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, e := range entries {
		val, ok := data[e.field]
		if !ok {
			continue
		}
		s := val.Raw()
		if val.Kind() == 's' {
			s, _ = val.String()
		}
		if i := strings.Index(s, "."); i >= 0 {
			decimals := len(s) - i - 1
			if decimals > e.max {
				bad = append(bad, e.field)
			}
		}
	}
	if len(bad) > 0 {
		return failResult("error", "decimal precision exceeded", map[string]any{"fields": bad})
	}
	return passResult("error", "ok")
}

func (DecimalPrecisionMax) Compile(params map[string]any) CompiledRunner {
	entries := decimalPrecisionEntriesFromParams(params)
	return func(ctx context.Context, output string) Result {
		return decimalPrecisionRun(ctx, output, entries)
	}
}

type NonNegative struct{}

func (NonNegative) Name() string { return "non_negative" }

func (NonNegative) Run(ctx context.Context, output string, params map[string]any) Result {
	fields := stringSliceParam(params["fields"])
	return nonNegativeRun(ctx, output, fields)
}

func nonNegativeRun(ctx context.Context, output string, fields []string) Result {
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
		if !ok {
			continue
		}
		n := shallowToFloat(val)
		if n < 0 {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return failResult("error", "negative values", map[string]any{"fields": bad})
	}
	return passResult("error", "ok")
}

func (NonNegative) Compile(params map[string]any) CompiledRunner {
	fields := stringSliceParam(params["fields"])
	return func(ctx context.Context, output string) Result {
		return nonNegativeRun(ctx, output, fields)
	}
}
