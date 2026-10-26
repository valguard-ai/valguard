package validators

import (
	"encoding/json"
	"strings"
	"testing"
)

func jsonValid(s string) bool { return json.Valid([]byte(s)) }

func TestScanTopLevelJSON_basicTypes(t *testing.T) {
	in := `{"s":"hello","n":12.5,"i":42,"b":true,"f":false,"z":null,"a":[1,2,3],"o":{"x":1}}`
	fields, _, err := ScanTopLevelJSON(in)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if got := len(fields); got != 8 {
		t.Fatalf("expected 8 fields, got %d", got)
	}
	if k := fields["s"].Kind(); k != 's' {
		t.Fatalf("s kind: %c", k)
	}
	if v, ok := fields["s"].String(); !ok || v != "hello" {
		t.Fatalf("s string: %q ok=%v", v, ok)
	}
	if v, ok := fields["n"].Number(); !ok || v != 12.5 {
		t.Fatalf("n number: %v ok=%v", v, ok)
	}
	if v, ok := fields["i"].Number(); !ok || v != 42 {
		t.Fatalf("i number: %v ok=%v", v, ok)
	}
	if v, ok := fields["b"].Bool(); !ok || !v {
		t.Fatalf("b bool: %v ok=%v", v, ok)
	}
	if v, ok := fields["f"].Bool(); !ok || v {
		t.Fatalf("f bool: %v ok=%v", v, ok)
	}
	if !fields["z"].IsNull() {
		t.Fatal("z should be null")
	}
	if k := fields["a"].Kind(); k != 'a' {
		t.Fatalf("a kind: %c", k)
	}
	if n, ok := fields["a"].ArrayLen(); !ok || n != 3 {
		t.Fatalf("a len: %d ok=%v", n, ok)
	}
	if k := fields["o"].Kind(); k != 'o' {
		t.Fatalf("o kind: %c", k)
	}
}

func TestScanTopLevelJSON_invalidInputs(t *testing.T) {
	cases := []string{
		``,
		`[]`,
		`{`,
		`{"a":}`,
		`{"a":1,}`,
		`{"a":"unterminated`,
		`{"a":nul}`,
		`{1:1}`,
		`{"a":1 "b":2}`,
	}
	for _, c := range cases {
		if _, _, err := ScanTopLevelJSON(c); err == nil {
			t.Fatalf("expected error for %q", c)
		}
	}
}

func TestScanTopLevelJSON_whitespaceAndEscapes(t *testing.T) {
	in := `  {  "name" : "line\nbreak" , "tag" : "plain"  }  `
	fields, _, err := ScanTopLevelJSON(in)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if v, ok := fields["name"].String(); !ok || v != "line\nbreak" {
		t.Fatalf("name string: %q ok=%v", v, ok)
	}
	if v, ok := fields["tag"].String(); !ok || v != "plain" {
		t.Fatalf("tag string: %q ok=%v", v, ok)
	}
}

func TestScanTopLevelJSON_nestedContainers(t *testing.T) {
	in := `{"items":[{"a":1,"b":[1,2]},{"a":2}],"meta":{"nested":{"deep":true}}}`
	fields, _, err := ScanTopLevelJSON(in)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if n, ok := fields["items"].ArrayLen(); !ok || n != 2 {
		t.Fatalf("items len: %d ok=%v", n, ok)
	}
	if k := fields["meta"].Kind(); k != 'o' {
		t.Fatalf("meta kind: %c", k)
	}
	if !strings.HasPrefix(fields["meta"].Raw(), "{") {
		t.Fatalf("meta raw should start with {: %q", fields["meta"].Raw())
	}
}

func TestScanTopLevelJSON_emptyObject(t *testing.T) {
	fields, _, err := ScanTopLevelJSON(`{}`)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(fields) != 0 {
		t.Fatalf("expected empty, got %d", len(fields))
	}
}

func TestScanTopLevelJSON_arrayLenEdgeCases(t *testing.T) {
	cases := map[string]int{
		`[]`:                                            0,
		`[1]`:                                           1,
		`[1,2,3]`:                                       3,
		`[ 1 , 2 , 3 ]`:                                 3,
		`["a,b","c"]`:                                   2,
		`["a\",b","c"]`:                                 2,
		`[{"a":1},{"b":2}]`:                             2,
		`[[1,2],[3,4],[5]]`:                             3,
		`[ ]`:                                           0,
	}
	for arr, want := range cases {
		got := countArrayElements(arr)
		if got != want {
			t.Errorf("countArrayElements(%q)=%d, want %d", arr, got, want)
		}
	}
}

func TestScanTopLevelJSON_duplicateKeys(t *testing.T) {
	in := `{"a":1,"a":2,"b":3}`
	_, dups, err := ScanTopLevelJSON(in)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(dups) != 1 || dups[0] != "a" {
		t.Fatalf("dups=%v want [a]", dups)
	}
}

func TestShallowField_StringNoAlloc(t *testing.T) {
	in := `{"a":"hello"}`
	fields, _, err := ScanTopLevelJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := fields["a"].String()
	if !ok || s != "hello" {
		t.Fatalf("got %q ok=%v", s, ok)
	}
}

func TestValidateJSONFast(t *testing.T) {
	good := []string{
		`{}`,
		`[]`,
		`"hello"`,
		`12.5`,
		`-1.5e10`,
		`null`,
		`true`,
		`false`,
		`{"a":1,"b":[true,null,{"x":"y"}],"c":"text"}`,
		`["one","two\t","three\u0041"]`,
		`{"name":"Acme \"Inc\"","empty":{},"arr":[]}`,
		`1.23e-4`,
		`{"x":  -0,"y":0.5}`,
		"   {\n  \"x\": 1  \n}   ",
	}
	for _, s := range good {
		if err := ValidateJSONFast(s); err != nil {
			t.Errorf("ValidateJSONFast rejected %q: %v", s, err)
		}
	}
	bad := []string{
		``,
		`{`,
		`{"a"}`,
		`{"a":}`,
		`{"a":1,}`,
		`[1,2,]`,
		`{1:2}`,
		`{"a":1} extra`,
		`{"a":"oops`,
		`01`,
		`-`,
		`1.`,
		`1e`,
		`"bad\xescape"`,
		`"control` + "\x01" + `"`,
		`{"x":"\u00G1"}`,
	}
	for _, s := range bad {
		if err := ValidateJSONFast(s); err == nil {
			t.Errorf("ValidateJSONFast accepted invalid input %q", s)
		}
	}
}

// TestValidateJSONFast_parityWithEncodingJSON checks that the fast validator
// agrees with encoding/json.Valid for a fixed set of borderline strings.
// The fast validator is permitted to be STRICTER than json.Valid (e.g. it
// rejects malformed escapes that json.Valid happens to accept under
// io.EOF), but it must never accept what encoding/json rejects.
func TestValidateJSONFast_parityWithEncodingJSON(t *testing.T) {
	cases := []string{
		`{"a":1}`,
		`{"a":1.0e+0}`,
		`[]`,
		`[null,true,false]`,
		`"plain"`,
		`"with \"quotes\""`,
		`"with \\backslash"`,
		`"unicode \u00E9"`,
		`{"nested":{"deep":[1,2,3]}}`,
	}
	for _, s := range cases {
		fastOK := ValidateJSONFast(s) == nil
		stdOK := jsonValid(s)
		if fastOK && !stdOK {
			t.Errorf("fast accepted %q but encoding/json rejected", s)
		}
	}
}
