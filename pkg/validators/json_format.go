package validators

import (
	"context"

	gojson "github.com/goccy/go-json"
)

// ValidateJSONString checks whether s is well-formed JSON (RFC 8259).
func ValidateJSONString(s string) error {
	if err := validateJSONFast(s); err == nil {
		return nil
	}
	normalized := NormalizeJSONDocumentForValidation(s)
	if normalized != s {
		if err := validateJSONFast(normalized); err == nil {
			return nil
		}
		s = normalized
	}
	var v any
	return gojson.Unmarshal([]byte(s), &v)
}

// ParseJSONDocument parses a JSON document into a generic value.
func ParseJSONDocument(raw string) (any, error) {
	var v any
	if err := gojson.Unmarshal([]byte(raw), &v); err != nil {
		return nil, err
	}
	return v, nil
}

func validateJSONFormatCtx(ctx context.Context, output string) error {
	trim := LazyOutputFrom(ctx)
	if trim == "" {
		trim = output
	}
	// Align with ParseShallowFields: one extracted document for all JSON-shape rules.
	trim = NormalizeJSONDocumentForValidation(trim)
	return ValidateJSONString(trim)
}
