package slackbridge

import (
	"testing"
	"time"
)

func TestMrkdwnToHTML(t *testing.T) {
	names := func(id string) string {
		if id == "U1" {
			return "Eran Artzi"
		}
		return ""
	}
	tests := []struct {
		name, in, want string
	}{
		{"plain", "hello", "<p>hello</p>"},
		{"lines and paragraphs", "one\ntwo\n\nthree", "<p>one<br>two</p><p>three</p>"},
		{"formatting", "*bold* _it_ ~gone~ `x`", "<p><strong>bold</strong> <em>it</em> <s>gone</s> <code>x</code></p>"},
		{"link with label", "see <https://clyr.io/a|the docs>", `<p>see <a href="https://clyr.io/a">the docs</a></p>`},
		{"bare link", "<https://clyr.io>", `<p><a href="https://clyr.io">https://clyr.io</a></p>`},
		{"mailto", "<mailto:a@b.co|a@b.co>", `<p><a href="mailto:a@b.co">a@b.co</a></p>`},
		{"unsafe scheme stays text", "<javascript:alert(1)|x>", "<p>&lt;javascript:alert(1)|x&gt;</p>"},
		{"user mention", "ask <@U1>", "<p>ask @Eran Artzi</p>"},
		{"unknown mention", "ask <@U9>", "<p>ask @U9</p>"},
		{"channel", "in <#C1|support>", "<p>in #support</p>"},
		{"special mention", "<!here> look", "<p>@here look</p>"},
		{"escaped text stays escaped", "a &lt;b&gt; &amp; c", "<p>a &lt;b&gt; &amp; c</p>"},
		{"code block", "before\n```x *y*```\nafter", "<p>before</p><pre>x *y*</pre><p>after</p>"},
		{"empty", "  ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mrkdwnToHTML(tt.in, names); got != tt.want {
				t.Fatalf("mrkdwnToHTML(%q)\n got %q\nwant %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestStripMention(t *testing.T) {
	if got := stripMention("<@UBOT> thanks, fixed now", "UBOT"); got != "thanks, fixed now" {
		t.Fatalf("got %q", got)
	}
	if got := stripMention(" hi ", ""); got != "hi" {
		t.Fatalf("got %q", got)
	}
}

func TestQuoteAndEscape(t *testing.T) {
	if got := quote(escapeMrkdwn("a <b> & c\nd")); got != "> a &lt;b&gt; &amp; c\n> d" {
		t.Fatalf("got %q", got)
	}
	if got := quote("  "); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestVerifySignature(t *testing.T) {
	// Example from Slack's "Verifying requests from Slack" guide.
	const (
		secret = "8f742231b10e8888abcd99yyyzzz85a5"
		ts     = "1531420618"
		body   = "token=xyzz0WbapA4vBCDEFasx0q6G&team_id=T1DC2JH3J&team_domain=testteamnow&channel_id=G8PSS9T3V&channel_name=foobar&user_id=U2CERLKJA&user_name=roadrunner&command=%2Fwebhook-collect&text=&response_url=https%3A%2F%2Fhooks.slack.com%2Fcommands%2FT1DC2JH3J%2F397700885554%2F96rGlfmibIGlgcZRskXaIFfN&trigger_id=398738663015.47445629121.803a0bc887a14d10d2c447fce8b6703c"
		sig    = "v0=a2114d57b48eac39b9ad189dd8316235a7b4a8d21a10bd27519666489c69b503"
	)
	now := time.Unix(1531420618, 0).Add(30 * time.Second)
	if !VerifySignature(secret, ts, []byte(body), sig, now) {
		t.Fatal("valid signature rejected")
	}
	if VerifySignature("other", ts, []byte(body), sig, now) {
		t.Fatal("wrong secret accepted")
	}
	if VerifySignature(secret, ts, []byte(body+"x"), sig, now) {
		t.Fatal("changed body accepted")
	}
	if VerifySignature(secret, ts, []byte(body), sig, now.Add(10*time.Minute)) {
		t.Fatal("old request accepted")
	}
	if VerifySignature(secret, "abc", []byte(body), sig, now) || VerifySignature("", ts, []byte(body), sig, now) {
		t.Fatal("malformed input accepted")
	}
}
