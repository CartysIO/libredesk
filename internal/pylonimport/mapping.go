package pylonimport

import (
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Libredesk status names (schema.sql and migration v2.9.0-rc.9.clyr.4).
const (
	StatusOpen       = "Open"
	StatusOnYou      = "On you"
	StatusOnCustomer = "On customer"
	StatusSnoozed    = "Snoozed"
	StatusResolved   = "Resolved"
)

// MapStatus maps a Pylon issue state to a Libredesk status name. Unknown states become Open.
func MapStatus(state string) string {
	switch state {
	case "closed":
		return StatusResolved
	case "waiting_on_customer":
		return StatusOnCustomer
	case "on_hold":
		return StatusSnoozed
	case "waiting_on_you":
		return StatusOnYou
	default:
		return StatusOpen
	}
}

// KeepTag reports whether a Pylon tag is copied. The loop-* tags were the pylon-loop automation's state machine.
func KeepTag(tag string) bool {
	t := strings.TrimSpace(tag)
	return t != "" && !strings.HasPrefix(strings.ToLower(t), "loop-")
}

// QuestionType is the question_type custom field as text: multi-select values joined by ", ".
func QuestionType(fields map[string]CustomField) string {
	f, ok := fields["question_type"]
	if !ok {
		return ""
	}
	var vals []string
	for _, v := range append([]string{f.Value}, f.Values...) {
		v = strings.TrimSpace(v)
		if v != "" && !slices.Contains(vals, v) {
			vals = append(vals, v)
		}
	}
	return strings.Join(vals, ", ")
}

// OrgExternalID is the Libredesk organization external_id of a Pylon account: the Clyr team id
// (external id label clyr_team_id, else the teamid custom field), else the Pylon account id.
func OrgExternalID(a Account) string {
	for _, e := range a.ExternalIDs {
		if e.Label == "clyr_team_id" && strings.TrimSpace(e.ExternalID) != "" {
			return strings.TrimSpace(e.ExternalID)
		}
	}
	if f, ok := a.CustomFields["teamid"]; ok && strings.TrimSpace(f.Value) != "" {
		return strings.TrimSpace(f.Value)
	}
	return a.ID
}

// SplitName splits a display name into first and last name. An empty name falls back to the email's local part.
func SplitName(name, email string) (string, string) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	if name == "" {
		return "Unknown", ""
	}
	first, last, _ := strings.Cut(name, " ")
	return clip(first, 140), clip(last, 140)
}

// Titled capitalizes the first letter of each word: "jaleel" becomes "Jaleel".
func Titled(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r, size := utf8.DecodeRuneInString(w)
		words[i] = string(unicode.ToUpper(r)) + w[size:]
	}
	return strings.Join(words, " ")
}

// NormalizeEmail lowercases and trims an email address.
func NormalizeEmail(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}

var subjectPrefix = regexp.MustCompile(`(?i)^\s*((re|fw|fwd|aw|sv)\s*(\[\d+\])?\s*:\s*)+`)

// NormalizeSubject lowercases a subject and drops reply and forward prefixes and repeated spaces.
func NormalizeSubject(s string) string {
	s = subjectPrefix.ReplaceAllString(s, "")
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// DuplicateWindow is how far apart a Pylon issue and a Libredesk conversation may start and still count as the same email.
const DuplicateWindow = 24 * time.Hour

// SameThread reports whether a Libredesk conversation looks like the same email thread as a Pylon issue,
// for issues whose Message-IDs did not match: same contact email, same subject, started within DuplicateWindow.
func SameThread(issueEmail, issueSubject string, issueAt time.Time, convEmail, convSubject string, convAt time.Time) bool {
	if NormalizeEmail(issueEmail) == "" || NormalizeEmail(issueEmail) != NormalizeEmail(convEmail) {
		return false
	}
	if s := NormalizeSubject(issueSubject); s == "" || s != NormalizeSubject(convSubject) {
		return false
	}
	d := issueAt.Sub(convAt)
	return d <= DuplicateWindow && d >= -DuplicateWindow
}

// ReferenceFloor is the first reference number new Libredesk conversations get after the import:
// at least floor, and above every Pylon number.
func ReferenceFloor(floor, maxPylonNumber int) int {
	return max(floor, maxPylonNumber+1)
}

// Renumber is one change of an existing Libredesk conversation's reference number.
type Renumber struct {
	ConversationID int
	From           string
	// To is the Pylon number for a conversation that a Pylon issue is merged into, else empty: the next sequence value.
	To string
}

// ExistingConversation is a Libredesk conversation that was not created by this import.
type ExistingConversation struct {
	ID              int
	ReferenceNumber string
	Subject         string
	ContactEmail    string
	CreatedAt       time.Time
	// Merged is set when a Pylon issue was merged into this conversation on an earlier run.
	Merged bool
}

// PlanRenumbers lists the existing conversations whose reference number must change so that every Pylon issue
// can keep its number and every number up to the highest Pylon number means a Pylon issue. A conversation that a
// Pylon issue is merged into takes that issue's number. Any other conversation numbered 1 to maxPylonNumber moves
// to the sequence (To is empty), also where Pylon has a gap. A conversation merged on an earlier run is left alone.
// The sequence moves are listed first, so the numbers they free are free before a merge target takes one.
func PlanRenumbers(existing []ExistingConversation, maxPylonNumber int, mergedInto map[int]int) []Renumber {
	var moves, merges []Renumber
	for _, c := range existing {
		if c.Merged {
			continue
		}
		if num, ok := mergedInto[c.ID]; ok {
			if want := strconv.Itoa(num); c.ReferenceNumber != want {
				merges = append(merges, Renumber{ConversationID: c.ID, From: c.ReferenceNumber, To: want})
			}
			continue
		}
		if n, err := strconv.Atoi(strings.TrimSpace(c.ReferenceNumber)); err == nil && n >= 1 && n <= maxPylonNumber {
			moves = append(moves, Renumber{ConversationID: c.ID, From: c.ReferenceNumber})
		}
	}
	return append(moves, merges...)
}

var imgSrc = regexp.MustCompile(`(?i)<img\b[^>]*?\bsrc\s*=\s*("([^"]*)"|'([^']*)')`)

// InlineFileURLs returns the file URLs that the HTML shows inline in an <img>, matched without their signature,
// mapped to the exact src text in the HTML.
func InlineFileURLs(body string, fileURLs []string) map[string]string {
	byBase := map[string]string{}
	for _, u := range fileURLs {
		base, _, _ := strings.Cut(u, "?")
		byBase[base] = u
	}
	out := map[string]string{}
	for _, m := range imgSrc.FindAllStringSubmatch(body, -1) {
		src := m[2] + m[3]
		base, _, _ := strings.Cut(html.UnescapeString(src), "?")
		if u, ok := byBase[base]; ok {
			out[u] = src
		}
	}
	return out
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8Start(s[n]) {
		n--
	}
	return s[:n]
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }
