package validators

import (
	"context"
	"strings"
)

func parseJSONObjectCtx(ctx context.Context, output string) (map[string]any, error) {
	trim := trimmedOutputCtx(ctx, output)
	return parseJSONObject(trim)
}

func outputLowerFrom(ctx context.Context, fallback string) string {
	if lazy := LazyOutputFrom(ctx); lazy != "" {
		return strings.ToLower(lazy)
	}
	return strings.ToLower(fallback)
}

func parseShallowWithDupsCtx(ctx context.Context, output string) (map[string]ShallowField, []string, error) {
	trim := trimmedOutputCtx(ctx, output)
	return ScanTopLevelJSON(trim)
}

func trimmedOutputCtx(ctx context.Context, output string) string {
	trim := LazyOutputFrom(ctx)
	if trim == "" {
		trim = output
	}
	// Same document extraction as valid_format / ParseShallowFields for JSON-shape rules.
	return NormalizeJSONDocumentForValidation(trim)
}
