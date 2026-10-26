package policy

import (
	"context"
	"fmt"
	"time"

	"github.com/valguard-ai/valguard/pkg/engine"
	"github.com/valguard-ai/valguard/pkg/validators"
	"github.com/valguard-ai/valguard/pkg/validators/mvp"
)

type RunResult struct {
	Blocked      bool
	WouldEnforce bool
	Shadow       bool
	FailedRules  []string
	PassedRules  []string
}

// RuleResult is one validator outcome for shadow reporting.
type RuleResult struct {
	RuleID       string
	Status       string
	Message      string
	WouldEnforce bool
}

// DetailedRunResult includes per-rule outcomes (before shadow passthrough).
type DetailedRunResult struct {
	RunResult
	Rules []RuleResult
}

func Run(doc Document, output, inputText string) (RunResult, error) {
	detail, err := RunDetailed(doc, output, inputText)
	return detail.RunResult, err
}

func RunDetailed(doc Document, output, inputText string) (DetailedRunResult, error) {
	return RunDetailedWithTimeout(doc, output, inputText, DefaultValidationTimeout)
}

// RunDetailedWithTimeout runs validators with an explicit wall-clock cap (0 → default).
func RunDetailedWithTimeout(doc Document, output, inputText string, timeout time.Duration) (DetailedRunResult, error) {
	if timeout <= 0 {
		timeout = DefaultValidationTimeout
	}
	if len(output) > MaxOutputBytes {
		return DetailedRunResult{}, fmt.Errorf("output exceeds %d bytes", MaxOutputBytes)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return runWithContext(ctx, doc, output, inputText, timeout)
}

func runWithContext(ctx context.Context, doc Document, output, inputText string, timeout time.Duration) (DetailedRunResult, error) {
	// Offline OSS CLI runs only the 40 MVP rules (D6); full catalog stays on hosted proxy.
	reg := mvp.NewExtractedRegistry()
	instances := doc.ToEngineInstances()
	meta := validators.RunMeta{InputText: inputText}
	engine.ResolveInstances(reg, instances)
	out := engine.RunForPlan(ctx, reg, instances, output, meta, engine.RunOptions{})
	if err := ctx.Err(); err != nil {
		return DetailedRunResult{}, fmt.Errorf("validation timed out after %s: %w", timeout, err)
	}
	wouldEnforce := engine.WouldEnforce(out)

	rules := make([]RuleResult, 0, len(out.Results))
	for i, r := range out.Results {
		name := ""
		if i < len(instances) {
			name = instances[i].Name
		}
		would := false
		if i < len(instances) {
			would = engine.InstanceWouldEnforce(instances[i], r)
		}
		rules = append(rules, RuleResult{
			RuleID:       name,
			Status:       r.Status,
			Message:      r.Message,
			WouldEnforce: would,
		})
	}

	if doc.Mode == "shadow" {
		out = engine.ApplyShadowMode(out)
	}
	res := RunResult{Shadow: doc.Mode == "shadow"}
	for i, r := range out.Results {
		name := ""
		if i < len(instances) {
			name = instances[i].Name
		}
		if r.Status == "fail" || r.Status == "error" {
			res.FailedRules = append(res.FailedRules, name)
		} else if r.Status == "pass" {
			res.PassedRules = append(res.PassedRules, name)
		}
	}
	res.Blocked = out.Blocked
	res.WouldEnforce = wouldEnforce
	return DetailedRunResult{RunResult: res, Rules: rules}, nil
}
