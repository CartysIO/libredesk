package slackbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPIURL = "https://slack.com/api/"
	maxRetries    = 3
	// maxRetryWait caps one wait between retries. Slack's Retry-After can be longer; the job then fails.
	maxRetryWait = 30 * time.Second
	// maxFileSize caps a Slack file the bridge downloads.
	maxFileSize = 25 << 20
)

// slackAPI is a minimal client for the Slack Web API methods the bridge uses.
type slackAPI struct {
	token   string
	baseURL string
	http    *http.Client
	// wait pauses between retries and returns early with ctx's error. Tests replace it.
	wait func(ctx context.Context, d time.Duration) error
}

func waitContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func newSlackAPI(token, baseURL string) *slackAPI {
	if baseURL == "" {
		baseURL = defaultAPIURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &slackAPI{token: token, baseURL: baseURL, http: &http.Client{Timeout: 15 * time.Second}, wait: waitContext}
}

type slackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// call posts form values to a Web API method and decodes the JSON reply into out.
// It retries a network error or an HTTP 5xx with backoff (1s, 2s, 4s), and an HTTP 429 after its
// Retry-After delay. A delay longer than maxRetryWait fails the call instead of blocking the worker.
func (a *slackAPI) call(ctx context.Context, method string, form url.Values, out any) error {
	for attempt := 0; ; attempt++ {
		backoff := time.Duration(1<<attempt) * time.Second
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+method, strings.NewReader(form.Encode()))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+a.token)
		resp, err := a.http.Do(req)
		if err != nil {
			if attempt < maxRetries && ctx.Err() == nil {
				if werr := a.wait(ctx, backoff); werr == nil {
					continue
				}
			}
			return fmt.Errorf("slack %s: %w", method, err)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("slack %s: reading response: %w", method, err)
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			wait := backoff
			if resp.StatusCode == http.StatusTooManyRequests {
				secs, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
				wait = time.Duration(max(secs, 1)) * time.Second
			}
			if attempt < maxRetries && wait <= maxRetryWait {
				if err := a.wait(ctx, wait); err != nil {
					return fmt.Errorf("slack %s: %w", method, err)
				}
				continue
			}
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("slack %s: HTTP %d", method, resp.StatusCode)
		}
		var base slackResponse
		if err := json.Unmarshal(body, &base); err != nil {
			return fmt.Errorf("slack %s: decoding response: %w", method, err)
		}
		if !base.OK {
			return fmt.Errorf("slack %s: %s", method, base.Error)
		}
		if out != nil {
			return json.Unmarshal(body, out)
		}
		return nil
	}
}

// authTest returns the bot's own Slack user ID.
func (a *slackAPI) authTest(ctx context.Context) (string, error) {
	var out struct {
		UserID string `json:"user_id"`
	}
	if err := a.call(ctx, "auth.test", url.Values{}, &out); err != nil {
		return "", err
	}
	return out.UserID, nil
}

// postMessage posts text to a channel, in a thread when threadTS is set, and returns the message ts.
func (a *slackAPI) postMessage(ctx context.Context, channel, threadTS, text string) (string, error) {
	form := url.Values{"channel": {channel}, "text": {text}, "unfurl_links": {"false"}, "unfurl_media": {"false"}}
	if threadTS != "" {
		form.Set("thread_ts", threadTS)
	}
	var out struct {
		TS string `json:"ts"`
	}
	if err := a.call(ctx, "chat.postMessage", form, &out); err != nil {
		return "", err
	}
	return out.TS, nil
}

// updateMessage replaces the text of a message.
func (a *slackAPI) updateMessage(ctx context.Context, channel, ts, text string) error {
	return a.call(ctx, "chat.update", url.Values{"channel": {channel}, "ts": {ts}, "text": {text}}, nil)
}

// deleteMessage deletes a message the bot posted.
func (a *slackAPI) deleteMessage(ctx context.Context, channel, ts string) error {
	return a.call(ctx, "chat.delete", url.Values{"channel": {channel}, "ts": {ts}}, nil)
}

// addReaction adds an emoji reaction to a message. Slack reports already_reacted as an error; it is ignored.
func (a *slackAPI) addReaction(ctx context.Context, channel, ts, name string) error {
	err := a.call(ctx, "reactions.add", url.Values{"channel": {channel}, "timestamp": {ts}, "name": {name}}, nil)
	if err != nil && strings.HasSuffix(err.Error(), "already_reacted") {
		return nil
	}
	return err
}

// slackUser is the part of a Slack user profile the bridge needs.
type slackUser struct {
	ID      string
	Name    string
	Email   string
	IsBot   bool
	Deleted bool
}

// userInfo returns a Slack user's display name and profile email (needs users:read.email).
func (a *slackAPI) userInfo(ctx context.Context, userID string) (slackUser, error) {
	var out struct {
		User struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			IsBot   bool   `json:"is_bot"`
			Deleted bool   `json:"deleted"`
			Profile struct {
				RealName    string `json:"real_name"`
				DisplayName string `json:"display_name"`
				Email       string `json:"email"`
			} `json:"profile"`
		} `json:"user"`
	}
	if err := a.call(ctx, "users.info", url.Values{"user": {userID}}, &out); err != nil {
		return slackUser{}, err
	}
	u := out.User
	name := u.Profile.RealName
	if name == "" {
		name = u.Profile.DisplayName
	}
	if name == "" {
		name = u.Name
	}
	return slackUser{ID: u.ID, Name: name, Email: strings.ToLower(strings.TrimSpace(u.Profile.Email)), IsBot: u.IsBot, Deleted: u.Deleted}, nil
}

// download fetches a private Slack file with the bot token.
func (a *slackAPI) download(ctx context.Context, fileURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading slack file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading slack file: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("downloading slack file: %w", err)
	}
	if len(data) > maxFileSize {
		return nil, fmt.Errorf("slack file is larger than %d bytes", maxFileSize)
	}
	return data, nil
}
