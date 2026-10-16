package clerk

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
)

const (
	testIssuer = "https://sacred-chimp-95.clerk.accounts.dev"
	testRoot   = "https://support.example.com"
)

func testPublishableKey() string {
	return "pk_test_" + base64.StdEncoding.EncodeToString([]byte("sacred-chimp-95.clerk.accounts.dev$"))
}

func TestIssuerFromPublishableKey(t *testing.T) {
	got, err := IssuerFromPublishableKey(testPublishableKey())
	if err != nil {
		t.Fatal(err)
	}
	if got != testIssuer {
		t.Fatalf("issuer = %q, want %q", got, testIssuer)
	}

	live := "pk_live_" + base64.RawStdEncoding.EncodeToString([]byte("clerk.clyr.io$"))
	if got, _ := IssuerFromPublishableKey(live); got != "https://clerk.clyr.io" {
		t.Fatalf("live issuer = %q", got)
	}

	for _, bad := range []string{"", "sk_test_abc", "pk_other_abc", "pk_test_!!!"} {
		if _, err := IssuerFromPublishableKey(bad); err == nil {
			t.Errorf("IssuerFromPublishableKey(%q) returned no error", bad)
		}
	}
}

func newTestClient(t *testing.T) (*Client, *rsa.PrivateKey, time.Time) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	c := &Client{
		cfg:    Config{PublishableKey: testPublishableKey(), SecretKey: "sk_test_x"},
		issuer: testIssuer,
		keys:   &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}},
		http:   http.DefaultClient,
		now:    func() time.Time { return now },
	}
	return c, key, now
}

func sign(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(claims)
	jws, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	s, err := jws.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestVerifySession(t *testing.T) {
	c, key, now := newTestClient(t)
	valid := func() map[string]any {
		return map[string]any{
			"iss": testIssuer,
			"sub": "user_123",
			"azp": testRoot,
			"exp": now.Add(time.Minute).Unix(),
			"nbf": now.Add(-time.Second).Unix(),
		}
	}

	sub, err := c.VerifySession(context.Background(), sign(t, key, valid()), testRoot+"/")
	if err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if sub != "user_123" {
		t.Fatalf("subject = %q", sub)
	}

	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	cases := map[string]func() string{
		"wrong signer": func() string { return sign(t, otherKey, valid()) },
		"wrong issuer": func() string { m := valid(); m["iss"] = "https://evil.example"; return sign(t, key, m) },
		"no subject":   func() string { m := valid(); delete(m, "sub"); return sign(t, key, m) },
		"expired":      func() string { m := valid(); m["exp"] = now.Add(-time.Minute).Unix(); return sign(t, key, m) },
		"no expiry":    func() string { m := valid(); delete(m, "exp"); return sign(t, key, m) },
		"not yet":      func() string { m := valid(); m["nbf"] = now.Add(time.Minute).Unix(); return sign(t, key, m) },
		"wrong azp":    func() string { m := valid(); m["azp"] = "https://evil.example"; return sign(t, key, m) },
		"no azp":       func() string { m := valid(); delete(m, "azp"); return sign(t, key, m) },
		"garbage":      func() string { return "not.a.jwt" },
	}
	for name, token := range cases {
		if _, err := c.VerifySession(context.Background(), token(), testRoot); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: err = %v, want ErrInvalidToken", name, err)
		}
	}
}

func TestPrimaryEmail(t *testing.T) {
	users := map[string]string{
		"user_ok": `{"primary_email_address_id":"e2","email_addresses":[
			{"id":"e1","email_address":"old@clyr.io","verification":{"status":"verified"}},
			{"id":"e2","email_address":" Agent@Clyr.io ","verification":{"status":"verified"}}]}`,
		"user_unverified": `{"primary_email_address_id":"e1","email_addresses":[
			{"id":"e1","email_address":"agent@clyr.io","verification":{"status":"unverified"}}]}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_x" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, ok := users[r.URL.Path[len("/users/"):]]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c, _, _ := newTestClient(t)
	c.baseURL = srv.URL

	got, err := c.PrimaryEmail(context.Background(), "user_ok")
	if err != nil || got != "agent@clyr.io" {
		t.Fatalf("PrimaryEmail = %q, %v", got, err)
	}
	if _, err := c.PrimaryEmail(context.Background(), "user_unverified"); !errors.Is(err, ErrNoVerifiedMail) {
		t.Fatalf("unverified: err = %v", err)
	}
	if _, err := c.PrimaryEmail(context.Background(), "user_missing"); err == nil {
		t.Fatal("missing user: no error")
	}
}
