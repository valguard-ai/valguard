package validators

import (
	"context"
	"strings"
)

type NoMarkdownFencedWrapper struct{}

func (NoMarkdownFencedWrapper) Name() string { return "no_markdown_fenced_wrapper" }

func (NoMarkdownFencedWrapper) Run(ctx context.Context, output string, _ map[string]any) Result {
	trim := strings.TrimSpace(output)
	if strings.HasPrefix(trim, "```") {
		return failResult("error", "output wrapped in markdown code fence", nil)
	}
	if strings.HasPrefix(trim, "#") {
		return failResult("error", "output starts with markdown heading", nil)
	}
	return passResult("error", "no markdown wrapper")
}

type CodeBlockBalanced struct{}

func (CodeBlockBalanced) Name() string { return "code_block_balanced" }

func (CodeBlockBalanced) Run(ctx context.Context, output string, _ map[string]any) Result {
	count := strings.Count(output, "```")
	if count%2 != 0 {
		return failResult("error", "unbalanced code fences", map[string]any{"count": count})
	}
	return passResult("error", "fences balanced")
}
