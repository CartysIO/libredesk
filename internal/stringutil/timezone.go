package stringutil

import (
	"strings"
	"time"
)

// IsValidTimezone reports whether tz is an IANA timezone name. "Local" is rejected because Go
// resolves it to the host zone but Postgres' AT TIME ZONE does not accept it.
func IsValidTimezone(tz string) bool {
	tz = strings.TrimSpace(tz)
	if tz == "" || tz == "Local" {
		return false
	}
	if !isIANATimezoneSpelling(tz) {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

// isIANATimezoneSpelling requires each path segment to start with an uppercase letter, matching
// IANA zone naming and Postgres AT TIME ZONE (Go 1.27+ may load case-insensitive aliases).
func isIANATimezoneSpelling(tz string) bool {
	parts := strings.Split(tz, "/")
	for _, part := range parts {
		if part == "" {
			return false
		}
		c := part[0]
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

// NormalizeTimezone returns tz if valid, otherwise UTC.
func NormalizeTimezone(tz string) string {
	tz = strings.TrimSpace(tz)
	if IsValidTimezone(tz) {
		return tz
	}
	return "UTC"
}
