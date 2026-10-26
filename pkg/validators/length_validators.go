package validators

import (
	"context"
	"fmt"
)

type MaxLength struct{}

func (MaxLength) Name() string { return "max_length" }

func (MaxLength) Run(ctx context.Context, output string, params map[string]any) Result {
	max := int64(5000)
	if v, ok := params["max_chars"].(float64); ok {
		max = int64(v)
	}
	if int64(len(output)) > max {
		return Result{Status: "fail", Severity: "warn", Message: fmt.Sprintf("output exceeds max length %d", max), Details: map[string]any{"length": len(output), "max": max}}
	}
	return Result{Status: "pass", Severity: "warn", Message: "within max length"}
}

func (MaxLength) Compile(params map[string]any) CompiledRunner {
	max := int64(5000)
	if v, ok := params["max_chars"].(float64); ok {
		max = int64(v)
	}
	return func(_ context.Context, output string) Result {
		if int64(len(output)) > max {
			return Result{Status: "fail", Severity: "warn", Message: fmt.Sprintf("output exceeds max length %d", max), Details: map[string]any{"length": len(output), "max": max}}
		}
		return Result{Status: "pass", Severity: "warn", Message: "within max length"}
	}
}

type MinLength struct{}

func (MinLength) Name() string { return "min_length" }

func (MinLength) Run(ctx context.Context, output string, params map[string]any) Result {
	min := intFromParam(params["min_chars"], 1)
	if len(output) < min {
		return failResult("warn", fmt.Sprintf("below min length %d", min), map[string]any{"length": len(output)})
	}
	return passResult("warn", "min length ok")
}

func (MinLength) Compile(params map[string]any) CompiledRunner {
	min := intFromParam(params["min_chars"], 1)
	return func(_ context.Context, output string) Result {
		if len(output) < min {
			return failResult("warn", fmt.Sprintf("below min length %d", min), map[string]any{"length": len(output)})
		}
		return passResult("warn", "min length ok")
	}
}
