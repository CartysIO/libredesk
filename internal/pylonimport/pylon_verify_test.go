package pylonimport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFilterPresentIssues(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/issues/gone":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		case "/issues/here":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"id":"here","number":1,"title":"ok"}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	f := &Fetcher{
		APIRoot: srv.URL,
		HTTP:    srv.Client(),
		Token:   func() (string, error) { return "test", nil },
		Logf:    func(string, ...any) {},
	}
	issues := []Issue{
		{ID: "here", Number: 1, Title: "ok"},
		{ID: "gone", Number: 2, Title: "deleted"},
	}
	present, missing, err := f.FilterPresentIssues(context.Background(), issues)
	if err != nil {
		t.Fatal(err)
	}
	if len(present) != 1 || present[0].ID != "here" {
		t.Fatalf("present: %+v", present)
	}
	if len(missing) != 1 || missing[0].ID != "gone" {
		t.Fatalf("missing: %+v", missing)
	}
}
