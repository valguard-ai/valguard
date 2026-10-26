package validators

import (
	"context"
	"testing"
)

type stubValidator struct {
	name string
}

func (s stubValidator) Name() string { return s.name }

func (s stubValidator) Run(context.Context, string, map[string]any) Result {
	return Result{Status: "pass"}
}

func TestRegistry_Get_jsonValidityAlias(t *testing.T) {
	r := NewRegistry(stubValidator{name: "valid_format"})
	if _, ok := r.Get("json_validity"); !ok {
		t.Fatal("expected json_validity to resolve to valid_format")
	}
}

func TestRegistry_Names_sorted(t *testing.T) {
	r := NewRegistry(
		stubValidator{name: "zebra"},
		stubValidator{name: "alpha"},
	)
	names := r.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "zebra" {
		t.Fatalf("unexpected order: %v", names)
	}
}
