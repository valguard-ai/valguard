package validators

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// validateJSONFast is a zero-allocation well-formedness check for a JSON
// document. It is roughly 2x faster than encoding/json.Valid on typical
// 1–4 KB structured outputs because it avoids the state-table dispatch
// loop of encoding/json's scanner.
//
// The validator accepts the same grammar as encoding/json (RFC 8259). It
// does not check duplicate keys or character classes beyond what JSON
// requires (e.g. valid \uXXXX escapes).
//
// Returns nil on success; otherwise an error with the offset of the
// first malformed token.
// ValidateJSONFast is a zero-allocation well-formedness check for JSON (RFC 8259).
func ValidateJSONFast(s string) error {
	return validateJSONFast(s)
}

func validateJSONFast(s string) error {
	i := skipJSONWS(s, 0)
	end, err := validateJSONValue(s, i)
	if err != nil {
		return err
	}
	end = skipJSONWS(s, end)
	if end != len(s) {
		return fmt.Errorf("trailing garbage at offset %d", end)
	}
	return nil
}

func validateJSONValue(s string, i int) (int, error) {
	if i >= len(s) {
		return i, errors.New("unexpected end of JSON")
	}
	c := s[i]
	switch {
	case c == '{':
		return validateJSONObject(s, i)
	case c == '[':
		return validateJSONArray(s, i)
	case c == '"':
		return validateJSONString(s, i)
	case c == 't':
		if i+4 <= len(s) && s[i:i+4] == "true" {
			return i + 4, nil
		}
	case c == 'f':
		if i+5 <= len(s) && s[i:i+5] == "false" {
			return i + 5, nil
		}
	case c == 'n':
		if i+4 <= len(s) && s[i:i+4] == "null" {
			return i + 4, nil
		}
	case c == '-' || (c >= '0' && c <= '9'):
		return validateJSONNumber(s, i)
	}
	return i, fmt.Errorf("unexpected token %q at offset %d", c, i)
}

func validateJSONObject(s string, i int) (int, error) {
	i++ // consume '{'
	i = skipJSONWS(s, i)
	if i < len(s) && s[i] == '}' {
		return i + 1, nil
	}
	for {
		if i >= len(s) || s[i] != '"' {
			return i, errors.New("expected string key")
		}
		end, err := validateJSONString(s, i)
		if err != nil {
			return end, err
		}
		i = skipJSONWS(s, end)
		if i >= len(s) || s[i] != ':' {
			return i, errors.New("expected ':' after key")
		}
		i = skipJSONWS(s, i+1)
		end, err = validateJSONValue(s, i)
		if err != nil {
			return end, err
		}
		i = skipJSONWS(s, end)
		if i >= len(s) {
			return i, errors.New("unexpected end of object")
		}
		if s[i] == ',' {
			i = skipJSONWS(s, i+1)
			continue
		}
		if s[i] == '}' {
			return i + 1, nil
		}
		return i, fmt.Errorf("expected ',' or '}' at offset %d", i)
	}
}

func validateJSONArray(s string, i int) (int, error) {
	i++ // consume '['
	i = skipJSONWS(s, i)
	if i < len(s) && s[i] == ']' {
		return i + 1, nil
	}
	for {
		end, err := validateJSONValue(s, i)
		if err != nil {
			return end, err
		}
		i = skipJSONWS(s, end)
		if i >= len(s) {
			return i, errors.New("unexpected end of array")
		}
		if s[i] == ',' {
			i = skipJSONWS(s, i+1)
			continue
		}
		if s[i] == ']' {
			return i + 1, nil
		}
		return i, fmt.Errorf("expected ',' or ']' at offset %d", i)
	}
}

