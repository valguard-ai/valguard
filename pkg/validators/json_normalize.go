package validators

import (
	"encoding/json"
	"strings"

	gojson "github.com/goccy/go-json"
)

// NormalizeJSONDocumentForValidation strips BOM/whitespace, one markdown JSON fence,
// optional preamble, and trailing non-JSON text so valid_format accepts common LLM output.
func NormalizeJSONDocumentForValidation(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "\uFEFF")
	if stripped, ok := stripValidatedJSONFence(content); ok {
		content = stripped
	}
	if lead, ok := extractLeadingJSONValue(content); ok {
		content = lead
	}
	if trimmed, ok := trimTrailingJunkAfterJSON(content); ok {
		return trimmed
	}
	return content
}

func stripValidatedJSONFence(content string) (string, bool) {
	inner := UnwrapOneCodeFence(content)
	if inner == strings.TrimSpace(content) {
		return content, false
	}
	return strings.TrimSpace(inner), true
}

func extractLeadingJSONValue(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	idx := firstJSONStartIndex(s)
	if idx <= 0 {
		return s, false
	}
	cand := strings.TrimSpace(s[idx:])
	if trimmed, ok := trimTrailingJunkAfterJSON(cand); ok {
		return trimmed, true
	}
	var v any
	if err := gojson.Unmarshal([]byte(cand), &v); err != nil {
		return s, false
	}
	return cand, true
}

func firstJSONStartIndex(s string) int {
	obj := strings.Index(s, "{")
	arr := strings.Index(s, "[")
	switch {
	case obj >= 0 && arr >= 0:
		if obj < arr {
			return obj
		}
		return arr
	case obj >= 0:
		return obj
	case arr >= 0:
		return arr
	default:
		return -1
	}
}

func trimTrailingJunkAfterJSON(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return s, false
	}
	dec := json.NewDecoder(strings.NewReader(s))
	var v any
	if err := dec.Decode(&v); err != nil {
		return s, false
	}
	offset := int(dec.InputOffset())
	if offset <= 0 || offset > len(s) {
		return s, false
	}
	rest := strings.TrimSpace(s[offset:])
	if rest == "" {
		return s, false
	}
	return strings.TrimSpace(s[:offset]), true
}
