package validators

import "strings"

// unwrapOneCodeFence removes one surrounding Markdown fence, including ```json.
// Nested fences are left alone so a fenced code sample inside JSON stays intact.
// UnwrapOneCodeFence removes one surrounding Markdown fence, including ```json.
func UnwrapOneCodeFence(s string) string { return unwrapOneCodeFence(s) }

func unwrapOneCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	rest := s[3:]
	nl := strings.IndexByte(rest, '\n')
	if nl < 0 {
		return s
	}
	lang := strings.TrimSpace(rest[:nl])
	if strings.ContainsAny(lang, " \t") {
		return s
	}
	body := rest[nl+1:]
	trimmed := strings.TrimRight(body, " \t\r\n")
	if !strings.HasSuffix(trimmed, "```") {
		return s
	}
	inner := strings.TrimSpace(trimmed[:len(trimmed)-3])
	if strings.Contains(inner, "```") {
		return s
	}
	if inner == "" {
		return s
	}
	return inner
}
