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

// Default init policy (valid_format + required_fields) must agree on the extracted document.
func TestRequiredFields_matchesValidFormatOnPreambleFence(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		output string
		want   string // pass | fail
	}{
		{
			name:   "preamble_fence_thanks",
			output: "Sure.\n```json\n{\"message\":\"ok\"}\n```\nThanks.",
			want:   "pass",
		},
		{
			name:   "preamble_inline_json",
			output: `Sure. {"message":"ok"} Thanks.`,
			want:   "pass",
		},
		{
			name:   "trailing_emdash",
			output: `{"message":"ok"} — done`,
			want:   "pass",
		},
		{
			name:   "fence_only",
			output: "```json\n{\"message\":\"ok\"}\n```",
			want:   "pass",
		},
		{
			name:   "missing_field_raw",
			output: `{"foo":1}`,
			want:   "fail",
		},
		{
			name:   "missing_field_fence",
			output: "```json\n{\"foo\":1}\n```",
			want:   "fail",
		},
		{
			name:   "nested_fence_preamble",
			output: "Here:\n```json\n{\"message\":\"ok\",\"meta\":{\"a\":[1,2]}}\n```",
			want:   "pass",
		},
		{
			name:   "two_fences_first_wins",
			output: "```json\n{\"message\":\"ok\"}\n```\nand\n```json\n{\"x\":1}\n```",
			want:   "pass",
		},
		{
			name:   "prose_fence_missing_field",
			output: "Sure.\n```json\n{\"foo\":1}\n```",
			want:   "fail",
		},
	}
	fmtParams := map[string]any{"format": "json"}
	reqParams := map[string]any{"fields": []any{"message"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fmtRes := ValidFormat{}.Run(ctx, tc.output, fmtParams)
			reqRes := RequiredFields{}.Run(ctx, tc.output, reqParams)
			if tc.want == "pass" {
				if fmtRes.Status != "pass" {
					t.Fatalf("valid_format: got %s %s", fmtRes.Status, fmtRes.Message)
				}
				if reqRes.Status != "pass" {
					t.Fatalf("required_fields: got %s %s", reqRes.Status, reqRes.Message)
				}
				return
			}
			if fmtRes.Status != "pass" {
				t.Fatalf("valid_format should still parse extracted JSON, got %s %s", fmtRes.Status, fmtRes.Message)
			}
			if reqRes.Status != "fail" {
				t.Fatalf("required_fields: expected fail for missing message, got %s %s", reqRes.Status, reqRes.Message)
			}
		})
	}
}

func TestParseShallowFields_preambleFence(t *testing.T) {
	raw := "Sure.\n```json\n{\"message\":\"ok\"}\n```\nThanks."
	fields, err := ParseShallowFields(context.Background(), raw)
	if err != nil {
		t.Fatalf("ParseShallowFields: %v", err)
	}
	if _, ok := fields["message"]; !ok {
		t.Fatalf("expected message key, got %#v", fields)
	}
}