// validateJSONString consumes a JSON string starting at s[i] (which is the
// opening quote). It checks escape sequences for syntactic validity but
// does not decode them.
func validateJSONString(s string, i int) (int, error) {
	i++ // consume opening "
	for i < len(s) {
		c := s[i]
		if c == '\\' {
			if i+1 >= len(s) {
				return i, errors.New("dangling escape")
			}
			esc := s[i+1]
			switch esc {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				i += 2
			case 'u':
				if i+6 > len(s) {
					return i, errors.New("short \\u escape")
				}
				for k := 0; k < 4; k++ {
					if !isHexDigit(s[i+2+k]) {
						return i + 2 + k, errors.New("invalid \\u escape")
					}
				}
				i += 6
			default:
				return i, fmt.Errorf("invalid escape \\%c", esc)
			}
			continue
		}
		if c == '"' {
			return i + 1, nil
		}
		if c < 0x20 {
			return i, fmt.Errorf("control char in string at offset %d", i)
		}
		i++
	}
	return i, errors.New("unterminated string")
}

func validateJSONNumber(s string, i int) (int, error) {
	j := i
	if s[j] == '-' {
		j++
	}
	if j >= len(s) {
		return j, errors.New("number missing digits")
	}
	if s[j] == '0' {
		j++
	} else if s[j] >= '1' && s[j] <= '9' {
		j++
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
	} else {
		return j, errors.New("number missing digits")
	}
	if j < len(s) && s[j] == '.' {
		j++
		if j >= len(s) || s[j] < '0' || s[j] > '9' {
			return j, errors.New("number missing fractional digits")
		}
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
	}
	if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
		j++
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		if j >= len(s) || s[j] < '0' || s[j] > '9' {
			return j, errors.New("number missing exponent digits")
		}
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
	}
	return j, nil
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// Shallow JSON top-level scanner.
//
// Many validators only inspect top-level fields of a JSON object and never
// recurse into nested arrays/objects. Decoding the full document into
// map[string]any is wasteful in both CPU and allocations (the invoice fixture
// produces ~370 of 499 allocs/op from deep json.Unmarshal alone).
//
// scanTopLevelJSON walks the document once, records the byte span of every
// top-level value, and exposes typed lookups. Nested values are *not* decoded:
// the helpers expose the raw bytes for cases that need them. Strings are
// returned as substrings of the input when there are no escape sequences,
// avoiding any allocation in the common case.

// shallowField represents a top-level value in a JSON object whose raw bytes
// have been located but not decoded.
// ShallowField represents a top-level JSON object value located but not decoded.
type ShallowField struct {
	// kind is the value type:
	//   'o' object, 'a' array, 's' string,
	//   'n' number, 'b' boolean, 'z' null
	kind byte
	// raw is a slice of the original input covering the value.
	// For strings this includes the surrounding quotes.
	raw string
}

// Kind returns the value kind byte ('o','a','s','n','b','z').
func (f ShallowField) Kind() byte { return f.kind }

// Raw returns the raw JSON bytes of the value (slice of the input string).
func (f ShallowField) Raw() string { return f.raw }

// String decodes the JSON string value. Returns false for non-string values.
// The common no-escape path returns a substring with zero allocations.
func (f ShallowField) String() (string, bool) {
	if f.kind != 's' || len(f.raw) < 2 {
		return "", false
	}
	inner := f.raw[1 : len(f.raw)-1]
	if strings.IndexByte(inner, '\\') < 0 {
		return inner, true
	}
	var s string
	if err := json.Unmarshal([]byte(f.raw), &s); err != nil {
		return "", false
	}
	return s, true
}

