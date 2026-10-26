package report

import (
	"encoding/json"
	"strings"
	"testing"
)

func FuzzLoadJSONL(f *testing.F) {
	f.Add([]byte(`{"output":"{}"}` + "\n"))
	f.Add([]byte("not json"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 4<<20 {
			return
		}
		_, _ = LoadJSONL(strings.NewReader(string(raw)))
	})
}

func FuzzFixtureJSON(f *testing.F) {
	f.Add([]byte(`{"name":"x","policy":{"version":1,"name":"p","mode":"enforce","rules":[]},"output":"{}"}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 4<<20 {
			return
		}
		// Exercises the same JSON shape as conformance fixtures (policy + output fields).
		var fx struct {
			Policy struct {
				Version int `json:"version"`
			} `json:"policy"`
			Output string `json:"output"`
		}
		_ = json.Unmarshal(raw, &fx)
	})
}
