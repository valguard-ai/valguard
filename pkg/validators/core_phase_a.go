//go:build !osspublic

package validators

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	semverRe     = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	colorHexRe   = regexp.MustCompile(`^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`)
	jwtStructRe  = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)
	ipv4Re       = regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?)$`)
	ipv6SimpleRe = regexp.MustCompile(`^[0-9A-Fa-f:]+$`)
	mdImageRe    = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	mdImageBadRe = regexp.MustCompile(`!\[[^\]]*\]\([^)]*$|!\[[^\]]*$`)
	phaseACodeFenceRe = regexp.MustCompile("(?m)^```(\\S*)")
)

func jsonFieldsFromParams(params map[string]any) []string {
	fields := stringSliceParam(params["fields"])
	if len(fields) > 0 {
		return fields
	}
	if schema, ok := params["schema"].(map[string]any); ok {
		if props, ok := schema["properties"].(map[string]any); ok {
			for k := range props {
				fields = append(fields, k)
			}
		}
	}
	return fields
}

func validateJSONStringFields(ctx context.Context, output string, params map[string]any, check func(string) bool, label string) Result {
	fields := jsonFieldsFromParams(params)
	if len(fields) == 0 {
		return skipResult("no fields configured")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var bad []string
	for _, f := range fields {
		field, ok := data[f]
		if !ok {
			continue
		}
		if field.Kind() != 's' {
			bad = append(bad, f+": not a string")
			continue
		}
		s, ok := field.String()
		if !ok {
			bad = append(bad, f+": not a string")
			continue
		}
		if !check(s) {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return failResult("error", label, map[string]any{"invalid": bad})
	}
	return passResult("error", label+" ok")
}

type ValidBase64 struct{}

func (ValidBase64) Name() string { return "valid_base64" }

func (ValidBase64) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONStringFields(ctx, output, params, func(s string) bool {
		s = strings.TrimSpace(s)
		if s == "" {
			return false
		}
		_, err := base64.StdEncoding.DecodeString(s)
		return err == nil
	}, "invalid base64")
}

type ValidJWTStructure struct{}

func (ValidJWTStructure) Name() string { return "valid_jwt_structure" }

func (ValidJWTStructure) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONStringFields(ctx, output, params, func(s string) bool {
		return jwtStructRe.MatchString(strings.TrimSpace(s))
	}, "invalid JWT structure")
}

type ValidSemver struct{}

func (ValidSemver) Name() string { return "valid_semver" }

func (ValidSemver) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONStringFields(ctx, output, params, func(s string) bool {
		return semverRe.MatchString(strings.TrimSpace(s))
	}, "invalid semver")
}

type ValidColorHex struct{}

func (ValidColorHex) Name() string { return "valid_color_hex" }

func (ValidColorHex) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONStringFields(ctx, output, params, func(s string) bool {
		return colorHexRe.MatchString(strings.TrimSpace(s))
	}, "invalid color hex")
}

type ValidIPv4OrIPv6 struct{}

func (ValidIPv4OrIPv6) Name() string { return "valid_ipv4_or_ipv6" }

func (ValidIPv4OrIPv6) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONStringFields(ctx, output, params, func(s string) bool {
		s = strings.TrimSpace(s)
		if ipv4Re.MatchString(s) {
			return true
		}
		if strings.Contains(s, ":") && ipv6SimpleRe.MatchString(s) {
			return strings.Count(s, ":") >= 2
		}
		return false
	}, "invalid IP address")
}

type MarkdownImageWellFormed struct{}

func (MarkdownImageWellFormed) Name() string { return "markdown_image_well_formed" }

func (MarkdownImageWellFormed) Run(ctx context.Context, output string, _ map[string]any) Result {
	if mdImageBadRe.MatchString(output) {
		return failResult("error", "malformed markdown image", nil)
	}
	if strings.Contains(output, "![]") && !mdImageRe.MatchString(output) {
		return failResult("error", "malformed markdown image", nil)
	}
	return passResult("error", "images ok")
}

