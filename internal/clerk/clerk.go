// Package clerk signs agents in with a Clerk session token.
//
// The login page mounts Clerk's SignIn component. After sign-in, the browser
// sends the Clerk session token to the backend. The backend checks the token
// against the Clerk instance's public keys and reads the user's verified
// primary email from the Clerk Backend API. That email selects the agent.
package clerk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

const (
	backendAPI = "https://api.clerk.com/v1"
	// leeway allows for small clock differences between Clerk and this server.
	leeway = 5 * time.Second
)

var (
	ErrInvalidToken   = errors.New("invalid clerk session token")
	ErrNoVerifiedMail = errors.New("clerk user has no verified primary email")
)

// Config holds the Clerk instance keys.
type Config struct {
	PublishableKey string
	SecretKey      string
}

// Enabled reports whether both keys are set.
func (c Config) Enabled() bool {
	return c.PublishableKey != "" && c.SecretKey != ""
}

// keySet verifies a JWT signature and returns its payload.
type keySet interface {
	VerifySignature(ctx context.Context, jwt string) ([]byte, error)
}

// Client verifies Clerk session tokens and reads Clerk users.
type Client struct {
	cfg     Config
	issuer  string
	keys    keySet
	http    *http.Client
	baseURL string
	now     func() time.Time
}

// New returns a Client for the Clerk instance named in the publishable key.
func New(cfg Config) (*Client, error) {
	issuer, err := IssuerFromPublishableKey(cfg.PublishableKey)
	if err != nil {
		return nil, err
	}
	return &Client{
		cfg:     cfg,
		issuer:  issuer,
		keys:    oidc.NewRemoteKeySet(context.Background(), issuer+"/.well-known/jwks.json"),
		http:    &http.Client{Timeout: 10 * time.Second},
		baseURL: backendAPI,
		now:     time.Now,
	}, nil
}

// PublishableKey returns the key the login page needs to load Clerk.
func (c *Client) PublishableKey() string {
	return c.cfg.PublishableKey
}

// IssuerFromPublishableKey decodes the Frontend API URL from a publishable key.
// A key is "pk_test_" or "pk_live_" followed by base64("<frontend api host>$").
func IssuerFromPublishableKey(key string) (string, error) {
	parts := strings.SplitN(key, "_", 3)
	if len(parts) != 3 || parts[0] != "pk" || (parts[1] != "test" && parts[1] != "live") {
		return "", errors.New("clerk publishable key must start with pk_test_ or pk_live_")
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(parts[2], "="))
	if err != nil {
		return "", fmt.Errorf("decoding clerk publishable key: %w", err)
	}
	host := strings.TrimSuffix(string(raw), "$")
	if host == "" || strings.ContainsAny(host, "/$ ") {
		return "", errors.New("clerk publishable key does not contain a frontend api host")
	}
	return "https://" + host, nil
}

type sessionClaims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	AZP       string `json:"azp"`
	ExpiresAt int64  `json:"exp"`
	NotBefore int64  `json:"nbf"`
}

// VerifySession checks the token signature, issuer, lifetime and authorized
// party, and returns the Clerk user ID. rootURL is the Libredesk root URL;
// the token's azp claim must be its origin.
func (c *Client) VerifySession(ctx context.Context, token, rootURL string) (string, error) {
	payload, err := c.keys.VerifySignature(ctx, token)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	var claims sessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	now := c.now()
	switch {
	case claims.Issuer != c.issuer:
		return "", fmt.Errorf("%w: issuer %q", ErrInvalidToken, claims.Issuer)
	case claims.Subject == "":
		return "", fmt.Errorf("%w: no subject", ErrInvalidToken)
	case claims.ExpiresAt == 0 || now.After(time.Unix(claims.ExpiresAt, 0).Add(leeway)):
		return "", fmt.Errorf("%w: expired", ErrInvalidToken)
	case claims.NotBefore != 0 && now.Add(leeway).Before(time.Unix(claims.NotBefore, 0)):
		return "", fmt.Errorf("%w: not valid yet", ErrInvalidToken)
	}

	origin, err := originOf(rootURL)
	if err != nil {
		return "", err
	}
	if claims.AZP != origin {
		return "", fmt.Errorf("%w: authorized party %q, want %q", ErrInvalidToken, claims.AZP, origin)
	}
	return claims.Subject, nil
}

type clerkUser struct {
	PrimaryEmailAddressID string `json:"primary_email_address_id"`
	EmailAddresses        []struct {
		ID           string `json:"id"`
		EmailAddress string `json:"email_address"`
		Verification *struct {
			Status string `json:"status"`
		} `json:"verification"`
	} `json:"email_addresses"`
}

// PrimaryEmail returns the user's primary email when Clerk has verified it.
func (c *Client) PrimaryEmail(ctx context.Context, userID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/users/"+url.PathEscape(userID), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.SecretKey)
	req.Header.Set("User-Agent", "libredesk-clerk")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching clerk user: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching clerk user: status %d", resp.StatusCode)
	}

	var u clerkUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", fmt.Errorf("decoding clerk user: %w", err)
	}
	for _, e := range u.EmailAddresses {
		if e.ID == u.PrimaryEmailAddressID && e.Verification != nil && e.Verification.Status == "verified" {
			return strings.ToLower(strings.TrimSpace(e.EmailAddress)), nil
		}
	}
	return "", ErrNoVerifiedMail
}

// originOf returns the browser origin of rootURL: lowercase scheme and host,
// without the default port, which is how browsers report it in azp.
func originOf(rootURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(rootURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("app root url %q is not an absolute url", rootURL)
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port != "" {
		host += ":" + port
	}
	return scheme + "://" + host, nil
}
