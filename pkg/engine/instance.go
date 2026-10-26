package engine

import (
	"github.com/valguard-ai/valguard/pkg/validators"
)

// Instance is a store-free validator binding (no DB UUID types).
type Instance struct {
	ID       string
	Name     string
	Enabled  bool
	Priority int32
	OnFail   string
	Params   map[string]any

	// Pre-resolved at load time when a registry is available.
	Validator validators.Validator
	Pack      string
	Compiled  validators.CompiledRunner

	// ParamsInvalid is set when params JSON could not be parsed at load time.
	ParamsInvalid bool
}

// ResolveInstances binds validators and compiled runners from any registry implementation.
func ResolveInstances(reg ValidatorRegistry, instances []Instance) {
	if reg == nil {
		return
	}
	for i := range instances {
		resolveInstance(reg, &instances[i])
	}
}

func resolveInstance(reg ValidatorRegistry, inst *Instance) {
	if v, ok := reg.Get(inst.Name); ok {
		inst.Validator = v
		if runner, ok := validators.TryCompile(v, inst.Params); ok {
			inst.Compiled = runner
		}
	}
}
