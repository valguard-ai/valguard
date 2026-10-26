package validators

import (
	"regexp"
	"strings"
)

var codeFenceRe = regexp.MustCompile("(?is)```(?:python|py|javascript|js|typescript|ts|sql|graphql|dockerfile|hcl|terraform|yaml|yml|json)?\\s*\\n([\\s\\S]*?)```")

// ExtractMarkdownCodeBlocks returns fenced code block bodies from markdown output.
func ExtractMarkdownCodeBlocks(output string) []string {
	return extractMarkdownCodeBlocks(output)
}

func extractMarkdownCodeBlocks(output string) []string {
	matches := codeFenceRe.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		trimmed := strings.TrimSpace(output)
		if trimmed != "" {
			return []string{trimmed}
		}
		return nil
	}
	var blocks []string
	for _, m := range matches {
		if b := strings.TrimSpace(m[1]); b != "" {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

// ExtractCodeBlocksParam honors extract_code_blocks param when set.
func ExtractCodeBlocksParam(output string, params map[string]any) []string {
	return extractCodeBlocksParam(output, params)
}

func extractCodeBlocksParam(output string, params map[string]any) []string {
	if v, ok := params["extract_code_blocks"].(bool); ok && v {
		if blocks := extractMarkdownCodeBlocks(output); len(blocks) > 0 {
			return blocks
		}
	}
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil
	}
	return []string{trimmed}
}

// BalancedDelimiters checks delimiter pairs (e.g. parens, brackets).
func BalancedDelimiters(s string, pairs ...[2]byte) bool {
	return balancedDelimiters(s, pairs...)
}

func balancedDelimiters(s string, pairs ...[2]byte) bool {
	for _, p := range pairs {
		if !balancedPair(s, p[0], p[1]) {
			return false
		}
	}
	return true
}

func balancedPair(s string, open, close byte) bool {
	depth := 0
	inSingle := false
	inDouble := false
	escape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if c == '\\' && (inSingle || inDouble) {
			escape = true
			continue
		}
		if inSingle {
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if c == '"' {
				inDouble = false
			}
			continue
		}
		if c == '\'' {
			inSingle = true
			continue
		}
		if c == '"' {
			inDouble = true
			continue
		}
		if c == open {
			depth++
		} else if c == close {
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0 && !inSingle && !inDouble
}

// StripSQLComments removes line comments starting with --.
func StripSQLComments(sql string) string {
	return stripSQLComments(sql)
}

func stripSQLComments(sql string) string {
	lines := strings.Split(sql, "\n")
	var out []string
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
