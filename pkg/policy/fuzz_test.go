package policy

import "testing"

func FuzzParse(f *testing.F) {
	f.Add([]byte("version: 1\nname: fuzz\nmode: enforce\nrules: []\n"))
	f.Add([]byte("not yaml"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > MaxPolicyBytes {
			return
		}
		_, _ = Parse(raw)
	})
}
