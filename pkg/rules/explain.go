package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/valguard-ai/valguard/pkg/validators"
)

const ValidatorDocBaseURL = "https://www.valguard.ai/validators/"

// Explain holds offline metadata for `valguard rules explain`.
type Explain struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	OSS         string `json:"oss"` // pure | (empty if not in MVP set)
	InRegistry  bool   `json:"in_registry"`
	DocURL      string `json:"doc_url"`
}

// ExplainRule resolves catalog text and MVP audit status without network I/O.
func ExplainRule(id string) (Explain, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Explain{}, fmt.Errorf("rule id is required")
	}
	reg := validators.NewCatalogRegistry()
	names := reg.Names()
	inReg := slices.Contains(names, id)
	if !inReg {
		return Explain{}, fmt.Errorf("unknown rule %q (not in production registry)", id)
	}

	desc, category := lookupCatalog(id)
	out := Explain{
		ID:          id,
		Description: desc,
		Category:    category,
		OSS:         MVPAuditStatus(id),
		InRegistry:  true,
		DocURL:      ValidatorDocBaseURL + id,
	}
	if out.Description == "" {
		out.Description = "(no catalog description; see doc URL)"
	}
	return out, nil
}

func lookupCatalog(id string) (description, category string) {
	cat, err := validators.LoadCatalog(validators.DefaultCatalogPaths()...)
	if err != nil {
		return "", ""
	}
	for _, e := range cat.Validators {
		if e.Name == id {
			return strings.TrimSpace(e.Description), strings.TrimSpace(e.Category)
		}
	}
	return "", ""
}
