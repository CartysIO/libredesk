package media

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

type flakyPutStore struct {
	calls int
}

func (f *flakyPutStore) Put(name, cType string, content io.ReadSeeker) (string, error) {
	f.calls++
	if f.calls < 3 {
		return "", fmt.Errorf(`status code: 503 Service Unavailable: %q`, `<Error><Code>SlowDown</Code></Error>`)
	}
	return name, nil
}

func (f *flakyPutStore) Delete(string) error                                  { return nil }
func (f *flakyPutStore) GetURL(string, string, string) string                 { return "" }
func (f *flakyPutStore) GetBlob(string) ([]byte, error)                       { return nil, errors.New("n/a") }
func (f *flakyPutStore) Name() string                                         { return "flaky" }
func (f *flakyPutStore) SignedURLValidator() func(string, string, int64) bool { return nil }

func TestUploadBulkRetriesSlowDown(t *testing.T) {
	t.Parallel()
	db := testutil.NewDB(t, "media_bulk_upload")
	lo := logf.New(logf.Opts{})
	store := &flakyPutStore{}
	mm, err := New(Opts{Store: store, Lo: &lo, DB: db, I18n: testutil.NewI18n(t), RootURL: func() string { return "http://x" }})
	if err != nil {
		t.Fatal(err)
	}
	name, cType, err := mm.UploadBulk("id", models.NormalizeContentType("image/png"), strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "id" || cType != "image/png" {
		t.Fatalf("got %q %q", name, cType)
	}
	if store.calls != 3 {
		t.Fatalf("put calls = %d, want 3", store.calls)
	}
}

func TestUploadBulkTimeoutBudget(t *testing.T) {
	if UploadBulkTimeout() < MaxBulkPutWait() {
		t.Fatalf("timeout %s must cover backoff %s", UploadBulkTimeout(), MaxBulkPutWait())
	}
}
