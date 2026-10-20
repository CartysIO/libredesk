// Package pylonimport copies Clyr's Pylon support history into Libredesk: accounts become organizations,
// contacts become contacts, issues become conversations with their messages, notes and files.
//
// The Pylon side is read-only: the client only sends GET requests. Everything it reads is cached on disk,
// so a second run (or a resumed one) reads the cache and does not call Pylon again.
package pylonimport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const apiBase = "https://api.usepylon.com"

// Ref is a reference to another Pylon object.
type Ref struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// ExternalID is an id of a Pylon account in another system.
type ExternalID struct {
	ExternalID string `json:"external_id"`
	Label      string `json:"label"`
}

// CustomField is a Pylon custom field value. Multi-select fields fill Values.
type CustomField struct {
	Value  string   `json:"value"`
	Values []string `json:"values"`
}

// Account is a Pylon account (a customer company).
type Account struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	ExternalIDs  []ExternalID           `json:"external_ids"`
	CustomFields map[string]CustomField `json:"custom_fields"`
}

// Contact is a Pylon contact (a customer person).
type Contact struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Emails  []string `json:"emails"`
	Account *Ref     `json:"account"`
}

// User is a Pylon user (a staff member or an API identity).
type User struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Email         string   `json:"email"`
	Emails        []string `json:"emails"`
	IsDeactivated bool     `json:"is_deactivated"`
}

// Issue is a Pylon issue (a ticket).
type Issue struct {
	ID                string                 `json:"id"`
	Number            int                    `json:"number"`
	Title             string                 `json:"title"`
	Link              string                 `json:"link"`
	BodyHTML          string                 `json:"body_html"`
	State             string                 `json:"state"`
	Source            string                 `json:"source"`
	Account           *Ref                   `json:"account"`
	Assignee          *Ref                   `json:"assignee"`
	Requester         *Ref                   `json:"requester"`
	Tags              []string               `json:"tags"`
	CustomFields      map[string]CustomField `json:"custom_fields"`
	AttachmentURLs    []string               `json:"attachment_urls"`
	CreatedAt         time.Time              `json:"created_at"`
	FirstResponseTime *time.Time             `json:"first_response_time"`
	ResolutionTime    *time.Time             `json:"resolution_time"`
	LatestMessageTime *time.Time             `json:"latest_message_time"`
}

// MessageAuthor is who wrote a message: a Pylon user, a contact, or neither (an integration such as Linear).
type MessageAuthor struct {
	Name    string `json:"name"`
	User    *Ref   `json:"user"`
	Contact *Ref   `json:"contact"`
}

// EmailInfo holds the email headers of a message that went through email.
type EmailInfo struct {
	FromEmail string   `json:"from_email"`
	ToEmails  []string `json:"to_emails"`
	CCEmails  []string `json:"cc_emails"`
	BCCEmails []string `json:"bcc_emails"`
	MessageID string   `json:"message_id"`
}

// Message is one message of an issue: a customer message, a staff reply or an internal note.
type Message struct {
	ID          string         `json:"id"`
	MessageHTML string         `json:"message_html"`
	Timestamp   time.Time      `json:"timestamp"`
	Source      string         `json:"source"`
	Author      *MessageAuthor `json:"author"`
	IsPrivate   bool           `json:"is_private"`
	FileURLs    []string       `json:"file_urls"`
	EmailInfo   *EmailInfo     `json:"email_info"`
	ThreadID    string         `json:"thread_id"`
}

// Data is everything the import reads from Pylon.
type Data struct {
	Users    []User
	Accounts []Account
	Contacts []Contact
	Issues   []Issue
	// Messages by issue id, oldest first.
	Messages map[string][]Message
}

// Fetcher reads Pylon through its REST API and caches every response under Dir.
type Fetcher struct {
	Dir    string
	Token  func() (string, error)
	HTTP   *http.Client
	Logf   func(format string, args ...any)
	Oldest time.Time

	limiter <-chan time.Time
}

