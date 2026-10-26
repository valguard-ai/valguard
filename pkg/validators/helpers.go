package validators

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

// PassResult builds a passing validator Result.
func PassResult(severity, msg string) Result {
	return Result{Status: "pass", Severity: severity, Message: msg}
}

// FailResult builds a failing validator Result.
func FailResult(severity, msg string, details map[string]any) Result {
	return Result{Status: "fail", Severity: severity, Message: msg, Details: details}
}

// SkipResult builds a skipped validator Result.
func SkipResult(msg string) Result {
	return Result{Status: "skipped", Severity: "error", Message: msg}
}

func passResult(severity, msg string) Result { return PassResult(severity, msg) }
func failResult(severity, msg string, details map[string]any) Result {
	return FailResult(severity, msg, details)
}
func skipResult(msg string) Result { return SkipResult(msg) }

// ErrNoValidPatterns is returned when custom regex params compile to nothing.
var ErrNoValidPatterns = errNoValidPatterns

// CloneParams shallow-copies validator params.
func CloneParams(params map[string]any) map[string]any {
	return cloneParams(params)
}

func cloneParams(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}

func deepCloneAny(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = deepCloneAny(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = deepCloneAny(val)
		}
		return out
	default:
		return v
	}
}

// CloneParamsDeep deep-copies validator params.
func CloneParamsDeep(params map[string]any) map[string]any {
	return cloneParamsDeep(params)
}

func cloneParamsDeep(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}
	if m, ok := deepCloneAny(params).(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// ParseJSONObject unmarshals a JSON object document.
func ParseJSONObject(output string) (map[string]any, error) {
	return parseJSONObject(output)
}

func parseJSONObject(output string) (map[string]any, error) {
	var data map[string]any
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &data)
	return data, err
}

// StringSliceParam coerces YAML/JSON array params to []string.
func StringSliceParam(v any) []string {
	return stringSliceParam(v)
}

func stringSliceParam(v any) []string {
	arr, _ := v.([]any)
	var out []string
	for _, x := range arr {
		if s, ok := x.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

var errNoValidPatterns = errors.New("no valid patterns in params")

// compileCustomRegexPatterns compiles non-empty custom pattern strings.
// Returns errNoValidPatterns when patterns were provided but none compiled.
// CompileCustomRegexPatterns compiles custom pattern strings from params.
func CompileCustomRegexPatterns(custom []string) ([]*regexp.Regexp, error) {
	return compileCustomRegexPatterns(custom)
}

func compileCustomRegexPatterns(custom []string) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(custom))
	for _, p := range custom {
		if re, err := compileCached(p); err == nil {
			out = append(out, re)
		}
	}
	if len(out) == 0 {
		return nil, errNoValidPatterns
	}
	return out, nil
}

// MatchAnyPattern returns patterns that matched output.
func MatchAnyPattern(output string, patterns []string, caseInsensitive bool) ([]string, error) {
	return matchAnyPattern(output, patterns, caseInsensitive)
}

func matchAnyPattern(output string, patterns []string, caseInsensitive bool) ([]string, error) {
	var matched []string
	validPatterns := 0
	for _, p := range patterns {
		if p == "" {
			continue
		}
		pat := p
		if caseInsensitive {
			pat = "(?i)" + p
		}
		re, err := compileCached(pat)
		if err != nil {
			if caseInsensitive && strings.Contains(strings.ToLower(output), strings.ToLower(p)) {
				matched = append(matched, p)
			}
			continue
		}
		validPatterns++
		if re.MatchString(output) {
			matched = append(matched, p)
		}
	}
	if len(patterns) > 0 && validPatterns == 0 {
		return nil, errNoValidPatterns
	}
	return matched, nil
}

