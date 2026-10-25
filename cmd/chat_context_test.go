package main

import (
	"strings"
	"testing"
)

const testChromeMacUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

func TestBuildChatContextAttributes(t *testing.T) {
	got := buildChatContextAttributes(testChromeMacUA, chatContextReq{
		PageURL:  "https://app.clyr.io/receipts?id=1#top",
		Timezone: "Asia/Jerusalem",
		App:      "web",
	})
	want := map[string]any{
		chatAttrBrowser:    "Chrome 126",
		chatAttrOS:         "macOS",
		chatAttrDeviceType: "desktop",
		chatAttrPageURL:    "https://app.clyr.io/receipts?id=1",
		chatAttrUserAgent:  testChromeMacUA,
		chatAttrTimezone:   "Asia/Jerusalem",
		chatAttrApp:        "web",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
}

func TestBuildChatContextAttributesUntrusted(t *testing.T) {
	got := buildChatContextAttributes(strings.Repeat("x", 5000), chatContextReq{
		PageURL:  "javascript:alert(1)",
		Timezone: "<script>",
		App:      "we\x00b\n" + strings.Repeat("é", 200),
	})
	if _, ok := got[chatAttrPageURL]; ok {
		t.Error("a javascript: URL was kept")
	}
	if _, ok := got[chatAttrTimezone]; ok {
		t.Error("an invalid timezone was kept")
	}
	if ua := got[chatAttrUserAgent].(string); len(ua) != maxChatContextUALen {
		t.Errorf("user agent length = %d", len(ua))
	}
	app := got[chatAttrApp].(string)
	if len(app) > maxChatContextAppLen || strings.ContainsAny(app, "\x00\n") || !strings.HasPrefix(app, "web") {
		t.Errorf("app = %q", app)
	}

	long := buildChatContextAttributes("", chatContextReq{PageURL: "https://a.example/" + strings.Repeat("a", 5000)})
	if len(long[chatAttrPageURL].(string)) != maxChatContextURLLen {
		t.Errorf("page url length = %d", len(long[chatAttrPageURL].(string)))
	}
	if empty := buildChatContextAttributes("", chatContextReq{}); len(empty) != 0 {
		t.Errorf("empty input gave %v", empty)
	}
}

func TestMergeChatContextKeepsExisting(t *testing.T) {
	got := mergeChatContext(map[string]any{chatAttrApp: "from-form", "other": 1}, map[string]any{chatAttrApp: "web", chatAttrBrowser: "Chrome 126"})
	if got[chatAttrApp] != "from-form" || got[chatAttrBrowser] != "Chrome 126" || got["other"] != 1 {
		t.Fatalf("got %v", got)
	}
	if got := mergeChatContext(nil, map[string]any{"a": "b"}); got["a"] != "b" {
		t.Fatalf("nil attrs: %v", got)
	}
}

func TestCleanPageURLDropsFragmentAndUserinfo(t *testing.T) {
	got := cleanPageURL("https://user:pass@app.clyr.io/cb?step=2#access_token=abc")
	if want := "https://app.clyr.io/cb?step=2"; got != want {
		t.Errorf("cleanPageURL = %q, want %q", got, want)
	}
}

func TestCleanPageURLDropsCredentialQuery(t *testing.T) {
	got := cleanPageURL("https://app.clyr.io/reset?tab=plan&token=abc&code=xyz&__clerk_db_jwt=j&State=s&page=2")
	if want := "https://app.clyr.io/reset?tab=plan&page=2"; got != want {
		t.Errorf("cleanPageURL = %q, want %q", got, want)
	}
	if got := cleanPageURL("https://app.clyr.io/receipts?id=1"); got != "https://app.clyr.io/receipts?id=1" {
		t.Errorf("plain query changed: %q", got)
	}
}
