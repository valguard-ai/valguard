package evidence

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const devSecret = "dev-outbound-webhook-hmac-secret"

// Record is the downloadable proof envelope (subset of production schema).
type Record struct {
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"`
	Payload   string `json:"payload"`
}

func platformSecret() (string, error) {
	for _, key := range []string{"EVIDENCE_HMAC_SECRET", "OUTBOUND_WEBHOOK_HMAC_SECRET"} {
		if secret := strings.TrimSpace(os.Getenv(key)); secret != "" {
			if secret == devSecret && strings.EqualFold(strings.TrimSpace(os.Getenv("ENV")), "production") {
				return "", fmt.Errorf("%s must not use the development placeholder in production", key)
			}
			return secret, nil
		}
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ENV")), "production") {
		return "", fmt.Errorf("EVIDENCE_HMAC_SECRET or OUTBOUND_WEBHOOK_HMAC_SECRET is required in production")
	}
	return devSecret, nil
}

func deriveOrgKey(platformSecret, keyID string) []byte {
	mac := hmac.New(sha256.New, []byte(platformSecret))
	_, _ = mac.Write([]byte(keyID))
	return mac.Sum(nil)
}

func hmacHex(key []byte, payload string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(payload))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Verify checks signature over exact payload bytes (matches production verifier).
func Verify(keyID, payload, signature string) error {
	secret, err := platformSecret()
	if err != nil {
		return err
	}
	expected := hmacHex(deriveOrgKey(secret, keyID), payload)
	if !hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature))) {
		return fmt.Errorf("evidence signature mismatch")
	}
	return nil
}

// ParseRecord unmarshals envelope JSON from file or stdin.
func ParseRecord(raw []byte) (Record, error) {
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return Record{}, err
	}
	if rec.KeyID == "" || rec.Payload == "" || rec.Signature == "" {
		return Record{}, fmt.Errorf("envelope must include key_id, payload, and signature")
	}
	return rec, nil
}
