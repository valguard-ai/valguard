package validators

import "context"

// Result is the outcome of a single validator Run (mirrors proxy internal/validators).
type Result struct {
	Status   string // pass, fail, skipped
	Severity string
	Message  string
	Details  map[string]any
}

// Validator is a pure output validator: no network, no store.
type Validator interface {
	Name() string
	Run(ctx context.Context, output string, params map[string]any) Result
}