type MarkdownTableColumnsConsistent struct{}

func (MarkdownTableColumnsConsistent) Name() string { return "markdown_table_columns_consistent" }

func (MarkdownTableColumnsConsistent) Run(ctx context.Context, output string, _ map[string]any) Result {
	var tableRows []int
	for _, line := range strings.Split(output, "\n") {
		trim := strings.TrimSpace(line)
		if !strings.Contains(trim, "|") {
			continue
		}
		cells := strings.Split(trim, "|")
		if len(cells) >= 2 && strings.TrimSpace(cells[0]) == "" {
			cells = cells[1:]
		}
		if len(cells) >= 1 && strings.TrimSpace(cells[len(cells)-1]) == "" {
			cells = cells[:len(cells)-1]
		}
		if len(cells) < 2 {
			continue
		}
		if strings.Contains(trim, "---") {
			continue
		}
		tableRows = append(tableRows, len(cells))
	}
	if len(tableRows) == 0 {
		return passResult("warn", "no markdown table")
	}
	want := tableRows[0]
	for _, n := range tableRows[1:] {
		if n != want {
			return failResult("warn", "inconsistent table columns", map[string]any{"expected": want, "got": n})
		}
	}
	return passResult("warn", "table columns ok")
}

type CodeBlockLanguageWhitelist struct{}

func (CodeBlockLanguageWhitelist) Name() string { return "code_block_language_whitelist" }

func (CodeBlockLanguageWhitelist) Run(ctx context.Context, output string, params map[string]any) Result {
	allowed := stringSliceParam(params["allowed_languages"])
	if len(allowed) == 0 {
		return skipResult("no allowed_languages configured")
	}
	allow := map[string]bool{}
	for _, lang := range allowed {
		allow[strings.ToLower(lang)] = true
	}
	var bad []string
	for _, m := range phaseACodeFenceRe.FindAllStringSubmatch(output, -1) {
		lang := strings.ToLower(strings.TrimSpace(m[1]))
		if lang == "" {
			continue
		}
		if !allow[lang] {
			bad = append(bad, lang)
		}
	}
	if len(bad) > 0 {
		return failResult("warn", "code block language not allowed", map[string]any{"languages": bad})
	}
	return passResult("warn", "languages ok")
}

type InstructionCompliance struct{}

func (InstructionCompliance) Name() string { return "instruction_compliance" }

func (InstructionCompliance) Run(ctx context.Context, output string, params map[string]any) Result {
	required := stringSliceParam(params["required_phrases"])
	if len(required) == 0 {
		return skipResult("no required_phrases configured")
	}
	ci, _ := params["case_insensitive"].(bool)
	text := output
	if ci {
		text = outputLowerFrom(ctx, output)
	}
	var missing []string
	for _, phrase := range required {
		p := phrase
		if ci {
			p = strings.ToLower(phrase)
		}
		if !strings.Contains(text, p) {
			missing = append(missing, phrase)
		}
	}
	if len(missing) > 0 {
		return failResult("error", "missing required phrases", map[string]any{"missing": missing})
	}
	return passResult("error", "instructions followed")
}

type ForbiddenContent struct{}

func (ForbiddenContent) Name() string { return "forbidden_content" }

func (ForbiddenContent) Run(ctx context.Context, output string, params map[string]any) Result {
	forbidden := stringSliceParam(params["forbidden"])
	if len(forbidden) == 0 {
		return skipResult("no forbidden content configured")
	}
	ci, _ := params["case_insensitive"].(bool)
	matched, err := matchAnyPattern(output, forbidden, ci)
	if err != nil {
		return failResult("error", "no valid patterns in params", map[string]any{"error": err.Error()})
	}
	if len(matched) > 0 {
		return failResult("error", "forbidden content matched", map[string]any{"matched": matched})
	}
	return passResult("error", "no forbidden content")
}

