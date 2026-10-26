package validators

import "context"

type RegexPattern struct{}

func (RegexPattern) Name() string { return "regex_pattern" }

func (RegexPattern) Run(ctx context.Context, output string, params map[string]any) Result {
	pattern, _ := params["pattern"].(string)
	if pattern == "" {
		return Result{Status: "skipped", Severity: "error", Message: "no pattern configured"}
	}
	re, err := compileCached(pattern)
	if err != nil {
		return Result{Status: "fail", Severity: "error", Message: "invalid regex in params", Details: map[string]any{"error": err.Error()}}
	}
	if !re.MatchString(output) {
		return Result{Status: "fail", Severity: "error", Message: "output does not match required pattern"}
	}
	return Result{Status: "pass", Severity: "error", Message: "pattern matched"}
}

func (RegexPattern) Compile(params map[string]any) CompiledRunner {
	pattern, _ := params["pattern"].(string)
	if pattern == "" {
		return func(_ context.Context, _ string) Result {
			return Result{Status: "skipped", Severity: "error", Message: "no pattern configured"}
		}
	}
	re, err := compileCached(pattern)
	if err != nil {
		msg := err.Error()
		return func(_ context.Context, _ string) Result {
			return Result{Status: "fail", Severity: "error", Message: "invalid regex in params", Details: map[string]any{"error": msg}}
		}
	}
	return func(_ context.Context, output string) Result {
		if !re.MatchString(output) {
			return Result{Status: "fail", Severity: "error", Message: "output does not match required pattern"}
		}
		return Result{Status: "pass", Severity: "error", Message: "pattern matched"}
	}
}
