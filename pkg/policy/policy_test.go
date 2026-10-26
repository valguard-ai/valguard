package policy

import "testing"

func TestParse_rejectsNonMVPRule(t *testing.T) {
	_, err := Parse([]byte(`version: 1
name: x
rules:
  - id: sql_injection
    on_fail: block
`))
	if err == nil {
		t.Fatal("expected error for non-MVP rule id")
	}
}

func TestParse_rejectsReask(t *testing.T) {
	_, err := Parse([]byte(`version: 1
name: x
rules:
  - id: valid_format
    on_fail: reask
`))
	if err == nil {
		t.Fatal("expected error")
	}
}
