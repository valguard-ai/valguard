package validators

import (
	"context"
	"strings"
)

// ParseShallowFields parses top-level JSON object keys without a deep decode.
// Uses LazyOutputFrom when the proxy attachCtx set trimmed output on ctx.
func ParseShallowFields(ctx context.Context, output string) (map[string]ShallowField, error) {
	trim := LazyOutputFrom(ctx)
	if trim == "" {
		trim = UnwrapOneCodeFence(strings.TrimSpace(output))
	}
	fields, _, err := ScanTopLevelJSON(trim)
	return fields, err
}

func parseShallowCtx(ctx context.Context, output string) (map[string]ShallowField, error) {
	return ParseShallowFields(ctx, output)
}
