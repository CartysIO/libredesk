package email

import (
	"regexp"
	"strings"
)

var authResultsComment = regexp.MustCompile(`\([^)]*\)`)

// dmarcPass reports whether an Authentication-Results header value records dmarc=pass for header.from
// fromDomain (Clyr fork). Pass the topmost header: the receiving server prepends its own, so headers below
// it can come from the sender.
// ponytail: trusts the topmost header without checking its authserv-id; check it against the mail
// provider's id (for example mx.google.com) if an inbox's provider does not add Authentication-Results.
func dmarcPass(header, fromDomain string) bool {
	fromDomain = strings.ToLower(strings.TrimSpace(fromDomain))
	if fromDomain == "" {
		return false
	}
	header = authResultsComment.ReplaceAllString(strings.ToLower(header), " ")
	parts := strings.Split(header, ";")
	// The first part is the authserv-id; each later one is a method result.
	for _, part := range parts[1:] {
		fields := strings.Fields(part)
		if len(fields) == 0 || fields[0] != "dmarc=pass" {
			continue
		}
		for _, f := range fields[1:] {
			if d, ok := strings.CutPrefix(f, "header.from="); ok && strings.TrimSuffix(d, ".") == fromDomain {
				return true
			}
		}
	}
	return false
}
