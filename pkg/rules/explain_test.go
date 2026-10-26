package rules

import (
	"testing"
)

func TestExplainRule_mvpRule(t *testing.T) {
	ex, err := ExplainRule("valid_format")
	if err != nil {
		t.Fatal(err)
	}
	if ex.OSS != "pure" {
		t.Fatalf("oss=%q want pure", ex.OSS)
	}
	if ex.Description == "" || ex.Description == "(no catalog description; see doc URL)" {
		t.Fatalf("expected catalog description, got %q", ex.Description)
	}
	if ex.DocURL != ValidatorDocBaseURL+"valid_format" {
		t.Fatalf("doc url=%q", ex.DocURL)
	}
}

func TestExplainRule_unknown(t *testing.T) {
	_, err := ExplainRule("not_a_real_validator_xyz")
	if err == nil {
		t.Fatal("expected error")
	}
}
