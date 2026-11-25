package validators

import (
	"context"
	"strings"
)

// AttachRunLite attaches RunMeta and a trimmed output view for OSS engine runs.
// Proxy production paths use the pooled AttachRun in apps/proxy; this helper
// avoids importing the monorepo while keeping meta validators correct.
func AttachRunLite(ctx context.Context, m RunMeta, output string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if !m.IsZero() {
		ctx = context.WithValue(ctx, MetaContextKey, m)
	}
	if output != "" {
		// Same extraction as valid_format and ParseShallowFields so lazy and direct
		// paths never disagree on which JSON document rules see.
		trimmed := NormalizeJSONDocumentForValidation(strings.TrimSpace(output))
		ctx = context.WithValue(ctx, LazyOutputContextKey, trimmed)
	}
	return ctx
}
