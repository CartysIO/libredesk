package useragent

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want Info
	}{
		{"chrome mac", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
			Info{"Chrome 126", "macOS", "desktop", ""}},
		{"safari mac", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
			Info{"Safari 17", "macOS", "desktop", ""}},
		{"edge windows", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.2535.51",
			Info{"Edge 125", "Windows 10", "desktop", ""}},
		{"firefox linux", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/127.0",
			Info{"Firefox 127", "Linux", "desktop", ""}},
		{"opera", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 OPR/111.0.0.0",
			Info{"Opera 111", "Windows 10", "desktop", ""}},
		{"safari iphone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
			Info{"Safari 17", "iOS 17.5", "mobile", "iPhone"}},
		{"chrome ios", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/126.0.6478.54 Mobile/15E148 Safari/604.1",
			Info{"Chrome 126", "iOS 17.5", "mobile", "iPhone"}},
		{"ipad", "Mozilla/5.0 (iPad; CPU OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",
			Info{"Safari 16", "iOS 16.6", "tablet", "iPad"}},
		{"chrome android phone", "Mozilla/5.0 (Linux; Android 14; Pixel 7 Build/UP1A.231105.001) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.6478.71 Mobile Safari/537.36",
			Info{"Chrome 126", "Android 14", "mobile", "Pixel 7"}},
		{"chrome android reduced", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36",
			Info{"Chrome 126", "Android 10", "mobile", ""}},
		{"samsung tablet", "Mozilla/5.0 (Linux; Android 13; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/24.0 Chrome/117.0.0.0 Safari/537.36",
			Info{"Samsung Internet 24", "Android 13", "tablet", "SM-X710"}},
		{"ie11", "Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko",
			Info{"Internet Explorer 11", "Windows 7", "desktop", ""}},
		{"chromeos", "Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
			Info{"Chrome 126", "ChromeOS", "desktop", ""}},
		{"headless chrome", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/147.0.0.0 Safari/537.36",
			Info{"Chrome 147", "macOS", "desktop", ""}},
		{"unknown", "curl/8.4.0", Info{"", "", "desktop", ""}},
		{"empty", "", Info{}},
		{"spaces", "   ", Info{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Parse(tt.ua); got != tt.want {
				t.Fatalf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseLongInput(t *testing.T) {
	if got := Parse(strings.Repeat("a", 1<<20)); got.Browser != "" {
		t.Fatalf("got %+v", got)
	}
}
