package validators

import (
	"context"
	"regexp"
)

type patternValidator struct {
	name string
}

func (p patternValidator) Name() string { return p.name }

func (p patternValidator) runPatterns(output string, params map[string]any, key string) Result {
	patterns := stringSliceParam(params[key])
	if len(patterns) == 0 {
		patterns = stringSliceParam(params["patterns"])
	}
	if len(patterns) == 0 {
		return skipResult("no patterns configured")
	}
	ci, _ := params["case_insensitive"].(bool)
	matched, err := matchAnyPattern(output, patterns, ci)
	if err != nil {
		return failResult(severityFrom(params), "no valid patterns in params", map[string]any{"error": err.Error()})
	}
	if len(matched) > 0 {
		return failResult(severityFrom(params), "pattern matched", map[string]any{"matched": matched})
	}
	return passResult(severityFrom(params), "ok")
}

func severityFrom(params map[string]any) string {
	if s, _ := params["_severity"].(string); s != "" {
		return s
	}
	return "error"
}

func mustCompileAll(patterns []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		out[i] = regexp.MustCompile(p)
	}
	return out
}

func matchFirstCompiled(output string, regs []*regexp.Regexp, sources []string) string {
	for i, re := range regs {
		if re.MatchString(output) {
			if i < len(sources) {
				return sources[i]
			}
			return re.String()
		}
	}
	return ""
}

// PatternValidatorRun is a thin wrapper for proxy compile bindings that still
// reference pattern-style validators by name.
func PatternValidatorRun(_ context.Context, name, output string, params map[string]any, key string) Result {
	return patternValidator{name: name}.runPatterns(output, params, key)
}
