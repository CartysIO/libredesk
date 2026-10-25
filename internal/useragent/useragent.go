// Package useragent reads the browser, operating system and device from a User-Agent header.
// It is a small parser for the common browsers, not a full database (Clyr fork).
package useragent

import (
	"regexp"
	"strings"
)

// Info is what the parser found. A field is empty when the User-Agent does not say.
type Info struct {
	Browser     string // Name and major version, for example "Chrome 126".
	OS          string // Name and version, for example "iOS 17.5".
	DeviceType  string // "desktop", "mobile" or "tablet".
	DeviceModel string // For example "iPhone", "iPad" or "Pixel 7".
}

// maxLen stops a long header from costing work. Real headers are far shorter.
const maxLen = 1024

var (
	// Order matters: Edge, Opera and Samsung Internet also say "Chrome", and Chrome also says "Safari".
	browserRes = []struct {
		name string
		re   *regexp.Regexp
	}{
		{"Edge", regexp.MustCompile(`\b(?:Edg|EdgA|EdgiOS|Edge)/(\d+)`)},
		{"Opera", regexp.MustCompile(`\b(?:OPR|OPiOS)/(\d+)`)},
		{"Samsung Internet", regexp.MustCompile(`\bSamsungBrowser/(\d+)`)},
		{"Firefox", regexp.MustCompile(`\b(?:Firefox|FxiOS)/(\d+)`)},
		// No word boundary, so "HeadlessChrome/147" counts as Chrome too.
		{"Chrome", regexp.MustCompile(`(?:Chrome|CriOS)/(\d+)`)},
		{"Internet Explorer", regexp.MustCompile(`(?:MSIE |Trident/.*rv:)(\d+)`)},
		{"Safari", regexp.MustCompile(`\bVersion/(\d+)[\d.]* (?:Mobile/\S+ )?Safari/`)},
	}
	windowsRe = regexp.MustCompile(`Windows NT (\d+)\.(\d+)`)
	androidRe = regexp.MustCompile(`Android (\d+(?:\.\d+)*)`)
	iosRe     = regexp.MustCompile(`(?:iPhone OS|CPU OS) (\d+)[_.](\d+)`)
	macRe     = regexp.MustCompile(`Mac OS X (\d+)[_.](\d+)`)
	// "Android 13; Pixel 7 Build/..." or "Android 13; SM-S918B)".
	androidModelRe = regexp.MustCompile(`Android [\d.]+; ([^;)]+?)(?: Build/[^;)]*)?[;)]`)
)

// Parse reads ua. It never fails: an unknown User-Agent gives empty fields. DeviceType is "desktop"
// for a non-empty header that does not look mobile.
func Parse(ua string) Info {
	if len(ua) > maxLen {
		ua = ua[:maxLen]
	}
	ua = strings.TrimSpace(ua)
	var info Info
	if ua == "" {
		return info
	}

	for _, b := range browserRes {
		if m := b.re.FindStringSubmatch(ua); m != nil {
			info.Browser = b.name + " " + m[1]
			break
		}
	}

	switch {
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPod"):
		info.OS, info.DeviceType, info.DeviceModel = iosName(ua), "mobile", "iPhone"
	case strings.Contains(ua, "iPad"):
		info.OS, info.DeviceType, info.DeviceModel = iosName(ua), "tablet", "iPad"
	case strings.Contains(ua, "Android"):
		info.OS = "Android"
		if m := androidRe.FindStringSubmatch(ua); m != nil {
			info.OS += " " + m[1]
		}
		// Android phones say "Mobile"; tablets do not.
		info.DeviceType = "tablet"
		if strings.Contains(ua, "Mobile") {
			info.DeviceType = "mobile"
		}
		if m := androidModelRe.FindStringSubmatch(ua); m != nil {
			// "K" is the placeholder that Chrome's reduced User-Agent uses.
			if model := strings.TrimSpace(m[1]); model != "K" && model != "" {
				info.DeviceModel = model
			}
		}
	case strings.Contains(ua, "Windows"):
		info.OS, info.DeviceType = windowsName(ua), "desktop"
	case strings.Contains(ua, "Macintosh"), strings.Contains(ua, "Mac OS X"):
		info.OS, info.DeviceType = "macOS", "desktop"
		// Chrome and Safari freeze the version at 10.15.7, so it is shown only when it is not that.
		if m := macRe.FindStringSubmatch(ua); m != nil && !(m[1] == "10" && m[2] == "15") {
			info.OS += " " + m[1] + "." + m[2]
		}
	case strings.Contains(ua, "CrOS"):
		info.OS, info.DeviceType = "ChromeOS", "desktop"
	case strings.Contains(ua, "Linux"), strings.Contains(ua, "X11"):
		info.OS, info.DeviceType = "Linux", "desktop"
	default:
		info.DeviceType = "desktop"
	}
	return info
}

func iosName(ua string) string {
	if m := iosRe.FindStringSubmatch(ua); m != nil {
		return "iOS " + m[1] + "." + m[2]
	}
	return "iOS"
}

func windowsName(ua string) string {
	m := windowsRe.FindStringSubmatch(ua)
	if m == nil {
		return "Windows"
	}
	// Windows 11 also says "Windows NT 10.0", so the two cannot be told apart from the header.
	switch m[1] + "." + m[2] {
	case "10.0":
		return "Windows 10"
	case "6.3":
		return "Windows 8.1"
	case "6.2":
		return "Windows 8"
	case "6.1":
		return "Windows 7"
	}
	return "Windows"
}