// Fetch returns users, accounts, contacts, issues and messages, from the cache when present.
func (f *Fetcher) Fetch(ctx context.Context) (Data, error) {
	if f.HTTP == nil {
		f.HTTP = &http.Client{Timeout: 2 * time.Minute}
	}
	if f.Logf == nil {
		f.Logf = func(string, ...any) {}
	}
	if f.Oldest.IsZero() {
		f.Oldest = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	// Messages allow 120 requests a minute; one request every 550ms stays under every endpoint's limit but /issues.
	f.limiter = time.Tick(550 * time.Millisecond)
	if err := os.MkdirAll(filepath.Join(f.Dir, "messages"), 0o700); err != nil {
		return Data{}, err
	}

	var d Data
	if err := f.cached(ctx, "users.json", "/users", url.Values{"include_deactivated": {"true"}}, &d.Users); err != nil {
		return d, fmt.Errorf("users: %w", err)
	}
	if err := f.cached(ctx, "accounts.json", "/accounts", url.Values{"limit": {"999"}}, &d.Accounts); err != nil {
		return d, fmt.Errorf("accounts: %w", err)
	}
	if err := f.cached(ctx, "contacts.json", "/contacts", url.Values{"limit": {"999"}}, &d.Contacts); err != nil {
		return d, fmt.Errorf("contacts: %w", err)
	}
	if err := f.fetchIssues(ctx, &d.Issues); err != nil {
		return d, fmt.Errorf("issues: %w", err)
	}

	d.Messages = make(map[string][]Message, len(d.Issues))
	for n, is := range d.Issues {
		var msgs []Message
		name := filepath.Join("messages", is.ID+".json")
		if err := f.cached(ctx, name, "/issues/"+is.ID+"/messages", nil, &msgs); err != nil {
			return d, fmt.Errorf("messages of issue #%d: %w", is.Number, err)
		}
		d.Messages[is.ID] = msgs
		if (n+1)%200 == 0 {
			f.Logf("pylon: messages read for %d of %d issues", n+1, len(d.Issues))
		}
	}
	return d, nil
}

// fetchIssues lists issues in windows of 360 days, newest first, back to Oldest.
func (f *Fetcher) fetchIssues(ctx context.Context, out *[]Issue) error {
	path := filepath.Join(f.Dir, "issues.json")
	if b, err := os.ReadFile(path); err == nil {
		return json.Unmarshal(b, out)
	}
	seen := map[string]bool{}
	end := time.Now().UTC().Add(time.Hour).Truncate(time.Hour)
	for end.After(f.Oldest) {
		start := end.Add(-360 * 24 * time.Hour)
		var page []Issue
		q := url.Values{"start_time": {start.Format(time.RFC3339)}, "end_time": {end.Format(time.RFC3339)}}
		if err := f.getAll(ctx, "/issues", q, &page); err != nil {
			return err
		}
		f.Logf("pylon: %d issues created %s to %s", len(page), start.Format("2006-01-02"), end.Format("2006-01-02"))
		for _, is := range page {
			if !seen[is.ID] {
				seen[is.ID] = true
				*out = append(*out, is)
			}
		}
		end = start
	}
	return writeJSON(path, *out)
}

// cached reads name from the cache, or GETs path (all pages) and writes it there.
func (f *Fetcher) cached(ctx context.Context, name, path string, q url.Values, out any) error {
	p := filepath.Join(f.Dir, name)
	if b, err := os.ReadFile(p); err == nil {
		return json.Unmarshal(b, out)
	}
	var raw []json.RawMessage
	if err := f.getAll(ctx, path, q, &raw); err != nil {
		return err
	}
	if raw == nil {
		raw = []json.RawMessage{}
	}
	if err := writeJSON(p, raw); err != nil {
		return err
	}
	b, _ := json.Marshal(raw)
	return json.Unmarshal(b, out)
}

// getAll follows pagination.cursor and appends every page's data to out (a pointer to a slice).
func (f *Fetcher) getAll(ctx context.Context, path string, q url.Values, out any) error {
	q2 := url.Values{}
	for k, v := range q {
		q2[k] = v
	}
	var all []json.RawMessage
	for {
		var resp struct {
			Data       []json.RawMessage `json:"data"`
			Pagination struct {
				Cursor      string `json:"cursor"`
				HasNextPage bool   `json:"has_next_page"`
			} `json:"pagination"`
		}
		if err := f.get(ctx, path, q2, &resp); err != nil {
			return err
		}
		all = append(all, resp.Data...)
		if !resp.Pagination.HasNextPage || resp.Pagination.Cursor == "" {
			break
		}
		q2.Set("cursor", resp.Pagination.Cursor)
	}
	b, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// get sends one GET request. It waits and retries on 429 and 5xx. GET is the only method this client sends.
func (f *Fetcher) get(ctx context.Context, path string, q url.Values, out any) error {
	token, err := f.Token()
	if err != nil {
		return err
	}
	u := apiBase + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	for attempt := 0; ; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-f.limiter:
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "libredesk-pylon-import/1.0 (read-only)")
		resp, err := f.HTTP.Do(req)
		if err != nil {
			if attempt < 5 {
				time.Sleep(time.Duration(attempt+1) * 5 * time.Second)
				continue
			}
			return err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}
		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) && attempt < 8 {
			wait, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
			time.Sleep(time.Duration(max(wait, 5)+1) * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("GET %s: HTTP %d: %.300s", path, resp.StatusCode, body)
		}
		return json.Unmarshal(body, out)
	}
}

