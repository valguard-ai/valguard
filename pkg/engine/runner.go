package engine

import (
	"context"

	"github.com/valguard-ai/valguard/pkg/validators"
)

// Verdict is one validator outcome after dispatch (store-free).
type Verdict struct {
	RuleID  string
	Status  string
	Message string
	Result  validators.Result
}

// RunVerdicts executes enabled instances and returns aligned verdicts.
// It uses the same dispatch path as RunForPlan and recycles pooled buffers.
func RunVerdicts(ctx context.Context, reg ValidatorRegistry, instances []Instance, output string, meta validators.RunMeta, opts RunOptions) []Verdict {
	out := RunForPlan(ctx, reg, instances, output, meta, opts)
	defer ReleaseRun(out)
	verdicts := make([]Verdict, len(out.Results))
	for i, r := range out.Results {
		name := ""
		if i < len(instances) {
			name = instances[i].Name
		}
		verdicts[i] = Verdict{
			RuleID:  name,
			Status:  r.Status,
			Message: r.Message,
			Result:  r,
		}
	}
	return verdicts
}
