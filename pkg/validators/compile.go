package validators

import "context"

// CompiledRunner is a params-bound validator run function.
type CompiledRunner func(ctx context.Context, output string) Result

// Compilable validators can pre-bind params at instance load time.
type Compilable interface {
	Validator
	Compile(params map[string]any) CompiledRunner
}

// TryCompile returns a bound runner when the validator supports Compile.
func TryCompile(v Validator, params map[string]any) (CompiledRunner, bool) {
	if c, ok := v.(Compilable); ok {
		return c.Compile(params), true
	}
	return nil, false
}
