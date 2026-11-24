package validators

import (
	"context"
	"testing"
)

func TestNormalizeJSONDocumentForValidation_fenceAndPreamble(t *testing.T) {
	raw := "Sure.\n```json\n{\"email\":\"a@b.c\",\"name\":\"Ada\"}\n```\nThanks."
	got := NormalizeJSONDocumentForValidation(raw)
	want := `{"email":"a@b.c","name":"Ada"}`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidFormat_jsonTrailingJunk(t *testing.T) {
	out := `{"a":1} — done`
	r := ValidFormat{}.Run(context.Background(), out, map[string]any{"format": "json"})
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s %s", r.Status, r.Message)
	}
}
