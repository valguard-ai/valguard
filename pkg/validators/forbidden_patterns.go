package validators

import (
	"context"
	"strings"
)

type ForbiddenPatterns struct{}

func (ForbiddenPatterns) Name() string { return "forbidden_patterns" }

func (ForbiddenPatterns) Run(ctx context.Context, output string, params map[string]any) Result {
	patterns, _ := params["patterns"].([]any)
	configured := false
	for _, p := range patterns {
		s, _ := p.(string)
		if s == "" {
			continue
		}
		configured = true
		if containsForbiddenPattern(output, s) {
			return Result{Status: "fail", Severity: "error", Message: forbiddenPatternMessage(s), Details: map[string]any{"matched": s}}
		}
	}
	if !configured {
		return Result{Status: "skipped", Severity: "error", Message: "no patterns configured"}
	}
	return Result{Status: "pass", Severity: "error", Message: "no forbidden patterns"}
}

type forbiddenPatternEntry struct {
	needle string
	fail   Result
}

func (ForbiddenPatterns) Compile(params map[string]any) CompiledRunner {
	patterns := stringSliceParam(params["patterns"])
	entries := make([]forbiddenPatternEntry, 0, len(patterns))
	for _, s := range patterns {
		if s == "" {
			continue
		}
		entries = append(entries, forbiddenPatternEntry{
			needle: s,
			fail: Result{
				Status:   "fail",
				Severity: "error",
				Message:  forbiddenPatternMessage(s),
				Details:  map[string]any{"matched": s},
			},
		})
	}
	if len(entries) == 0 {
		skipped := Result{Status: "skipped", Severity: "error", Message: "no patterns configured"}
		return func(_ context.Context, _ string) Result { return skipped }
	}
	pass := Result{Status: "pass", Severity: "error", Message: "no forbidden patterns"}
	return func(_ context.Context, output string) Result {
		for i := range entries {
			if containsForbiddenPattern(output, entries[i].needle) {
				return entries[i].fail
			}
		}
		return pass
	}
}

func forbiddenPatternMessage(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "forbidden pattern detected"
	}
	if len(pattern) > 80 {
		pattern = pattern[:80]
	}
	return "forbidden pattern detected: " + pattern
}

func containsForbiddenPattern(s, sub string) bool {
	if isASCIIWordPattern(sub) {
		return containsFoldASCIIWord(s, sub)
	}
	return containsFoldASCII(s, sub)
}

func isASCIIWordPattern(sub string) bool {
	if sub == "" {
		return false
	}
	for i := 0; i < len(sub); i++ {
		c := sub[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return false
	}
	return true
}

func isASCIIWordByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func containsFoldASCIIWord(s, sub string) bool {
	if len(sub) == 0 || len(sub) > len(s) {
		return len(sub) == 0
	}
	first := asciiToLower(sub[0])
	upper := asciiToUpper(sub[0])
	for i := 0; i+len(sub) <= len(s); i++ {
		if i > 0 && isASCIIWordByte(s[i-1]) {
			continue
		}
		c := s[i]
		if c != first && c != upper {
			continue
		}
		match := true
		for j := 1; j < len(sub); j++ {
			if asciiToLower(s[i+j]) != asciiToLower(sub[j]) {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		if i+len(sub) < len(s) && isASCIIWordByte(s[i+len(sub)]) {
			continue
		}
		return true
	}
	return false
}

func containsFoldASCII(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	first := asciiToLower(sub[0])
	upper := asciiToUpper(sub[0])
	for i := 0; i+len(sub) <= len(s); i++ {
		c := s[i]
		if c != first && c != upper {
			continue
		}
		match := true
		for j := 1; j < len(sub); j++ {
			if asciiToLower(s[i+j]) != asciiToLower(sub[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// AsciiToLower folds ASCII A-Z to a-z (non-ASCII unchanged).
func AsciiToLower(c byte) byte {
	return asciiToLower(c)
}

func asciiToLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

// AsciiToUpper folds ASCII a-z to A-Z (non-ASCII unchanged).
func AsciiToUpper(c byte) byte {
	return asciiToUpper(c)
}

func asciiToUpper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - ('a' - 'A')
	}
	return c
}
