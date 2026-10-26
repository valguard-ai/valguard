//go:build !osspublic

package validators

import "context"

type MedicalDisclaimerRequired struct{}

func (MedicalDisclaimerRequired) Name() string { return "medical_disclaimer_required" }

func (MedicalDisclaimerRequired) Run(ctx context.Context, output string, params map[string]any) Result {
	triggers := stringSliceParam(params["trigger_keywords"])
	if len(triggers) == 0 {
		triggers = []string{"diagnosis", "treatment", "medication"}
	}
	if !containsAnyKeyword(output, triggers) {
		return passResult("error", "no medical triggers")
	}
	phrases := stringSliceParam(params["disclaimer_phrases"])
	if len(phrases) == 0 {
		phrases = []string{"not medical advice", "consult a healthcare professional"}
	}
	if containsAnyPhrase(output, phrases) {
		return passResult("error", "medical disclaimer present")
	}
	return failResult("error", "medical disclaimer required", map[string]any{"triggers": triggers})
}

type FinancialDisclaimerRequired struct{}

func (FinancialDisclaimerRequired) Name() string { return "financial_disclaimer_required" }

func (FinancialDisclaimerRequired) Run(ctx context.Context, output string, params map[string]any) Result {
	triggers := stringSliceParam(params["trigger_keywords"])
	if len(triggers) == 0 {
		triggers = []string{"investment", "returns", "stock"}
	}
	if !containsAnyKeyword(output, triggers) {
		return passResult("error", "no financial triggers")
	}
	phrases := stringSliceParam(params["disclaimer_phrases"])
	if len(phrases) == 0 {
		phrases = []string{"not financial advice", "past performance"}
	}
	if containsAnyPhrase(output, phrases) {
		return passResult("error", "financial disclaimer present")
	}
	return failResult("error", "financial disclaimer required", nil)
}

type UnsubscribeLinkRequired struct{}

func (UnsubscribeLinkRequired) Name() string { return "unsubscribe_link_required" }

func (UnsubscribeLinkRequired) Run(ctx context.Context, output string, params map[string]any) Result {
	patterns := stringSliceParam(params["link_patterns"])
	if len(patterns) == 0 {
		patterns = []string{"unsubscribe", "opt-out", "opt out"}
	}
	if containsAnyPhrase(output, patterns) {
		return passResult("error", "unsubscribe present")
	}
	return failResult("error", "unsubscribe link required", nil)
}
