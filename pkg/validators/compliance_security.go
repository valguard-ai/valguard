//go:build !osspublic

package validators

import (
	"context"
)

type CommandInjectionPattern struct{}

func (CommandInjectionPattern) Name() string { return "command_injection_pattern" }

func (CommandInjectionPattern) Run(ctx context.Context, output string, params map[string]any) Result {
	patterns := stringSliceParam(params["patterns"])
	if len(patterns) == 0 {
		patterns = []string{`;\s*rm\s`, `\|\s*bash`, "`[^`]+`", `\$\([^)]+\)`}
	}
	if m, err := matchAnyPattern(output, patterns, false); err != nil {
		return failResult("error", "no valid patterns in params", map[string]any{"error": err.Error()})
	} else if len(m) > 0 {
		return failResult("error", "command injection pattern", map[string]any{"matched": m})
	}
	return passResult("error", "no command injection")
}

type TemplateInjection struct{}

func (TemplateInjection) Name() string { return "template_injection" }

func (TemplateInjection) Run(ctx context.Context, output string, params map[string]any) Result {
	patterns := stringSliceParam(params["patterns"])
	if len(patterns) == 0 {
		patterns = []string{`\{\{.*\}\}`, `\$\{.*\}`, `<%.*%>`}
	}
	if m, err := matchAnyPattern(output, patterns, false); err != nil {
		return failResult("error", "no valid patterns in params", map[string]any{"error": err.Error()})
	} else if len(m) > 0 {
		return failResult("error", "template injection pattern", map[string]any{"matched": m})
	}
	return passResult("error", "no template injection")
}

type XXEPattern struct{}

func (XXEPattern) Name() string { return "xxe_pattern" }

func (XXEPattern) Run(ctx context.Context, output string, params map[string]any) Result {
	patterns := stringSliceParam(params["patterns"])
	if len(patterns) == 0 {
		patterns = []string{`<!ENTITY`, `SYSTEM\s+["']`}
	}
	if m, err := matchAnyPattern(output, patterns, true); err != nil {
		return failResult("error", "no valid patterns in params", map[string]any{"error": err.Error()})
	} else if len(m) > 0 {
		return failResult("error", "XXE pattern detected", map[string]any{"matched": m})
	}
	return passResult("error", "no XXE patterns")
}
