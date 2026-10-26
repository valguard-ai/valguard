//go:build !osspublic

package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// branchOutputMap resolves fan-out branch outputs from merge input, current output,
// or PriorOutputs (first validator package use of PriorOutputs).
func branchOutputMap(ctx context.Context, output string, params map[string]any) (map[string]string, error) {
	meta := RunMetaFrom(ctx)
	keys := stringSliceParam(params["branch_step_keys"])

	if meta.InputText != "" {
		if sections := parseAppendOutputSections(meta.InputText); len(sections) > 0 {
			return sections, nil
		}
		if obj, err := parseJSONObject(meta.InputText); err == nil && len(obj) > 0 {
			return stringifyJSONObjectMap(obj), nil
		}
	}

	if obj, err := parseJSONObject(output); err == nil && len(obj) > 0 {
		return stringifyJSONObjectMap(obj), nil
	}

	if sections := parseAppendOutputSections(output); len(sections) > 0 {
		return sections, nil
	}

	if len(meta.PriorOutputs) > 0 {
		out := make(map[string]string, len(meta.PriorOutputs))
		if len(keys) > 0 {
			for i, key := range keys {
				if i < len(meta.PriorOutputs) {
					out[key] = meta.PriorOutputs[i]
				}
			}
			if len(out) > 0 {
				return out, nil
			}
		}
		for i, raw := range meta.PriorOutputs {
			out[fmt.Sprintf("branch_%d", i)] = raw
		}
		return out, nil
	}

	return nil, fmt.Errorf("no branch outputs available")
}

func stringifyJSONObjectMap(obj map[string]any) map[string]string {
	out := make(map[string]string, len(obj))
	for k, v := range obj {
		switch t := v.(type) {
		case string:
			out[k] = t
		default:
			b, err := json.Marshal(t)
			if err != nil {
				out[k] = fmt.Sprint(v)
			} else {
				out[k] = string(b)
			}
		}
	}
	return out
}

func parseAppendOutputSections(text string) map[string]string {
	const marker = "=== "
	out := make(map[string]string)
	lines := strings.Split(text, "\n")
	var currentKey string
	var buf []string
	flush := func() {
		if currentKey == "" {
			return
		}
		out[currentKey] = strings.TrimSpace(strings.Join(buf, "\n"))
		buf = nil
	}
	for _, line := range lines {
		if strings.HasPrefix(line, marker) && strings.HasSuffix(strings.TrimSpace(line), " ===") {
			flush()
			inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(line), marker), " ===")
			currentKey = strings.TrimSpace(inner)
			continue
		}
		if currentKey != "" {
			buf = append(buf, line)
		}
	}
	flush()
	if len(out) == 0 {
		return nil
	}
	return out
}

func jsonFieldFromBranch(raw string, field string) (any, bool) {
	obj, err := parseJSONObject(raw)
	if err != nil {
		return nil, false
	}
	return nestedJSONField(obj, field)
}

