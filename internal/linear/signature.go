package linear

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// maxWebhookAge is how far webhookTimestamp may be from now. Linear recommends one minute against replays.
const maxWebhookAge = time.Minute

// VerifySignature checks the Linear-Signature header: hex(HMAC-SHA256(secret, raw body)).
func VerifySignature(secret string, body []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	got, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), got)
}

// freshTimestamp reports whether webhookTimestamp (Unix milliseconds) is within maxWebhookAge of now.
func freshTimestamp(ms int64, now time.Time) bool {
	if ms <= 0 {
		return false
	}
	age := now.Sub(time.UnixMilli(ms))
	return age <= maxWebhookAge && age >= -maxWebhookAge
}
