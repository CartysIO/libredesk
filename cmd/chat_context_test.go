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

func TestCleanPageURLQueryWords(t *testing.T) {
	got := cleanPageURL("https://app.clyr.io/x?x-api-key=1&access_key=2&private_key=3&accessToken=4&__clerk_ticket=5&author=me&authorization_tab=a&ticket=123&keyword=k")
	if want := "https://app.clyr.io/x?author=me&authorization_tab=a&ticket=123&keyword=k"; got != want {
		t.Errorf("cleanPageURL = %q, want %q", got, want)
	}
}

func TestCleanPageURLRedactsTokenPathSegments(t *testing.T) {
	got := cleanPageURL("https://app.clyr.io/reset/a8Fk2LmQ9zXv7Bn3Rt5Yw1Hj/done")
	if want := "https://app.clyr.io/reset/redacted/done"; got != want {
		t.Errorf("cleanPageURL = %q, want %q", got, want)
	}
	keep := "https://app.clyr.io/teams/team_2b31a44f-a6e2-4f9f-b7f0-9e21a5f567e8/transactions/0b5e2c1a-2f9d-4c55-9f1e-3a7b8c9d0e1f/receipt-matching-settings"
	if got := cleanPageURL(keep); got != keep {
		t.Errorf("ids or words were redacted: %q", got)
	}
}

func TestCleanPageURLEmbeddedSecretNames(t *testing.T) {
	got := cleanPageURL("https://app.clyr.io/x?authtoken=1&apitoken=2&accesstoken=3&oauth2Token=4&XAPIKey=5&author=me&authorization_tab=a&keyword=k&ticket=7")
	if want := "https://app.clyr.io/x?author=me&authorization_tab=a&keyword=k&ticket=7"; got != want {
		t.Errorf("cleanPageURL = %q, want %q", got, want)
	}
}

func TestCleanPageURLKeepsSlugsAndRedactsHex(t *testing.T) {
	slug := "https://app.clyr.io/reports/customer-2024-quarterly-summary/receipt_matching_settings_v2"
	if got := cleanPageURL(slug); got != slug {
		t.Errorf("slug was redacted: %q", got)
	}
	got := cleanPageURL("https://app.clyr.io/verify/9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08")
	if want := "https://app.clyr.io/verify/redacted"; got != want {
		t.Errorf("cleanPageURL = %q, want %q", got, want)
	}
}