func nestedJSONField(obj map[string]any, field string) (any, bool) {
	parts := strings.Split(field, ".")
	var cur any = obj
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[p]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func branchCountsAsPass(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	obj, err := parseJSONObject(raw)
	if err != nil {
		return false
	}
	if blocked, ok := obj["blocked"].(bool); ok && blocked {
		return false
	}
	if status, ok := obj["validation_status"].(string); ok {
		switch strings.ToLower(status) {
		case "block", "blocked", "fail", "failed":
			return false
		}
	}
	if pass, ok := obj["pass"].(bool); ok && !pass {
		return false
	}
	return true
}

// BranchOutputsPresent ensures merged or prior branch payloads include expected step keys.
type BranchOutputsPresent struct{}

func (BranchOutputsPresent) Name() string { return "branch_outputs_present" }

func (BranchOutputsPresent) Run(ctx context.Context, output string, params map[string]any) Result {
	keys := stringSliceParam(params["branch_step_keys"])
	if len(keys) == 0 {
		return skipResult("branch_step_keys required")
	}
	branches, err := branchOutputMap(ctx, output, params)
	if err != nil {
		return failResult("error", "branch outputs unavailable", map[string]any{"err": err.Error()})
	}
	var missing []string
	for _, key := range keys {
		raw, ok := branches[key]
		if !ok || strings.TrimSpace(raw) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return failResult("error", "missing branch outputs", map[string]any{"missing": missing})
	}
	return passResult("error", "all branch outputs present")
}

// CrossBranchFieldMatch compares fields across fan-out branch JSON outputs.
type CrossBranchFieldMatch struct{}

func (CrossBranchFieldMatch) Name() string { return "cross_branch_field_match" }

func (CrossBranchFieldMatch) Run(ctx context.Context, output string, params map[string]any) Result {
	comparisons, ok := params["comparisons"].([]any)
	if !ok || len(comparisons) == 0 {
		return skipResult("comparisons required")
	}
	branches, err := branchOutputMap(ctx, output, params)
	if err != nil {
		return failResult("error", "branch outputs unavailable", map[string]any{"err": err.Error()})
	}
	for i, item := range comparisons {
		rule, ok := item.(map[string]any)
		if !ok {
			return skipResult(fmt.Sprintf("comparisons[%d] must be an object", i))
		}
		leftKey, _ := rule["left_key"].(string)
		rightKey, _ := rule["right_key"].(string)
		leftField, _ := rule["left_field"].(string)
		rightField, _ := rule["right_field"].(string)
		if leftKey == "" || rightKey == "" || leftField == "" || rightField == "" {
			return skipResult("each comparison needs left_key, right_key, left_field, right_field")
		}
		leftRaw, ok := branches[leftKey]
		if !ok {
			return failResult("error", "left branch output missing", map[string]any{"branch": leftKey})
		}
		rightRaw, ok := branches[rightKey]
		if !ok {
			return failResult("error", "right branch output missing", map[string]any{"branch": rightKey})
		}
		leftVal, ok := jsonFieldFromBranch(leftRaw, leftField)
		if !ok {
			return failResult("error", "left field missing", map[string]any{"branch": leftKey, "field": leftField})
		}
		rightVal, ok := jsonFieldFromBranch(rightRaw, rightField)
		if !ok {
			return failResult("error", "right field missing", map[string]any{"branch": rightKey, "field": rightField})
		}
		tolerance := 0.0
		if v, ok := rule["tolerance"].(float64); ok {
			tolerance = v
		}
		if !valuesMatchWithTolerance(leftVal, rightVal, tolerance) {
			return failResult("error", "cross-branch field mismatch", map[string]any{
				"left_key": leftKey, "left_field": leftField, "left_value": leftVal,
				"right_key": rightKey, "right_field": rightField, "right_value": rightVal,
			})
		}
	}
	return passResult("error", "cross-branch fields match")
}

func valuesMatchWithTolerance(left, right any, tolerance float64) bool {
	lf, lok := toFloat64(left)
	rf, rok := toFloat64(right)
	if lok && rok {
		return math.Abs(lf-rf) <= tolerance
	}
	return fmt.Sprint(left) == fmt.Sprint(right)
}

func toFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// BranchPassCount validates enough parallel branches passed for partial-success flows.
type BranchPassCount struct{}

func (BranchPassCount) Name() string { return "branch_pass_count" }

func (BranchPassCount) Run(ctx context.Context, output string, params map[string]any) Result {
	minPassing := 1
	if v, ok := params["min_passing_branches"].(float64); ok && int(v) > 0 {
		minPassing = int(v)
	} else if v, ok := params["min_passing_branches"].(int); ok && v > 0 {
		minPassing = v
	}
	keys := stringSliceParam(params["branch_step_keys"])
	branches, err := branchOutputMap(ctx, output, params)
	if err != nil {
		return failResult("error", "branch outputs unavailable", map[string]any{"err": err.Error()})
	}
	passCount := 0
	total := 0
	if len(keys) > 0 {
		total = len(keys)
		for _, key := range keys {
			if branchCountsAsPass(branches[key]) {
				passCount++
			}
		}
	} else {
		for _, raw := range branches {
			total++
			if branchCountsAsPass(raw) {
				passCount++
			}
		}
	}
	if passCount < minPassing {
		return failResult("error", "not enough branches passed", map[string]any{
			"pass_count": passCount, "total": total, "min_passing": minPassing,
		})
	}
	return passResult("error", "branch pass count ok")
}
