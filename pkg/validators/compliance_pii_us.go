//go:build !osspublic

package validators

import (
	"context"
	"regexp"
	"strconv"
)

var (
	einRe  = regexp.MustCompile(`\b\d{2}-\d{7}\b`)
	itinRe = regexp.MustCompile(`\b9\d{2}-\d{2}-\d{4}\b`)
	mbiRe  = regexp.MustCompile(`(?i)\b[1-9][ACDEFGHJKMNPQRTUVWXY][0-9ACDEFGHJKMNPQRTUVWXY]{8}[0-9]\b`)
)

func itinValid(s string) bool {
	d := digits(s)
	if len(d) != 9 || d[0] != '9' {
		return false
	}
	group, err := strconv.Atoi(d[3:5])
	if err != nil {
		return false
	}
	if group >= 50 && group <= 65 {
		return true
	}
	if group >= 70 && group <= 88 {
		return true
	}
	if group >= 90 && group <= 92 {
		return true
	}
	return group >= 94 && group <= 99
}

type PIIEINUS struct{}

func (PIIEINUS) Name() string { return "pii_ein_us" }

func (PIIEINUS) Run(ctx context.Context, output string, _ map[string]any) Result {
	if einRe.MatchString(output) {
		return failResult("error", "US EIN pattern detected", nil)
	}
	return passResult("error", "no US EIN")
}

type PIIITINUS struct{}

func (PIIITINUS) Name() string { return "pii_itin_us" }

func (PIIITINUS) Run(ctx context.Context, output string, _ map[string]any) Result {
	if scanMatches(output, itinRe, itinValid) {
		return failResult("error", "US ITIN pattern detected", nil)
	}
	return passResult("error", "no US ITIN")
}

type PIIMedicareMBI struct{}

func (PIIMedicareMBI) Name() string { return "pii_medicare_mbi" }

func (PIIMedicareMBI) Run(ctx context.Context, output string, _ map[string]any) Result {
	if mbiRe.MatchString(output) {
		return failResult("error", "Medicare MBI pattern detected", nil)
	}
	return passResult("error", "no Medicare MBI")
}
