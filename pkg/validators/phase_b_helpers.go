//go:build !osspublic

package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	wordTokenRe     = regexp.MustCompile(`[A-Za-z0-9]{3,}`)
	numberRe        = regexp.MustCompile(`\d+(?:\.\d+)?`)
	homoglyphURLRe  = regexp.MustCompile(`https?://[^\s]+`)
	homoglyphHostRe = regexp.MustCompile(`https?://([^/\s]+)`)
	vatFullRe       = regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]+$`)
	vatDERe         = regexp.MustCompile(`^\d{9}$`)
	vatPLRe         = regexp.MustCompile(`^\d{10}$`)
	vatFRRe         = regexp.MustCompile(`^[A-HJ-NP-Z0-9]{2}\d{9}$`)
	vatITRe         = regexp.MustCompile(`^\d{11}$`)
	vatESRe         = regexp.MustCompile(`^[A-Z0-9]\d{7}[A-Z0-9]$`)
)

// ParseRAGSourcesHeader parses X-VG-Sources: JSON array of strings or {id,text} objects.
func ParseRAGSourcesHeader(raw string) []RAGSource {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	switch raw[0] {
	case '[':
		inner := strings.TrimSpace(raw[1:])
		if len(inner) > 0 && inner[0] == '{' {
			var objs []RAGSource
			if err := json.Unmarshal([]byte(raw), &objs); err != nil {
				return nil
			}
			return ragSourcesFromObjects(objs)
		}
		var texts []string
		if err := json.Unmarshal([]byte(raw), &texts); err != nil {
			return nil
		}
		return ragSourcesFromStrings(texts)
	case '{':
		var obj RAGSource
		if err := json.Unmarshal([]byte(raw), &obj); err != nil {
			return nil
		}
		return ragSourcesFromObjects([]RAGSource{obj})
	default:
		return nil
	}
}

func ragSourcesFromStrings(texts []string) []RAGSource {
	out := make([]RAGSource, 0, len(texts))
	for i, t := range texts {
		if strings.TrimSpace(t) == "" {
			continue
		}
		out = append(out, RAGSource{ID: strconv.Itoa(i + 1), Text: t})
	}
	return out
}

func ragSourcesFromObjects(objs []RAGSource) []RAGSource {
	out := make([]RAGSource, 0, len(objs))
	for i, o := range objs {
		if strings.TrimSpace(o.Text) == "" {
			continue
		}
		id := strings.TrimSpace(o.ID)
		if id == "" {
			id = strconv.Itoa(i + 1)
		}
		out = append(out, RAGSource{ID: id, Text: o.Text})
	}
	return out
}

// RagCorpus joins input text and RAG source bodies for fidelity validators.
func RagCorpus(meta RunMeta) string {
	return ragCorpus(meta)
}

func ragCorpus(meta RunMeta) string {
	var parts []string
	if meta.InputText != "" {
		parts = append(parts, meta.InputText)
	}
	for _, s := range meta.RAGSources {
		if s.Text != "" {
			parts = append(parts, s.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func ragSourceIDs(meta RunMeta) []string {
	return buildRagSourceIDs(meta.RAGSources)
}

// TokenSet lowercases word tokens from text for corpus coverage checks.
func TokenSet(text string) map[string]struct{} {
	return tokenSet(text)
}

func tokenSet(text string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, tok := range wordTokenRe.FindAllString(strings.ToLower(text), -1) {
		set[tok] = struct{}{}
	}
	return set
}

// ExtractNumbers finds numeric literals in text for corpus checks.
func ExtractNumbers(text string) []string {
	return extractNumbers(text)
}

func extractNumbers(text string) []string {
	return numberRe.FindAllString(text, -1)
}

func jsonFieldString(obj map[string]any, field string) (string, bool) {
	v, ok := obj[field]
	if !ok {
		return "", false
	}
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(t), true
	default:
		return fmt.Sprint(v), true
	}
}

func parseJSONArrayField(ctx context.Context, output string, field string) ([]map[string]any, error) {
	obj, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return nil, err
	}
	raw, ok := obj[field]
	if !ok {
		return nil, fmt.Errorf("missing field %s", field)
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("field %s is not array", field)
	}
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("step is not object")
		}
		out = append(out, m)
	}
	return out, nil
}

func hasHomoglyphURL(text string) bool {
	for _, u := range homoglyphURLRe.FindAllString(text, -1) {
		if mixedScriptHost(u) {
			return true
		}
	}
	return false
}

func mixedScriptHost(rawURL string) bool {
	m := homoglyphHostRe.FindStringSubmatch(rawURL)
	if len(m) < 2 {
		return false
	}
	host := m[1]
	var latin, other int
	for _, r := range host {
		if !unicode.IsLetter(r) {
			continue
		}
		if unicode.Is(unicode.Latin, r) {
			latin++
		} else {
			other++
		}
	}
	n := norm.NFKC.String(host)
	if n != host && (latin > 0 && other > 0) {
		return true
	}
	return latin > 0 && other > 0
}

func validEUVATNumber(vat string) bool {
	vat = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(vat), " ", ""))
	if len(vat) < 4 {
		return false
	}
	country := vat[:2]
	body := vat[2:]
	if !vatFullRe.MatchString(vat) {
		return false
	}
	switch country {
	case "DE":
		return vatDERe.MatchString(body)
	case "PL":
		return vatPLRe.MatchString(body)
	case "FR":
		return vatFRRe.MatchString(body)
	case "IT":
		return vatITRe.MatchString(body)
	case "ES":
		return vatESRe.MatchString(body)
	default:
		return len(body) >= 2 && len(body) <= 13
	}
}

