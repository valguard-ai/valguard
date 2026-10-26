package validators

import (
	"context"
	"strings"

	gojson "github.com/goccy/go-json"
)

// ValidateJSONString checks whether s is well-formed JSON (RFC 8259).
func ValidateJSONString(s string) error {
	if err := validateJSONFast(s); err == nil {
		return nil
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
	trim := strings.TrimSpace(output)
	if lazy := LazyOutputFrom(ctx); lazy != "" {
		trim = lazy
	} else {
		trim = UnwrapOneCodeFence(trim)
	}
	return ValidateJSONString(trim)
}
