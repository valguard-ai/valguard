//go:build osspublic

package validators

import "context"

// validateJSONFields checks named shallow-JSON fields with fn (shared by MVP format validators).
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
