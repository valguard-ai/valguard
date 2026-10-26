package mvp

import (
	"github.com/valguard-ai/valguard/pkg/validators"
)

// ExtractedRuleIDs lists all 40 MVP rules implemented in the OSS module (slices 2–3).
var ExtractedRuleIDs = RuleIDs

// AllExtracted returns every MVP validator owned by the OSS module.
func AllExtracted() []validators.Validator {
	return validators.AllMVPValidators()
}

// NewExtractedRegistry builds a registry containing only OSS-extracted MVP rules.
func NewExtractedRegistry() *validators.Registry {
	return validators.NewRegistry(AllExtracted()...)
}
