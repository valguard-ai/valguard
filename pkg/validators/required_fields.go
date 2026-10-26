package validators

import (
	"context"
	"strings"
)

// RequiredFields ensures top-level JSON keys are present and non-empty.
type RequiredFields struct{}

func (RequiredFields) Name() string { return "required_fields" }

func (RequiredFields) Run(ctx context.Context, output string, params map[string]any) Result {
	fields, _ := params["fields"].([]any)
	if len(fields) == 0 {
		return Result{Status: "skipped", Severity: "error", Message: "no fields configured"}
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return Result{Status: "fail", Severity: "error", Message: "cannot parse JSON for required_fields"}
	}
	var missing []string
	for _, f := range fields {
		name, _ := f.(string)
		if name == "" {
			continue
		}
		if _, present := data[name]; requiredValueMissing(data[name], present) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return missingRequiredFields(missing, defaultsFromParams(params))
	}
	return Result{Status: "pass", Severity: "error", Message: "all required fields present"}
}

func requiredValueMissing(value any, present bool) bool {
	if !present || value == nil {
		return true
	}
	switch field := value.(type) {
	case ShallowField:
		if field.IsNull() {
			return true
		}
		text, ok := field.String()
		return ok && strings.TrimSpace(text) == ""
	case string:
		return strings.TrimSpace(field) == ""
	default:
		return false
	}
}

func defaultsFromParams(params map[string]any) map[string]bool {
	out := map[string]bool{}
	if params == nil {
		return out
	}
	defaults, _ := params["defaults"].(map[string]any)
	for name, value := range defaults {
		if name != "" && configuredDefault(value) {
			out[name] = true
		}
	}
	return out
}

func configuredDefault(value any) bool {
	if value == nil {
		return false
	}
	text, ok := value.(string)
	return !ok || strings.TrimSpace(text) != ""
}

func missingRequiredFields(missing []string, hasDefault map[string]bool) Result {
	parts := make([]string, 0, len(missing))
	for _, name := range missing {
		if hasDefault != nil && hasDefault[name] {
			parts = append(parts, name)
			continue
		}
		parts = append(parts, name+" (no default configured)")
	}
	return Result{
		Status:   "fail",
		Severity: "error",
		Message:  "missing required fields: " + strings.Join(parts, ", "),
		Details:  map[string]any{"missing": missing},
	}
}

func (RequiredFields) Compile(params map[string]any) CompiledRunner {
	fields := stringSliceParam(params["fields"])
	return func(ctx context.Context, output string) Result {
		if len(fields) == 0 {
			return Result{Status: "skipped", Severity: "error", Message: "no fields configured"}
		}
		data, err := parseShallowCtx(ctx, output)
		if err != nil {
			return Result{Status: "fail", Severity: "error", Message: "cannot parse JSON for required_fields"}
		}
		var missing []string
		for _, name := range fields {
			if _, present := data[name]; requiredValueMissing(data[name], present) {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			return missingRequiredFields(missing, defaultsFromParams(params))
		}
		return Result{Status: "pass", Severity: "error", Message: "all required fields present"}
	}
}