// Number decodes the JSON number value. Returns false for non-numeric values.
func (f ShallowField) Number() (float64, bool) {
	if f.kind != 'n' {
		return 0, false
	}
	n, err := strconv.ParseFloat(f.raw, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// Bool decodes the JSON boolean value. Returns false for non-boolean values.
func (f ShallowField) Bool() (bool, bool) {
	switch f.raw {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}

// IsNull reports whether the value is JSON null.
func (f ShallowField) IsNull() bool { return f.kind == 'z' }

// ArrayLen counts top-level elements of a JSON array without decoding.
// Returns false for non-array values.
func (f ShallowField) ArrayLen() (int, bool) {
	if f.kind != 'a' {
		return 0, false
	}
	return countArrayElements(f.raw), true
}

// Display returns a string form suitable for human-facing messages or
// stringly-typed comparisons (matches fmt.Sprint of a parsed value).
// For strings this is the decoded value, for numbers/bools/null the raw
// JSON literal, and for containers a compact placeholder.
func (f ShallowField) Display() string {
	switch f.kind {
	case 's':
		if s, ok := f.String(); ok {
			return s
		}
		return f.raw
	case 'n', 'b', 'z':
		return f.raw
	case 'o':
		return "map[]"
	case 'a':
		return "[]"
	}
	return f.raw
}

// scanTopLevelJSON parses the top-level object of a JSON document into a
// map of field name -> shallowField. Nested arrays/objects are recorded as
// raw byte slices without being decoded. Duplicate top-level keys are
// collected in dups (first occurrence is kept in out).
//
// Callers that hold a re-usable destination (e.g. outputCache) should use
// scanTopLevelJSONInto to avoid allocating a fresh map per call.
// ScanTopLevelJSON parses top-level object fields without deep decoding.
func ScanTopLevelJSON(raw string) (map[string]ShallowField, []string, error) {
	return ScanTopLevelJSONInto(raw, nil, nil)
}

// ScanTopLevelJSONInto parses into a reusable destination map.
func ScanTopLevelJSONInto(raw string, dst map[string]ShallowField, dupBuf []string) (map[string]ShallowField, []string, error) {
	return scanTopLevelJSONInto(raw, dst, dupBuf)
}

func scanTopLevelJSON(raw string) (map[string]ShallowField, []string, error) {
	return scanTopLevelJSONInto(raw, nil, nil)
}

// scanTopLevelJSONInto parses into a caller-provided map. When dst is nil a
// fresh map is allocated with a sensible default capacity. dupBuf is reused
// for the duplicate-key slice (passed as nil:nil from non-pooled callers).
// The returned `out` map is always cleared before parsing.
func scanTopLevelJSONInto(raw string, dst map[string]ShallowField, dupBuf []string) (map[string]ShallowField, []string, error) {
	i := skipJSONWS(raw, 0)
	if i >= len(raw) || raw[i] != '{' {
		return nil, nil, errors.New("not a JSON object")
	}
	i++
	out := dst
	if out == nil {
		out = make(map[string]ShallowField, 16)
	} else {
		for k := range out {
			delete(out, k)
		}
	}
	dups := dupBuf[:0]
	i = skipJSONWS(raw, i)
	if i < len(raw) && raw[i] == '}' {
		return out, dups, nil
	}
	for {
		if i >= len(raw) {
			return nil, nil, errors.New("unexpected end of JSON")
		}
		if raw[i] != '"' {
			return nil, nil, errors.New("expected string key")
		}
		key, next, err := readJSONStringValue(raw, i)
		if err != nil {
			return nil, nil, err
		}
		i = skipJSONWS(raw, next)
		if i >= len(raw) || raw[i] != ':' {
			return nil, nil, errors.New("expected ':' after key")
		}
		i = skipJSONWS(raw, i+1)
		kind, end, err := scanJSONValue(raw, i)
		if err != nil {
			return nil, nil, err
		}
		if _, exists := out[key]; exists {
			dups = append(dups, key)
		} else {
			out[key] = ShallowField{kind: kind, raw: raw[i:end]}
		}
		i = skipJSONWS(raw, end)
		if i >= len(raw) {
			return nil, nil, errors.New("unexpected end of JSON")
		}
		switch raw[i] {
		case ',':
			i = skipJSONWS(raw, i+1)
			continue
		case '}':
			return out, dups, nil
		default:
			return nil, nil, fmt.Errorf("expected ',' or '}' at offset %d", i)
		}
	}
}

func skipJSONWS(s string, i int) int {
	for i < len(s) {
		switch s[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

// readJSONStringValue reads a JSON string starting at s[i] (which must be '"').
// Returns the decoded value and the index AFTER the closing quote.
// Common no-escape path returns a substring of s (no allocation).
func readJSONStringValue(s string, i int) (string, int, error) {
	if i >= len(s) || s[i] != '"' {
		return "", i, errors.New("expected string")
	}
	start := i + 1
	j := start
	hasEscape := false
	for j < len(s) {
		c := s[j]
		if c == '\\' {
			hasEscape = true
			if j+1 >= len(s) {
				return "", j, errors.New("dangling escape")
			}
			j += 2
			continue
		}
		if c == '"' {
			if !hasEscape {
				return s[start:j], j + 1, nil
			}
			var out string
			if err := json.Unmarshal([]byte(s[i:j+1]), &out); err != nil {
				return "", j + 1, err
			}
			return out, j + 1, nil
		}
		j++
	}
	return "", j, errors.New("unterminated string")
}

// scanJSONValue identifies the kind and end index (exclusive) of a JSON
// value at s[i:]. Containers are matched without decoding their contents.
func scanJSONValue(s string, i int) (byte, int, error) {
	if i >= len(s) {
		return 0, i, errors.New("unexpected end of JSON value")
	}
	c := s[i]
	switch {
	case c == '"':
		j := i + 1
		for j < len(s) {
			if s[j] == '\\' {
				if j+1 >= len(s) {
					return 0, j, errors.New("dangling escape")
				}
				j += 2
				continue
			}
			if s[j] == '"' {
				return 's', j + 1, nil
			}
			j++
		}
		return 0, i, errors.New("unterminated string")
	case c == '{', c == '[':
		return scanJSONContainer(s, i)
	case c == 't':
		if i+4 <= len(s) && s[i:i+4] == "true" {
			return 'b', i + 4, nil
		}
	case c == 'f':
		if i+5 <= len(s) && s[i:i+5] == "false" {
			return 'b', i + 5, nil
		}
	case c == 'n':
		if i+4 <= len(s) && s[i:i+4] == "null" {
			return 'z', i + 4, nil
		}
	case c == '-' || (c >= '0' && c <= '9'):
		j := i
		if s[j] == '-' {
			j++
		}
		for j < len(s) {
			d := s[j]
			if (d >= '0' && d <= '9') || d == '.' || d == 'e' || d == 'E' || d == '+' || d == '-' {
				j++
				continue
			}
			break
		}
		if j == i || (j == i+1 && s[i] == '-') {
			return 0, i, errors.New("invalid number")
		}
		return 'n', j, nil
	}
	return 0, i, fmt.Errorf("unexpected token %q", c)
}

// scanJSONContainer matches a balanced JSON container starting at s[i].
// s[i] must be '{' or '['. Returns the kind byte and the index AFTER the
// matching closing bracket.
func scanJSONContainer(s string, i int) (byte, int, error) {
	open := s[i]
	closeCh := byte(']')
	kind := byte('a')
	if open == '{' {
		closeCh = '}'
		kind = 'o'
	}
	depth := 1
	j := i + 1
	for j < len(s) {
		c := s[j]
		if c == '"' {
			j++
			for j < len(s) {
				if s[j] == '\\' {
					if j+1 >= len(s) {
						return 0, j, errors.New("dangling escape")
					}
					j += 2
					continue
				}
				if s[j] == '"' {
					j++
					break
				}
				j++
			}
			continue
		}
		switch c {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				if c != closeCh {
					return 0, i, errors.New("mismatched container")
				}
				return kind, j + 1, nil
			}
		}
		j++
	}
	return 0, i, errors.New("unterminated container")
}

// countArrayElements returns the number of top-level elements in a JSON
// array. arrRaw must begin with '[' and end with ']'. Nested containers
// and strings are correctly skipped without being decoded.
func countArrayElements(arrRaw string) int {
	if len(arrRaw) < 2 || arrRaw[0] != '[' {
		return 0
	}
	inner := arrRaw[1 : len(arrRaw)-1]
	count := 0
	depth := 0
	hasElement := false
	j := 0
	for j < len(inner) {
		c := inner[j]
		switch c {
		case ' ', '\t', '\n', '\r':
			j++
			continue
		case '"':
			hasElement = true
			j++
			for j < len(inner) {
				if inner[j] == '\\' {
					if j+1 >= len(inner) {
						return count
					}
					j += 2
					continue
				}
				if inner[j] == '"' {
					j++
					break
				}
				j++
			}
			continue
		case '{', '[':
			hasElement = true
			depth++
			j++
			continue
		case '}', ']':
			depth--
			j++
			continue
		case ',':
			if depth == 0 {
				count++
				hasElement = false
				j++
				continue
			}
			j++
			continue
		default:
			hasElement = true
			j++
		}
	}
	if hasElement {
		count++
	}
	return count
}
