package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const defaultAPIURL = "https://api.linear.app/graphql"

// reIdentifier matches a Linear issue identifier: the team key, a dash and the issue number.
var reIdentifier = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,9}-[1-9][0-9]{0,9}$`)

// ParseIssueRef returns the issue identifier (ENG-123) from an identifier or a Linear issue URL
// such as https://linear.app/acme/issue/ENG-123/some-title.
func ParseIssueRef(ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if reIdentifier.MatchString(ref) {
		return strings.ToUpper(ref), true
	}
	if !strings.Contains(ref, "://") {
		ref = "https://" + ref
	}
	u, err := url.Parse(ref)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", false
	}
	if host := strings.ToLower(u.Hostname()); host != "linear.app" && host != "www.linear.app" {
		return "", false
	}
	// Path: /<workspace>/issue/<identifier>[/<slug>]
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 3 || parts[1] != "issue" || !reIdentifier.MatchString(parts[2]) {
		return "", false
	}
	return strings.ToUpper(parts[2]), true
}

// issue is the part of a Linear issue the link stores.
type issue struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	State      struct {
		Name string `json:"name"`
	} `json:"state"`
}

const issueQuery = `query Issue($id: String!) { issue(id: $id) { id identifier url title state { name } } }`

// fetchIssue looks an issue up by identifier. Linear's issue(id:) accepts an identifier as well as a UUID.
func (m *Manager) fetchIssue(ctx context.Context, identifier string) (issue, error) {
	body, _ := json.Marshal(map[string]any{"query": issueQuery, "variables": map[string]string{"id": identifier}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.APIURL, bytes.NewReader(body))
	if err != nil {
		return issue{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	// A personal API key goes in the header as is; an OAuth token needs its "Bearer " prefix in the config.
	req.Header.Set("Authorization", m.cfg.APIKey)
	resp, err := m.http.Do(req)
	if err != nil {
		return issue{}, fmt.Errorf("calling linear: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return issue{}, fmt.Errorf("reading linear response: %w", err)
	}
	var out struct {
		Data struct {
			Issue *issue `json:"issue"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return issue{}, fmt.Errorf("linear answered HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	for _, e := range out.Errors {
		if strings.Contains(strings.ToLower(e.Message), "not found") {
			return issue{}, ErrIssueNotFound
		}
	}
	if len(out.Errors) > 0 {
		return issue{}, fmt.Errorf("linear error (HTTP %d): %s", resp.StatusCode, out.Errors[0].Message)
	}
	if resp.StatusCode != http.StatusOK {
		return issue{}, fmt.Errorf("linear answered HTTP %d", resp.StatusCode)
	}
	if out.Data.Issue == nil || out.Data.Issue.ID == "" {
		return issue{}, ErrIssueNotFound
	}
	return *out.Data.Issue, nil
}
