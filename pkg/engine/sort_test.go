package engine

import (
	"context"
	"testing"

	"github.com/valguard-ai/valguard/pkg/validators"
)

func TestSortInstancesForFailFast_ordersCheapBeforeExpensive(t *testing.T) {
	instances := []Instance{
		{Name: "fact_match", Priority: 1, Enabled: true, OnFail: "block"},
		{Name: "max_length", Priority: 99, Enabled: true, OnFail: "block", Params: map[string]any{"max_chars": float64(3)}},
	}
	SortInstancesForFailFast(instances)
	if instances[0].Name != "max_length" {
		t.Fatalf("expected max_length first, got %s", instances[0].Name)
	}
}

func TestRunForPlan_StopOnFirstEnforce_skipsExpensiveAfterCheapBlock(t *testing.T) {
	reg := validators.NewRegistry(validators.MaxLength{})
	instances := []Instance{
		{Name: "json_schema_validate", Enabled: true, OnFail: "block", Priority: 1},
		{Name: "max_length", Enabled: true, OnFail: "block", Priority: 99, Params: map[string]any{"max_chars": float64(2)}},
	}
	ResolveInstances(reg, instances)
	SortInstancesForFailFast(instances)

	out := RunForPlan(context.Background(), reg, instances, "abcd", validators.RunMeta{}, RunOptions{StopOnFirstEnforce: true})
	defer ReleaseRun(out)
	if !out.Blocked {
		t.Fatal("expected block from max_length")
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected fail-fast to run only first validator, got %d results", len(out.Results))
	}
	if out.Results[0].Status != "fail" {
		t.Fatalf("expected max_length fail, got %s", out.Results[0].Status)
	}
}
