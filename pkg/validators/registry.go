package validators

import "sort"

// Registry maps validator name → implementation. Safe for read-only use after construction.
type Registry struct {
	validators map[string]Validator
}

// NewRegistry builds a registry from zero or more validators (duplicate names: last wins).
func NewRegistry(vs ...Validator) *Registry {
	r := &Registry{validators: make(map[string]Validator, len(vs))}
	for _, v := range vs {
		r.Register(v)
	}
	return r
}

// Register adds or replaces a validator by Name().
func (r *Registry) Register(v Validator) {
	if v == nil || r == nil {
		return
	}
	name := v.Name()
	if name == "" {
		return
	}
	if r.validators == nil {
		r.validators = make(map[string]Validator)
	}
	r.validators[name] = v
}

// Get resolves a validator by catalog name, including legacy aliases.
func (r *Registry) Get(name string) (Validator, bool) {
	// Backward compatibility: legacy agents/templates use json_validity.
	if name == "json_validity" {
		name = "valid_format"
	}
	if r == nil {
		return nil, false
	}
	v, ok := r.validators[name]
	return v, ok
}

// Names returns registered validator names in sorted order.
func (r *Registry) Names() []string {
	if r == nil || len(r.validators) == 0 {
		return nil
	}
	names := make([]string, 0, len(r.validators))
	for k := range r.validators {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
