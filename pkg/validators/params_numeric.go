package validators

import (
	"strconv"
)

// IntFromParam reads an int from YAML/JSON-decoded params (float64, int, or string).
func IntFromParam(v any, def int) int {
	return intFromParam(v, def)
}

func intFromParam(v any, def int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return def
	}
}

func floatFromParam(v any, def float64) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return def
	}
}

// ShallowToFloat extracts a numeric value from a shallow JSON field.
func ShallowToFloat(v ShallowField) float64 {
	return shallowToFloat(v)
}

func shallowToFloat(v ShallowField) float64 {
	switch v.Kind() {
	case 'n':
		if n, ok := v.Number(); ok {
			return n
		}
	case 's':
		if s, ok := v.String(); ok {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		}
	}
	return 0
}
