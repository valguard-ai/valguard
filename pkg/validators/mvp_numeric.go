package validators

import (
	"context"
	"math"
	"strings"
)

type RangeValidator struct{}

func (RangeValidator) Name() string { return "range_validator" }

func (RangeValidator) Run(ctx context.Context, output string, params map[string]any) Result {
	field, _ := params["field"].(string)
	minV := floatFromParam(params["min"], math.Inf(-1))
	maxV := floatFromParam(params["max"], math.Inf(1))
	if field == "" {
		return skipResult("field not found")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	val, ok := data[field]
	if !ok {
		return skipResult("field not found")
	}
	n := shallowToFloat(val)
	if n < minV || n > maxV {
		return failResult("error", "value out of range", map[string]any{"value": n, "min": minV, "max": maxV})
	}
	return passResult("error", "in range")
}

func (RangeValidator) Compile(params map[string]any) CompiledRunner {
	field, _ := params["field"].(string)
	minV := floatFromParam(params["min"], math.Inf(-1))
	maxV := floatFromParam(params["max"], math.Inf(1))
	return func(ctx context.Context, output string) Result {
		if field == "" {
			return skipResult("field not found")
		}
		data, err := parseShallowCtx(ctx, output)
		if err != nil {
			return failResult("error", "cannot parse JSON", nil)
		}
		val, ok := data[field]
		if !ok {
			return skipResult("field not found")
		}
		n := shallowToFloat(val)
		if n < minV || n > maxV {
			return failResult("error", "value out of range", map[string]any{"value": n, "min": minV, "max": maxV})
		}
		return passResult("error", "in range")
	}
}

type NumericConsistency struct{}

func (NumericConsistency) Name() string { return "numeric_consistency" }

func (NumericConsistency) Run(ctx context.Context, output string, params map[string]any) Result {
	sumField, _ := params["sum_field"].(string)
	fields, _ := params["fields"].([]any)
	if sumField == "" || len(fields) == 0 {
		return skipResult("sum_field and fields required")
	}
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var sum float64
	for _, f := range fields {
		name, _ := f.(string)
		if v, ok := data[name]; ok {
			sum += shallowToFloat(v)
		}
	}
	expected := shallowToFloat(data[sumField])
	if math.Abs(sum-expected) > 0.01 {
		return failResult("error", "sum mismatch", map[string]any{"computed": sum, "expected": expected})
	}
	return passResult("error", "numeric consistency ok")
}

// NumericRelations supports cross-field numeric expressions (also used by cross_field_logic).
type NumericRelations struct{}

type numericRelationProgram struct {
	leftTerms []string
	rightTerm string
	op        string
	ok        bool
}

func parseNumericRelation(expr string) numericRelationProgram {
	expr = strings.TrimSpace(expr)
	if parts := strings.Split(expr, "<="); len(parts) == 2 {
		return numericRelationProgram{
			ok:        true,
			op:        "<=",
			leftTerms: []string{strings.TrimSpace(parts[0])},
			rightTerm: strings.TrimSpace(parts[1]),
		}
	}
	parts := strings.Split(expr, "==")
	if len(parts) != 2 {
		return numericRelationProgram{}
	}
	lhs := strings.TrimSpace(parts[0])
	rhs := strings.TrimSpace(parts[1])
	terms := strings.Split(lhs, "+")
	out := numericRelationProgram{ok: true, op: "==", rightTerm: rhs, leftTerms: make([]string, 0, len(terms))}
	for _, t := range terms {
		out.leftTerms = append(out.leftTerms, strings.TrimSpace(t))
	}
	return out
}

func (NumericRelations) Name() string { return "numeric_relations" }

func (NumericRelations) Run(ctx context.Context, output string, params map[string]any) Result {
	expr, _ := params["expression"].(string)
	if expr == "" {
		return skipResult("expression required")
	}
	prog := parseNumericRelation(expr)
	if !prog.ok {
		return skipResult("unsupported expression; use 'a + b == c' or 'a <= b'")
	}
	return numericRelationsEval(ctx, output, prog)
}

func numericRelationsEval(ctx context.Context, output string, prog numericRelationProgram) Result {
	data, err := parseShallowCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON", nil)
	}
	var left float64
	for _, t := range prog.leftTerms {
		left += evalFieldRefShallow(t, data)
	}
	right := evalFieldRefShallow(prog.rightTerm, data)
	if prog.op == "<=" {
		if left > right+0.01 {
			return failResult("error", "relation failed", map[string]any{"left": left, "right": right, "op": "<="})
		}
		return passResult("error", "relation ok")
	}
	if math.Abs(left-right) > 0.01 {
		return failResult("error", "relation failed", map[string]any{"left": left, "right": right})
	}
	return passResult("error", "relation ok")
}

func (NumericRelations) Compile(params map[string]any) CompiledRunner {
	expr, _ := params["expression"].(string)
	if expr == "" {
		return func(_ context.Context, _ string) Result {
			return skipResult("expression required")
		}
	}
	prog := parseNumericRelation(expr)
	if !prog.ok {
		return func(_ context.Context, _ string) Result {
			return skipResult("unsupported expression; use 'a + b == c' or 'a <= b'")
		}
	}
	return func(ctx context.Context, output string) Result {
		return numericRelationsEval(ctx, output, prog)
	}
}

func evalFieldRefShallow(name string, data map[string]ShallowField) float64 {
	if v, ok := data[name]; ok {
		return shallowToFloat(v)
	}
	return 0
}

type CrossFieldLogic struct{}

func (CrossFieldLogic) Name() string { return "cross_field_logic" }

func (CrossFieldLogic) Run(ctx context.Context, output string, params map[string]any) Result {
	return (NumericRelations{}).Run(ctx, output, params)
}
