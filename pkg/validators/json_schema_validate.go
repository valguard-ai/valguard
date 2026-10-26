package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ValidateSimpleJSONSchema validates data against a simplified JSON Schema subset.
func ValidateSimpleJSONSchema(data any, schema map[string]any) error {
	return validateSimpleJSONSchema(data, schema)
}

func validateSimpleJSONSchema(data any, schema map[string]any) error {
	st, _ := schema["type"].(string)
	if st == "" {
		switch {
		case schema["properties"] != nil, schema["required"] != nil:
			st = "object"
		case schema["items"] != nil:
			st = "array"
		default:
			return fmt.Errorf("schema type required")
		}
	}
	switch st {
	case "object":
		obj, ok := data.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object")
		}
		if req, ok := schema["required"].([]any); ok {
			for _, r := range req {
				if name, ok := r.(string); ok {
					if _, ok := obj[name]; !ok {
						return fmt.Errorf("missing required field %s", name)
					}
				}
			}
		}
		if props, ok := schema["properties"].(map[string]any); ok {
			for k, ps := range props {
				if v, ok := obj[k]; ok {
					if pm, ok := ps.(map[string]any); ok {
						if err := validateSimpleJSONSchema(v, pm); err != nil {
							return fmt.Errorf("%s: %w", k, err)
						}
					}
				}
			}
		}
	case "array":
		arr, ok := data.([]any)
		if !ok {
			return fmt.Errorf("expected array")
		}
		if itemSchema, ok := schema["items"].(map[string]any); ok {
			for i, el := range arr {
				if err := validateSimpleJSONSchema(el, itemSchema); err != nil {
					return fmt.Errorf("[%d]: %w", i, err)
				}
			}
		}
	case "string":
		if _, ok := data.(string); !ok {
			return fmt.Errorf("expected string")
		}
	case "number", "integer":
		switch data.(type) {
		case float64, int, int64:
		default:
			return fmt.Errorf("expected number")
		}
	case "boolean":
		if _, ok := data.(bool); !ok {
			return fmt.Errorf("expected boolean")
		}
	}
	return nil
}

type JSONSchemaValidate struct{}

func (JSONSchemaValidate) Name() string { return "json_schema_validate" }

func (JSONSchemaValidate) Run(ctx context.Context, output string, params map[string]any) Result {
	return jsonSchemaValidateRun(ctx, output, params)
}

func (JSONSchemaValidate) Compile(params map[string]any) CompiledRunner {
	schema, ok := params["schema"].(map[string]any)
	if !ok || len(schema) == 0 {
		return func(_ context.Context, _ string) Result {
			return skipResult("schema required")
		}
	}
	return cachedJSONSchemaRunner(schema)
}

func jsonSchemaValidateRun(ctx context.Context, output string, params map[string]any) Result {
	schema, ok := params["schema"].(map[string]any)
	if !ok || len(schema) == 0 {
		return skipResult("schema required")
	}
	return jsonSchemaValidateRunWithSchema(ctx, output, schema)
}

func jsonSchemaValidateRunWithSchema(ctx context.Context, output string, schema map[string]any) Result {
	var data any
	if obj, err := parseJSONObjectCtx(ctx, output); err == nil {
		data = obj
	} else if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &data); err != nil {
		return failResult("error", "invalid JSON for schema validation", map[string]any{"err": err.Error()})
	}
	if err := validateSimpleJSONSchema(data, schema); err != nil {
		return failResult("error", "schema validation failed", map[string]any{"err": err.Error()})
	}
	return passResult("error", "schema valid")
}
