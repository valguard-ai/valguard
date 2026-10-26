package validators

import (
	"regexp"
	"strings"
)

type compiledPatternPlan struct {
	severity   string
	regs       []*regexp.Regexp
	sources    []string
	ciFallback []string
	errNoValid bool
	errMsg     string
}

func (plan compiledPatternPlan) run(output string) Result {
	if plan.errNoValid {
		return failResult(plan.severity, "no valid patterns in params", map[string]any{"error": plan.errMsg})
	}
	var matched []string
	for i, re := range plan.regs {
		if re.MatchString(output) {
			matched = append(matched, plan.sources[i])
		}
	}
	if len(plan.ciFallback) > 0 {
		lower := strings.ToLower(output)
		for _, p := range plan.ciFallback {
			if strings.Contains(lower, strings.ToLower(p)) {
				matched = append(matched, p)
			}
		}
	}
	if len(matched) > 0 {
		return failResult(plan.severity, "pattern matched", map[string]any{"matched": matched})
	}
	return passResult(plan.severity, "ok")
}

func scanMatches(output string, re *regexp.Regexp, validate func(string) bool) bool {
	for _, m := range re.FindAllString(output, -1) {
		if validate(m) {
			return true
		}
	}
	return false
}
