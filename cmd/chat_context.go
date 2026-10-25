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
// V2_9_0_RC9_CLYR9 creates a definition for each, so the agent sidebar shows them.
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
	return cleanContextText(raw, maxChatContextURLLen)
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
