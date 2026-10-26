package rules

import ossmvp "github.com/valguard-ai/valguard/pkg/validators/mvp"

// MVPRuleIDs are the 40 OSS v0.1 rule IDs (appendix A, B1 audit).
var MVPRuleIDs = ossmvp.RuleIDs

// MVPAuditStatus returns B1 audit label: pure, adapt:…, blocked:…, or empty if not in MVP set.
func MVPAuditStatus(id string) string { return ossmvp.AuditStatus(id) }
