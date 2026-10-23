package evidence

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestVerifyRoundTrip(t *testing.T) {
	t.Setenv("ENV", "development")
	payload := `{"event":"test"}`
	keyID := "org-test"
	secret, err := platformSecret()
	if err != nil {
		t.Fatal(err)
	}
	orgKey := deriveOrgKey(secret, keyID)
	mac := hmac.New(sha256.New, orgKey)
	_, _ = mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	sigHeader := "sha256=" + sig
	if err := Verify(keyID, payload, sigHeader); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	rec := Record{KeyID: keyID, Signature: sigHeader, Payload: payload}
	raw, _ := json.Marshal(rec)
	parsed, err := ParseRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(parsed.KeyID, parsed.Payload, parsed.Signature); err != nil {
		t.Fatalf("Verify parsed: %v", err)
	}
}
