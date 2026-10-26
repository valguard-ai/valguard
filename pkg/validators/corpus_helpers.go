package validators

import (
	"regexp"
	"strings"
)

var (
	wordTokenRe = regexp.MustCompile(`[A-Za-z0-9]{3,}`)
	numberRe    = regexp.MustCompile(`\d+(?:\.\d+)?`)
)

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

func tokenSet(text string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, tok := range wordTokenRe.FindAllString(strings.ToLower(text), -1) {
		set[tok] = struct{}{}
	}
	return set
}

func extractNumbers(text string) []string {
	return numberRe.FindAllString(text, -1)
}