// FileKey is the cache name of a Pylon file URL: the hash of the URL without its signature.
func FileKey(fileURL string) string {
	base, _, _ := strings.Cut(fileURL, "?")
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:16])
}

// FileName is the original file name in a Pylon file URL. Pylon prefixes it with a UUID and a dash.
func FileName(fileURL string) string {
	u, err := url.Parse(fileURL)
	if err != nil {
		return "file"
	}
	name := u.Path
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	if dec, err := url.PathUnescape(name); err == nil {
		name = dec
	}
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	// "<uuid>-name.png": a UUID is 36 characters.
	if len(name) > 37 && name[36] == '-' && strings.Count(name[:36], "-") == 4 {
		name = name[37:]
	}
	if name == "" {
		return "file"
	}
	return name
}

// DownloadFiles downloads every URL into Dir/files, skipping files already there, with a few workers.
// Pylon file URLs are signed: they need no Authorization header. It returns the URLs that failed.
func (f *Fetcher) DownloadFiles(ctx context.Context, urls []string) map[string]error {
	if f.HTTP == nil {
		f.HTTP = &http.Client{Timeout: 2 * time.Minute}
	}
	dir := filepath.Join(f.Dir, "files")
	_ = os.MkdirAll(dir, 0o700)
	var (
		mu     sync.Mutex
		failed = map[string]error{}
		work   = make(chan string)
		wg     sync.WaitGroup
		done   int
	)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range work {
				err := f.download(ctx, u, filepath.Join(dir, FileKey(u)))
				mu.Lock()
				if err != nil {
					failed[u] = err
				}
				done++
				if done%500 == 0 {
					f.Logf("pylon: %d of %d files downloaded", done, len(urls))
				}
				mu.Unlock()
			}
		}()
	}
	for _, u := range urls {
		work <- u
	}
	close(work)
	wg.Wait()
	return failed
}

func (f *Fetcher) download(ctx context.Context, fileURL, path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	var lastErr error
	for attempt := range 4 {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 3 * time.Second)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "libredesk-pylon-import/1.0 (read-only)")
		resp, err := f.HTTP.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
				break
			}
			continue
		}
		tmp, err := os.CreateTemp(filepath.Dir(path), "dl-*.tmp")
		if err != nil {
			return err
		}
		_, werr := tmp.Write(body)
		cerr := tmp.Close()
		if werr != nil || cerr != nil {
			os.Remove(tmp.Name())
			return errors.Join(werr, cerr)
		}
		return os.Rename(tmp.Name(), path)
	}
	return lastErr
}

// ReadFile returns a downloaded file, or os.ErrNotExist.
func (f *Fetcher) ReadFile(fileURL string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(f.Dir, "files", FileKey(fileURL)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, os.ErrNotExist
	}
	return b, err
}

func writeJSON(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
