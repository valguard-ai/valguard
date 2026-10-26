package validators

import (
	"context"
	"fmt"
)

type TypeChecking struct{}

func (TypeChecking) Name() string { return "type_checking" }

func (TypeChecking) Run(ctx context.Context, output string, params map[string]any) Result {
	pairs, ok := typeCheckPairsFromParams(params)
	if !ok {
		return skipResult("no types configured")
	}
	return typeCheckRun(ctx, output, pairs)
}

type typeCheckPair struct {
	field, expected string
}

func typeCheckPairsFromParams(params map[string]any) ([]typeCheckPair, bool) {
	types, _ := params["types"].(map[string]any)
	if len(types) == 0 {
		return nil, false
	}
	pairs := make([]typeCheckPair, 0, len(types))
	for field, expected := range types {
		exp, _ := expected.(string)
		pairs = append(pairs, typeCheckPair{field: field, expected: exp})
	}
	return pairs, true
}

func typeCheckRun(ctx context.Context, output string, pairs []typeCheckPair) Result {
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, p := range pairs {
		val, ok := data[p.field]
		if !ok {
			continue
		}
		if !shallowMatchesType(val, p.expected) {
			bad = append(bad, p.field+": expected "+p.expected)
		}
	}
	if len(bad) > 0 {
		return failResult("error", "type mismatch", map[string]any{"errors": bad})
	}
	return passResult("error", "types ok")
}

func (TypeChecking) Compile(params map[string]any) CompiledRunner {
	pairs, ok := typeCheckPairsFromParams(params)
	if !ok {
		return func(_ context.Context, _ string) Result {
			return skipResult("no types configured")
		}
	}
	return func(ctx context.Context, output string) Result {
		return typeCheckRun(ctx, output, pairs)
	}
}

func shallowMatchesType(v ShallowField, expected string) bool {
	switch expected {
	case "string":
		return v.Kind() == 's'
	case "number":
		return v.Kind() == 'n'
	case "boolean":
		return v.Kind() == 'b'
	case "array":
		return v.Kind() == 'a'
	case "object":
		return v.Kind() == 'o'
	default:
		return false
	}
}

type EnumValidator struct{}

func (EnumValidator) Name() string { return "enum_validator" }

func (EnumValidator) Run(ctx context.Context, output string, params map[string]any) Result {
	enums, _ := params["enums"].(map[string]any)
	if len(enums) == 0 {
		return skipResult("no enums configured")
	}
	data, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for field, allowed := range enums {
		val, ok := data[field]
		if !ok {
			continue
		}
		s := fmt.Sprint(val)
		list, _ := allowed.([]any)
		found := false
		for _, a := range list {
			if fmt.Sprint(a) == s {
				found = true
				break
			}
		}
		if !found {
			bad = append(bad, field+": "+s)
		}
	}
	if len(bad) > 0 {
		return failResult("error", "enum violation", map[string]any{"fields": bad})
	}
	return passResult("error", "enums ok")
}

func (EnumValidator) Compile(params map[string]any) CompiledRunner {
	fields, ok := enumFieldsFromParams(params)
	if !ok {
		return func(_ context.Context, _ string) Result {
			return skipResult("no enums configured")
		}
	}
	return func(ctx context.Context, output string) Result {
		return enumValidatorRun(ctx, output, fields)
	}
}

type enumField struct {
	field   string
	allowed []string
}

func enumFieldsFromParams(params map[string]any) ([]enumField, bool) {
	enums, _ := params["enums"].(map[string]any)
	if len(enums) == 0 {
		return nil, false
	}
	fields := make([]enumField, 0, len(enums))
	for field, allowed := range enums {
		list, _ := allowed.([]any)
		strs := make([]string, 0, len(list))
		for _, a := range list {
			strs = append(strs, fmt.Sprint(a))
		}
		fields = append(fields, enumField{field: field, allowed: strs})
	}
	return fields, true
}

func enumValidatorRun(ctx context.Context, output string, fields []enumField) Result {
	data, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, ef := range fields {
		val, ok := data[ef.field]
		if !ok {
			continue
		}
		s := fmt.Sprint(val)
		found := false
		for _, a := range ef.allowed {
			if a == s {
				found = true
				break
			}
		}
		if !found {
			bad = append(bad, ef.field+": "+s)
		}
	}
	if len(bad) > 0 {
		return failResult("error", "enum violation", map[string]any{"fields": bad})
	}
	return passResult("error", "enums ok")
}

type ListLength struct{}

func (ListLength) Name() string { return "list_length" }

func (ListLength) Run(ctx context.Context, output string, params map[string]any) Result {
	field, _ := params["field"].(string)
	minN := intFromParam(params["min"], 0)
	maxN := intFromParam(params["max"], 9999)
	if field == "" {
		return skipResult("field required for list_length")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	v, ok := data[field]
	if !ok {
		return skipResult("field not present")
	}
	n, ok := v.ArrayLen()
	if !ok {
		return failResult("error", "field is not an array", map[string]any{"field": field})
	}
	if n < minN || n > maxN {
		return failResult("error", "list length out of range", map[string]any{"length": n, "min": minN, "max": maxN})
	}
	return passResult("error", "list length ok")
}

func (ListLength) Compile(params map[string]any) CompiledRunner {
	field, _ := params["field"].(string)
	minN := intFromParam(params["min"], 0)
	maxN := intFromParam(params["max"], 9999)
	return func(ctx context.Context, output string) Result {
		if field == "" {
			return skipResult("field required for list_length")
		}
		data, err := parseShallowCtx(ctx, output)
		if err != nil {
			return failResult("error", "cannot parse JSON", nil)
		}
		v, ok := data[field]
		if !ok {
			return skipResult("field not present")
		}
		n, ok := v.ArrayLen()
		if !ok {
			return failResult("error", "field is not an array", map[string]any{"field": field})
		}
		if n < minN || n > maxN {
			return failResult("error", "list length out of range", map[string]any{"field": field, "len": n, "min": minN, "max": maxN})
		}
		return passResult("error", "list length ok")
	}
}
