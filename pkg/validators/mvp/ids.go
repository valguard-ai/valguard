package mvp

// RuleIDs are the 40 OSS v0.1 rule IDs (appendix A, B1 audit).
var RuleIDs = []string{
	"valid_format", "json_schema_validate", "required_fields", "type_checking", "enum_validator",
	"json_no_extra_fields", "no_duplicate_json_keys", "json_max_depth", "list_length", "output_shape",
	"regex_pattern", "forbidden_patterns", "max_length", "min_length", "no_markdown_fenced_wrapper",
	"code_block_balanced", "no_placeholder_text", "no_truncation_markers",
	"range_validator", "numeric_consistency", "line_item_math_valid", "cross_field_logic",
	"decimal_precision_max", "non_negative", "amount_requires_currency",
	"pii_detection", "pii_pesel", "pii_nip", "pii_ssn_us", "valid_iban", "valid_credit_card_luhn",
	"masked_card_only", "valid_email_rfc5322",
	"secrets_detection", "no_system_prompt_leakage", "no_role_marker_leakage", "no_thinking_tag_leakage",
	"no_stack_trace_leak", "prompt_injection", "internal_network_url",
}

// IsRuleID reports whether id is one of the 40 OSS-supported rule IDs (D6).
func IsRuleID(id string) bool {
	for _, mvp := range RuleIDs {
		if mvp == id {
			return true
		}
	}
	return false
}

// AuditStatus returns B1 audit label: pure, adapt:…, blocked:…, or empty if not in MVP set.
func AuditStatus(id string) string {
	if IsRuleID(id) {
		return "pure"
	}
	return ""
}
