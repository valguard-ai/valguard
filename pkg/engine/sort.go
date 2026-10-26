package engine

import "sort"

// Fail-fast execution order: cheap string/length checks first, then regex-heavy
// security rules, then RAG / schema / checksum validators last. Within the same
// cost tier, DB priority ASC is preserved (stable sort).
//
// Tier 0: max_length, regex_pattern, required_fields, valid_format, …
// Tier 1: PII / injection / secrets (compiled regex sweeps)
// Tier 2: JSON schema, OpenAPI, compliance ID checks
// Tier 3: RAG fidelity (fact_match, input_coverage, rag_source_*)
func validatorCostTier(name string) int {
	switch name {
	case "max_length", "min_length", "regex_pattern", "forbidden_patterns",
		"required_fields", "valid_format", "word_count_range", "list_length",
		"enum_validator", "range_validator", "bracket_balanced", "markdown_format",
		"required_sections", "no_placeholder_text", "no_lorem_ipsum",
		"complete_sentence_ending", "integer_only", "non_negative",
		"decimal_precision_max", "currency_format", "percentage_format":
		return 0
	case "pii_detection", "sql_injection", "prompt_injection", "secrets_detection",
		"xss_pattern", "path_traversal", "url_safety", "forbidden_content",
		"sensitive_content", "homoglyph_detection":
		return 1
	case "json_schema_validate", "valid_openapi_spec", "openapi_spec",
		"valid_vat_eu", "valid_iso_currency", "valid_iso_country", "valid_iso_language",
		"pesel_valid", "nip_valid", "iban_valid", "nhs_valid", "ahv_valid",
		"ssn_valid", "type_checking", "cross_field_logic", "dependency_rules":
		return 2
	case "fact_match", "input_coverage", "input_output_diff",
		"rag_source_presence", "rag_source_coverage", "rag_source_match",
		"citation_format_valid", "step_consistency", "required_step_order",
		"state_transition_rules", "multi_step_numeric_logic", "input_fidelity":
		return 3
	default:
		return 1
	}
}

type instanceSortKey struct {
	tier     int
	priority int32
	index    int
}

// SortInstancesForFailFast reorders instances in place for tiered fail-fast dispatch.
func SortInstancesForFailFast(instances []Instance) {
	if len(instances) < 2 {
		return
	}
	keys := make([]instanceSortKey, len(instances))
	for i, inst := range instances {
		keys[i] = instanceSortKey{
			tier:     validatorCostTier(inst.Name),
			priority: inst.Priority,
			index:    i,
		}
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if keys[i].tier != keys[j].tier {
			return keys[i].tier < keys[j].tier
		}
		if keys[i].priority != keys[j].priority {
			return keys[i].priority < keys[j].priority
		}
		return keys[i].index < keys[j].index
	})
	sorted := make([]Instance, len(instances))
	for i, k := range keys {
		sorted[i] = instances[k.index]
	}
	copy(instances, sorted)
}
