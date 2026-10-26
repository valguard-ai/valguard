package validators

import (
	"context"
	"regexp"
)

var (
	peselRe = regexp.MustCompile(`\b\d{11}\b`)
	nipRe   = regexp.MustCompile(`\b\d{3}-?\d{3}-?\d{2}-?\d{2}\b|\b\d{10}\b`)
	ssnRe   = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

	piiDefaultPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		regexp.MustCompile(`\b\d{16}\b`),
		regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9]{11,30}\b`),
		regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`),
	}
	spacedCardRe = regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`)
)

type PIIDetection struct{}

func (PIIDetection) Name() string { return "pii_detection" }

func (PIIDetection) Run(_ context.Context, output string, params map[string]any) Result {
	regs, err := resolvePIIPatterns(params)
	if err != nil {
		return failResult("error", "no valid patterns in params", map[string]any{"error": err.Error()})
	}
	return piiDetectionRun(output, regs)
}

func (PIIDetection) Compile(params map[string]any) CompiledRunner {
	regs, err := resolvePIIPatterns(params)
	if err != nil {
		msg := err.Error()
		return func(_ context.Context, _ string) Result {
			return failResult("error", "no valid patterns in params", map[string]any{"error": msg})
		}
	}
	return func(_ context.Context, output string) Result {
		return piiDetectionRun(output, regs)
	}
}

func resolvePIIPatterns(params map[string]any) ([]*regexp.Regexp, error) {
	custom := stringSliceParam(params["patterns"])
	if len(custom) == 0 {
		return piiDefaultPatterns, nil
	}
	return compileCustomRegexPatterns(custom)
}

func piiDetectionRun(output string, regs []*regexp.Regexp) Result {
	for _, re := range regs {
		if re.MatchString(output) {
			return failResult("error", "possible PII detected", nil)
		}
	}
	if card := luhnCardInText(output); card != "" {
		return failResult("error", "possible PII detected", map[string]any{"kind": "card"})
	}
	return passResult("error", "no PII patterns")
}

func luhnCardInText(output string) string {
	for _, m := range spacedCardRe.FindAllString(output, -1) {
		d := digits(m)
		if len(d) < 13 || len(d) > 19 {
			continue
		}
		if luhnValid(d) {
			return d
		}
	}
	return ""
}

type PIIPesel struct{}

func (PIIPesel) Name() string { return "pii_pesel" }

func (PIIPesel) Run(_ context.Context, output string, _ map[string]any) Result {
	if scanMatches(output, peselRe, peselValid) {
		return failResult("error", "PESEL detected", nil)
	}
	return passResult("error", "no PESEL")
}

type PIINIP struct{}

func (PIINIP) Name() string { return "pii_nip" }

func (PIINIP) Run(_ context.Context, output string, _ map[string]any) Result {
	for _, m := range nipRe.FindAllString(output, -1) {
		if nipValid(m) {
			return failResult("error", "NIP detected", map[string]any{"match": m})
		}
	}
	return passResult("error", "no NIP")
}

type PIISsnUS struct{}

func (PIISsnUS) Name() string { return "pii_ssn_us" }

func (PIISsnUS) Run(_ context.Context, output string, _ map[string]any) Result {
	if ssnRe.MatchString(output) {
		return failResult("error", "US SSN pattern detected", nil)
	}
	return passResult("error", "no US SSN")
}
