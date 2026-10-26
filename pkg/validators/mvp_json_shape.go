package validators

import (
	"context"
	"strings"
)

type JSONNoExtraFields struct{}

func (JSONNoExtraFields) Name() string { return "json_no_extra_fields" }

func (JSONNoExtraFields) Run(ctx context.Context, output string, params map[string]any) Result {
	allowed := stringSliceParam(params["allowed_fields"])
	if len(allowed) == 0 {
		if schema, ok := params["schema"].(map[string]any); ok {
			if props, ok := schema["properties"].(map[string]any); ok {
				for k := range props {
					allowed = append(allowed, k)
				}
			}
		}
	}
	if len(allowed) == 0 {
		return skipResult("no allowed_fields configured")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	allowSet := map[string]struct{}{}
	for _, k := range allowed {
		allowSet[k] = struct{}{}
	}
	var extra []string
	for k := range data {
		if _, ok := allowSet[k]; !ok {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		return failResult("error", "extra JSON fields", map[string]any{"extra": extra})
	}
	return passResult("error", "no extra fields")
}

func (JSONNoExtraFields) Compile(params map[string]any) CompiledRunner {
	allowed := jsonNoExtraAllowedFromParams(params)
	return func(ctx context.Context, output string) Result {
		return jsonNoExtraFieldsRun(ctx, output, allowed)
	}
}

func jsonNoExtraAllowedFromParams(params map[string]any) []string {
	allowed := stringSliceParam(params["allowed_fields"])
	if len(allowed) == 0 {
		if schema, ok := params["schema"].(map[string]any); ok {
			if props, ok := schema["properties"].(map[string]any); ok {
				for k := range props {
					allowed = append(allowed, k)
				}
			}
		}
	}
	return allowed
}

func jsonNoExtraFieldsRun(ctx context.Context, output string, allowed []string) Result {
	if len(allowed) == 0 {
		return skipResult("no allowed_fields configured")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	allowSet := map[string]struct{}{}
	for _, k := range allowed {
		allowSet[k] = struct{}{}
	}
	var extra []string
	for k := range data {
		if _, ok := allowSet[k]; !ok {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		return failResult("error", "extra JSON fields", map[string]any{"extra": extra})
	}
	return passResult("error", "no extra fields")
}

type OutputShape struct{}

func (OutputShape) Name() string { return "output_shape" }

func (OutputShape) Run(ctx context.Context, output string, params map[string]any) Result {
	shape, _ := params["shape"].(string)
	if shape == "" {
		shape = "json"
	}
	trim := strings.TrimSpace(output)
	switch shape {
	case "json":
		if err := validateJSONFormatCtx(ctx, output); err != nil {
			return failResult("error", "output is not valid JSON", nil)
		}
	case "list":
		if trim == "" {
			return failResult("error", "output is not a list", nil)
		}
		for _, line := range strings.Split(trim, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			if !strings.HasPrefix(strings.TrimSpace(line), "-") && !strings.HasPrefix(strings.TrimSpace(line), "*") {
				return failResult("error", "output is not a bullet list", nil)
			}
		}
	case "table":
		if !strings.Contains(trim, "|") {
			return failResult("error", "output is not a markdown table", nil)
		}
	case "markdown":
		if !looksLikeMarkdown(trim) {
			return failResult("error", "output is not markdown", nil)
		}
	default:
		return skipResult("unknown shape: " + shape)
	}
	return passResult("error", "shape ok")
}

func (OutputShape) Compile(params map[string]any) CompiledRunner {
	shape, _ := params["shape"].(string)
	if shape == "" {
		shape = "json"
	}
	return func(ctx context.Context, output string) Result {
		trim := strings.TrimSpace(output)
		switch shape {
		case "json":
			if err := validateJSONFormatCtx(ctx, output); err != nil {
				return failResult("error", "output is not valid JSON", nil)
			}
		case "list":
			if trim == "" {
				return failResult("error", "output is not a list", nil)
			}
			for _, line := range strings.Split(trim, "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				if !strings.HasPrefix(strings.TrimSpace(line), "-") && !strings.HasPrefix(strings.TrimSpace(line), "*") {
					return failResult("error", "output is not a bullet list", nil)
				}
			}
		case "table":
			if !strings.Contains(trim, "|") {
				return failResult("error", "output is not a markdown table", nil)
			}
		case "markdown":
			if !looksLikeMarkdown(trim) {
				return failResult("error", "output is not markdown", nil)
			}
		default:
			return skipResult("unknown shape: " + shape)
		}
		return passResult("error", "shape ok")
	}
}

// LooksLikeMarkdown reports basic markdown structure in plain text.
func LooksLikeMarkdown(text string) bool {
	return looksLikeMarkdown(text)
}

func looksLikeMarkdown(text string) bool {
	if strings.Contains(text, "#") || strings.Contains(text, "**") || strings.Contains(text, "```") {
		return true
	}
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "- ") || strings.HasPrefix(trim, "* ") || strings.HasPrefix(trim, "1.") {
			return true
		}
	}
	return false
}

type NoDuplicateJSONKeys struct{}

func (NoDuplicateJSONKeys) Name() string { return "no_duplicate_json_keys" }

func (NoDuplicateJSONKeys) Run(ctx context.Context, output string, _ map[string]any) Result {
	trim := trimmedOutputCtx(ctx, output)
	if !strings.HasPrefix(trim, "{") {
		return passResult("error", "not a JSON object")
	}
	_, dups, err := parseShallowWithDupsCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	if len(dups) > 0 {
		return failResult("error", "duplicate JSON keys", map[string]any{"duplicate": dups})
	}
	return passResult("error", "no duplicate keys")
}
