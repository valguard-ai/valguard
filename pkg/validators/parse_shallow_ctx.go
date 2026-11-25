package validators

import (
	"context"
)

// ParseShallowFields parses top-level JSON object keys without a deep decode.
// Uses LazyOutputFrom when the proxy attachCtx set trimmed output on ctx, then the
// same NormalizeJSONDocumentForValidation path as valid_format so preamble/fence
// wrappers do not make required_fields disagree with format checks.
func ParseShallowFields(ctx context.Context, output string) (map[string]ShallowField, error) {
	trim := LazyOutputFrom(ctx)
	if trim == "" {
		trim = output
	}
	// Shared extraction with valid_format: never invent keys; only locate the JSON doc.
	trim = NormalizeJSONDocumentForValidation(trim)
	fields, _, err := ScanTopLevelJSON(trim)
	return fields, err
}

func parseShallowCtx(ctx context.Context, output string) (map[string]ShallowField, error) {
	return ParseShallowFields(ctx, output)
}
