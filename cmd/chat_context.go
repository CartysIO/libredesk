package main

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/useragent"
)

// Limits for the visitor context a chat widget sends. Every value comes from the visitor's browser,
// so it is untrusted text.
const (
	maxChatContextURLLen      = 2048
	maxChatContextUALen       = 512
	maxChatContextAppLen      = 100
	maxChatContextTimezoneLen = 64
)

// The conversation attribute keys that hold the visitor context (Clyr fork). The migration
// V2_9_0_RC10_CLYR1 creates a definition for each, so the agent sidebar shows them.
const (
	chatAttrBrowser    = "chat_browser"
	chatAttrOS         = "chat_operating_system"
	chatAttrDeviceType = "chat_device_type"
	chatAttrDeviceMdl  = "chat_device_model"
	chatAttrPageURL    = "chat_page_url"
	chatAttrUserAgent  = "chat_user_agent"
	chatAttrTimezone   = "chat_user_timezone"
	chatAttrApp        = "chat_app"
)

// chatContextReq is what the widget sends with chat init. The widget reads the page URL from the
// host page, and the timezone from the browser. The browser, OS and device come from the User-Agent header.
type chatContextReq struct {
	PageURL  string `json:"page_url"`
	Timezone string `json:"timezone"`
	App      string `json:"app"`
}

// ianaTimezoneRe accepts names like "Europe/Berlin", "America/Argentina/Buenos_Aires", "Etc/GMT+5" and "UTC".
var ianaTimezoneRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+\-]*(?:/[A-Za-z0-9_+\-]+){0,2}$`)

// buildChatContextAttributes turns the User-Agent header and the widget's context into conversation
// custom attributes. A value that is empty or invalid is left out.
func buildChatContextAttributes(userAgent string, c chatContextReq) map[string]any {
	attrs := map[string]any{}
	set := func(key, value string) {
		if value != "" {
			attrs[key] = value
		}
	}

	ua := cleanContextText(userAgent, maxChatContextUALen)
	info := useragent.Parse(ua)
	set(chatAttrBrowser, info.Browser)
	set(chatAttrOS, info.OS)
	set(chatAttrDeviceType, info.DeviceType)
	set(chatAttrDeviceMdl, info.DeviceModel)
	set(chatAttrUserAgent, ua)
	set(chatAttrPageURL, cleanPageURL(c.PageURL))
	if tz := strings.TrimSpace(c.Timezone); len(tz) <= maxChatContextTimezoneLen && ianaTimezoneRe.MatchString(tz) {
		set(chatAttrTimezone, tz)
	}
	set(chatAttrApp, cleanContextText(c.App, maxChatContextAppLen))
	return attrs
}

// mergeChatContext adds the context attributes to attrs. An attribute that attrs already has (set
// from the pre-chat form) is kept.
func mergeChatContext(attrs, context map[string]any) map[string]any {
	if attrs == nil {
		attrs = map[string]any{}
	}
	for k, v := range context {
		if _, exists := attrs[k]; !exists {
			attrs[k] = v
		}
	}
	return attrs
}

// cleanPageURL keeps only an http or https URL, and cuts it to the length limit.
func cleanPageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	// A fragment, userinfo, credential-like query parameter or token-like path segment can carry
	// secrets, so none is stored.
	u.User = nil
	u.Fragment, u.RawFragment = "", ""
	u.RawQuery = dropSensitiveQuery(u.RawQuery)
	u.Path, u.RawPath = redactTokenSegments(u.Path), ""
	return cleanContextText(u.String(), maxChatContextURLLen)
}

var (
	// sensitiveQueryWords are matched against the words of a parameter name ("x-api-key" -> x, api, key).
	sensitiveQueryWords = map[string]bool{
		"token": true, "secret": true, "password": true, "passwd": true, "pwd": true, "auth": true,
		"signature": true, "sig": true, "session": true, "sessionid": true, "credential": true,
		"credentials": true, "jwt": true, "key": true, "apikey": true, "code": true, "otp": true,
		"state": true, "nonce": true,
	}
	// sensitiveQueryParts also match inside a word ("authtoken", "oauth2Token", "XAPIKey"). They are
	// specific enough not to hit ordinary names such as author, keyword or ticket.
	sensitiveQueryParts = []string{"token", "apikey", "secret", "password", "passwd", "jwt"}
	// sensitiveQueryNames are whole parameter names that carry a secret but have no sensitive word.
	sensitiveQueryNames = map[string]bool{"__clerk_ticket": true}

	uuidSegmentRe  = regexp.MustCompile(`^(?:[A-Za-z]+_)*[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	tokenSegmentRe = regexp.MustCompile(`^[A-Za-z0-9_.~-]{24,}$`)
	hexSegmentRe   = regexp.MustCompile(`^[0-9a-fA-F]{32,}$`)
)

// dropSensitiveQuery removes the query parameters whose names look like credentials. The others keep their order.
func dropSensitiveQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	kept := make([]string, 0)
	for _, pair := range strings.Split(rawQuery, "&") {
		name, _, _ := strings.Cut(pair, "=")
		if unescaped, err := url.QueryUnescape(name); err == nil {
			name = unescaped
		}
		if pair != "" && !isSensitiveQueryName(name) {
			kept = append(kept, pair)
		}
	}
	return strings.Join(kept, "&")
}

func isSensitiveQueryName(name string) bool {
	if sensitiveQueryNames[strings.ToLower(name)] {
		return true
	}
	for _, word := range nameWords(name) {
		if sensitiveQueryWords[word] {
			return true
		}
	}
	lower := strings.ToLower(name)
	for _, part := range sensitiveQueryParts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

// nameWords splits a parameter name on separators and camelCase into lowercase words.
func nameWords(name string) []string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	runes := []rune(name)
	for i, r := range runes {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			flush()
		case unicode.IsUpper(r) && i > 0 && unicode.IsLower(runes[i-1]):
			flush()
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return words
}

// redactTokenSegments replaces path segments that look like random secrets (a reset or invite
// token) with "redacted". Slugs, UUIDs and prefixed UUIDs such as team_<uuid> stay, because agents
// use them to find the page or record.
func redactTokenSegments(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if looksLikeToken(seg) {
			segments[i] = "redacted"
		}
	}
	return strings.Join(segments, "/")
}

// looksLikeToken is true for a segment of 24 or more URL-safe characters that is either a long hex
// string of digits and letters (not a UUID) or mixes upper case, lower case and digits, as base64 and random tokens do.
// A lowercase slug such as customer-2024-quarterly-summary is not a token.
func looksLikeToken(seg string) bool {
	if !tokenSegmentRe.MatchString(seg) || uuidSegmentRe.MatchString(seg) {
		return false
	}
	if hexSegmentRe.MatchString(seg) {
		return strings.ContainsAny(seg, "0123456789") && strings.ContainsAny(strings.ToLower(seg), "abcdef")
	}
	var upper, lower, digit int
	for _, r := range seg {
		switch {
		case unicode.IsUpper(r):
			upper++
		case unicode.IsLower(r):
			lower++
		case unicode.IsDigit(r):
			digit++
		}
	}
	return upper >= 2 && lower >= 2 && digit >= 2
}

// cleanContextText removes control characters and cuts s to at most max bytes without splitting a character.
func cleanContextText(s string, max int) string {
	s = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == utf8.RuneError {
			return -1
		}
		return r
	}, s))
	if len(s) <= max {
		return s
	}
	s = s[:max]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
