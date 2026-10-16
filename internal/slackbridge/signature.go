package slackbridge

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// maxRequestAge is how old a signed Slack request may be. Slack recommends five minutes against replays.
const maxRequestAge = 5 * time.Minute

// VerifySignature checks a Slack request signature: X-Slack-Signature must equal
// "v0=" + hex(HMAC-SHA256(secret, "v0:" + timestamp + ":" + body)), and the timestamp must be recent.
func VerifySignature(secret, timestamp string, body []byte, signature string, now time.Time) bool {
	if secret == "" || timestamp == "" || signature == "" {
		return false
	}
	sec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	age := now.Sub(time.Unix(sec, 0))
	if age > maxRequestAge || age < -maxRequestAge {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("v0:" + timestamp + ":"))
	mac.Write(body)
	want := "v0=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(want), []byte(signature))
}
