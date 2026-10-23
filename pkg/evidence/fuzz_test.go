package evidence

import "testing"

func FuzzParseRecord(f *testing.F) {
	f.Add([]byte(`{"key_id":"k","payload":"p","signature":"sha256=ab"}`))
	f.Add([]byte("{bad"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 1<<20 {
			return
		}
		_, _ = ParseRecord(raw)
	})
}