type SensitiveContent struct{}

func (SensitiveContent) Name() string { return "sensitive_content" }

func (SensitiveContent) Run(ctx context.Context, output string, params map[string]any) Result {
	topics := stringSliceParam(params["forbidden_topics"])
	if len(topics) == 0 {
		return skipResult("no forbidden_topics configured")
	}
	ci, _ := params["case_insensitive"].(bool)
	matched, err := matchAnyPattern(output, topics, ci)
	if err != nil {
		return failResult("error", "no valid patterns in params", map[string]any{"error": err.Error()})
	}
	if len(matched) > 0 {
		return failResult("error", "sensitive topic matched", map[string]any{"matched": matched})
	}
	return passResult("error", "no sensitive topics")
}

type ToneStyleRules struct{}

func (ToneStyleRules) Name() string { return "tone_style_rules" }

func (ToneStyleRules) Run(ctx context.Context, output string, params map[string]any) Result {
	if start, ok := params["must_start_with"].(string); ok && start != "" {
		if !strings.HasPrefix(strings.TrimSpace(output), start) {
			return failResult("warn", "missing required opening", map[string]any{"must_start_with": start})
		}
	}
	if end, ok := params["must_end_with"].(string); ok && end != "" {
		trim := strings.TrimSpace(output)
		if !strings.HasSuffix(trim, end) {
			return failResult("warn", "missing required closing", map[string]any{"must_end_with": end})
		}
	}
	if bc := params["bullet_count"]; bc != nil {
		want := intFromParam(bc, -1)
		if want >= 0 {
			got := countMarkdownBullets(output)
			if got != want {
				return failResult("warn", "bullet count mismatch", map[string]any{"expected": want, "got": got})
			}
		}
	}
	return passResult("warn", "tone/style ok")
}

func countMarkdownBullets(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "- ") || strings.HasPrefix(trim, "* ") {
			count++
		}
	}
	return count
}

type WordCountRange struct{}

func (WordCountRange) Name() string { return "word_count_range" }

func (WordCountRange) Run(ctx context.Context, output string, params map[string]any) Result {
	return wordCountRangeRun(output, intFromParam(params["min_words"], 0), intFromParam(params["max_words"], 0))
}

func (WordCountRange) Compile(params map[string]any) CompiledRunner {
	min := intFromParam(params["min_words"], 0)
	max := intFromParam(params["max_words"], 0)
	return func(_ context.Context, output string) Result {
		return wordCountRangeRun(output, min, max)
	}
}

func wordCountRangeRun(output string, min, max int) Result {
	words := countWordsUnicode(output)
	if min > 0 && words < min {
		return failResult("warn", fmt.Sprintf("word count %d below min %d", words, min), map[string]any{"words": words})
	}
	if max > 0 && words > max {
		return failResult("warn", fmt.Sprintf("word count %d above max %d", words, max), map[string]any{"words": words})
	}
	return passResult("warn", "word count ok")
}

// countWordsUnicode returns the same value as len(strings.Fields(s)) but
// without allocating the intermediate []string. We follow the same
// definition: a "word" is a non-empty run of non-whitespace runes (as
// classified by unicode.IsSpace).
func countWordsUnicode(s string) int {
	count := 0
	inWord := false
	// Fast ASCII path: when the input only contains 7-bit characters, we
	// can scan bytes directly which is ~3x faster than rune iteration.
	if isASCII(s) {
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f' {
				if inWord {
					count++
					inWord = false
				}
				continue
			}
			inWord = true
		}
		if inWord {
			count++
		}
		return count
	}
	// Unicode-aware fallback uses unicode.IsSpace for parity with
	// strings.Fields.
	for _, r := range s {
		if unicode.IsSpace(r) {
			if inWord {
				count++
				inWord = false
			}
			continue
		}
		inWord = true
	}
	if inWord {
		count++
	}
	return count
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

