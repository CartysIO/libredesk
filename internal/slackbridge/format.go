package slackbridge

import (
	"html"
	"regexp"
	"strings"
)

var (
	// Slack wraps links, mentions and channel references in angle brackets: <https://x|label>, <@U123>, <#C123|name>, <!here>.
	reSlackToken  = regexp.MustCompile(`<([^<>]+)>`)
	reCodeBlock   = regexp.MustCompile("(?s)\n?```(.+?)```\n?")
	reInlineCode  = regexp.MustCompile("`([^`\n]+)`")
	reBold        = regexp.MustCompile(`(^|[\s(])\*([^*\n]+)\*`)
	reItalic      = regexp.MustCompile(`(^|[\s(])_([^_\n]+)_`)
	reStrike      = regexp.MustCompile(`(^|[\s(])~([^~\n]+)~`)
	reBlankLines  = regexp.MustCompile(`\n{2,}`)
	reSpaceBefore = regexp.MustCompile(`[ \t]+\n`)
)

// mentionText returns the display text for a Slack user mention, for example "@Eran Artzi".
type mentionText func(userID string) string

// stripMention removes every mention of the given Slack user ID from the text.
func stripMention(text, userID string) string {
	if userID == "" {
		return strings.TrimSpace(text)
	}
	text = strings.ReplaceAll(text, "<@"+userID+">", "")
	return strings.TrimSpace(text)
}

// mrkdwnToHTML converts Slack message text to the HTML that Libredesk stores.
// Slack already escapes &, < and > in user text, so the text is HTML-safe except for its own <...> tokens.
func mrkdwnToHTML(text string, mention mentionText) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// Pull code blocks out first so their content is not formatted.
	var blocks []string
	text = reCodeBlock.ReplaceAllStringFunc(text, func(m string) string {
		blocks = append(blocks, strings.Trim(strings.Trim(m, "\n"), "`\n"))
		return "\x00" + string(rune('a'+len(blocks)-1)) + "\x00"
	})

	text = reSlackToken.ReplaceAllStringFunc(text, func(m string) string {
		return slackToken(m[1:len(m)-1], mention)
	})
	text = reInlineCode.ReplaceAllString(text, "<code>$1</code>")
	text = reBold.ReplaceAllString(text, "$1<strong>$2</strong>")
	text = reItalic.ReplaceAllString(text, "$1<em>$2</em>")
	text = reStrike.ReplaceAllString(text, "$1<s>$2</s>")

	text = reSpaceBefore.ReplaceAllString(strings.TrimSpace(text), "\n")
	var out strings.Builder
	for _, para := range reBlankLines.Split(text, -1) {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		out.WriteString("<p>")
		out.WriteString(strings.ReplaceAll(para, "\n", "<br>"))
		out.WriteString("</p>")
	}
	result := out.String()
	for i, b := range blocks {
		result = strings.Replace(result, "\x00"+string(rune('a'+i))+"\x00", "</p><pre>"+b+"</pre><p>", 1)
	}
	return strings.ReplaceAll(result, "<p></p>", "")
}

// slackToken renders the inside of one <...> token.
func slackToken(tok string, mention mentionText) string {
	target, label, _ := strings.Cut(tok, "|")
	switch {
	case strings.HasPrefix(target, "@"):
		if mention != nil {
			if name := mention(target[1:]); name != "" {
				return html.EscapeString("@" + name)
			}
		}
		return "@" + target[1:]
	case strings.HasPrefix(target, "#"):
		if label != "" {
			return "#" + label
		}
		return "#" + target[1:]
	case strings.HasPrefix(target, "!"):
		if label != "" {
			return label
		}
		name, _, _ := strings.Cut(target[1:], "^")
		return "@" + name
	case strings.HasPrefix(target, "http://"), strings.HasPrefix(target, "https://"), strings.HasPrefix(target, "mailto:"):
		if label == "" {
			label = strings.TrimPrefix(target, "mailto:")
		}
		return `<a href="` + strings.ReplaceAll(target, `"`, "&quot;") + `">` + label + `</a>`
	}
	return "&lt;" + tok + "&gt;"
}

// escapeMrkdwn escapes plain text for a Slack message. Slack requires &, < and > to be escaped.
func escapeMrkdwn(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
}

// quote prefixes every line of text with "> ", so Slack shows it as a block quote.
func quote(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return "> " + strings.ReplaceAll(text, "\n", "\n> ")
}
