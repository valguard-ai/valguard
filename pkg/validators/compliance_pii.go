//go:build !osspublic

package validators

import (
	"context"
	"regexp"
)

var (
	nhsRe = regexp.MustCompile(`\b\d{3}\s?\d{3}\s?\d{4}\b`)
	ahvRe = regexp.MustCompile(`\b756\.\d{4}\.\d{4}\.\d{2}\b`)
)

type PIINHSNumber struct{}

func (PIINHSNumber) Name() string { return "pii_nhs_number" }

func (PIINHSNumber) Run(ctx context.Context, output string, _ map[string]any) Result {
	if nhsRe.MatchString(output) {
		return failResult("error", "UK NHS number pattern detected", nil)
	}
	return passResult("error", "no NHS number")
}

type PIIAHV struct{}

func (PIIAHV) Name() string { return "pii_ahv" }

func (PIIAHV) Run(ctx context.Context, output string, _ map[string]any) Result {
	if ahvRe.MatchString(output) {
		return failResult("error", "Swiss AHV pattern detected", nil)
	}
	return passResult("error", "no AHV")
}

type PHIUSHipaaPatterns struct{}

func (PHIUSHipaaPatterns) Name() string { return "phi_us_hipaa_patterns" }

func (PHIUSHipaaPatterns) Run(ctx context.Context, output string, params map[string]any) Result {
	patterns := stringSliceParam(params["patterns"])
	if len(patterns) == 0 {
		patterns = []string{
			`\bMRN[:\s#]*\d+`,
			`\bICD-?\d{2,}`,
			`\bpatient\s+id[:\s]+\d+`,
			`\bHIPAA\b`,
		}
	}
	if m, err := matchAnyPattern(output, patterns, true); err != nil {
		return failResult("error", "no valid patterns in params", map[string]any{"error": err.Error()})
	} else if len(m) > 0 {
		return failResult("error", "PHI pattern detected", map[string]any{"matched": m})
	}
	return passResult("error", "no PHI patterns")
}

type PCIDSSPatterns struct{}

func (PCIDSSPatterns) Name() string { return "pci_dss_patterns" }

func (PCIDSSPatterns) Run(ctx context.Context, output string, params map[string]any) Result {
	patterns := stringSliceParam(params["patterns"])
	if len(patterns) == 0 {
		patterns = []string{
			`\b\d{13,19}\b`,
			`\bCVV[:\s]*\d{3,4}\b`,
			`%B\d{13,19}`,
			`\btrack\s*2\b`,
		}
	}
	if m, err := matchAnyPattern(output, patterns, true); err != nil {
		return failResult("error", "no valid patterns in params", map[string]any{"error": err.Error()})
	} else if len(m) > 0 {
		return failResult("error", "PCI-DSS pattern detected", map[string]any{"matched": m})
	}
	if card := luhnCardInText(output); card != "" {
		return failResult("error", "PCI-DSS pattern detected", map[string]any{"kind": "card"})
	}
	return passResult("error", "no PCI patterns")
}
