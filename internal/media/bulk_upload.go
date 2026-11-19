package media

import (
	"errors"
	"io"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/media/stores/retriable"
)

const (
	bulkPutMaxAttempts = 8
	bulkPutMaxBackoff  = 60 * time.Second
	defaultBulkPutGap  = 350 * time.Millisecond
	maxBulkPutGap      = 2 * time.Second
)

// UploadBulk saves a file during bulk import jobs (e.g. Pylon) and retries transient object-store rate limits.
func (m *Manager) UploadBulk(fileName, contentType string, content io.ReadSeeker) (string, string, error) {
	m.paceBulkPut()
	m.lo.Debug("detecting content type for file before upload", "uuid", fileName, "source_content_type", contentType, "bulk", true)

	contentType, err := m.detectContentType(contentType, content)
	if err != nil {
		m.lo.Error("error detecting content type", "error", err, "file_name", fileName, "content_type", contentType, "store", m.store.Name())
		return "", "", err
	}

	for attempt := 0; ; attempt++ {
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			m.lo.Error("error rewinding media before upload", "error", err, "file_name", fileName, "store", m.store.Name())
			return "", "", err
		}
		fName, err := m.store.Put(fileName, contentType, content)
		if err == nil {
			return fName, contentType, nil
		}
		if attempt >= bulkPutMaxAttempts-1 || !retriable.Put(err) {
			m.lo.Error("error uploading media to store", "error", err, "file_name", fileName, "content_type", contentType, "store", m.store.Name(), "attempt", attempt+1)
			return "", "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.errorUploadingFile"), nil)
		}
		if retriable.SlowDown(err) {
			m.widenBulkPutGap()
		}
		wait := bulkPutBackoff(attempt, err)
		if attempt >= 2 || !retriable.SlowDown(err) {
			m.lo.Warn("object store refused upload, retrying", "file_name", fileName, "store", m.store.Name(), "attempt", attempt+1, "wait", wait, "error", err)
		} else {
			m.lo.Debug("object store refused upload, retrying", "file_name", fileName, "store", m.store.Name(), "attempt", attempt+1, "wait", wait)
		}
		time.Sleep(wait)
	}
}

func bulkPutBackoff(attempt int, err error) time.Duration {
	wait := time.Duration(1<<attempt) * time.Second
	if retriable.SlowDown(err) {
		wait *= 2
		if wait < 2*time.Second {
			wait = 2 * time.Second
		}
	}
	if wait > bulkPutMaxBackoff {
		wait = bulkPutMaxBackoff
	}
	return wait
}

func (m *Manager) paceBulkPut() {
	if m == nil {
		return
	}
	m.bulkPutMu.Lock()
	defer m.bulkPutMu.Unlock()
	gap := m.bulkPutMinGap
	if gap == 0 {
		gap = defaultBulkPutGap
	}
	if d := gap - time.Since(m.bulkPutLast); d > 0 {
		time.Sleep(d)
	}
	m.bulkPutLast = time.Now()
}

func (m *Manager) widenBulkPutGap() {
	m.bulkPutMu.Lock()
	defer m.bulkPutMu.Unlock()
	if m.bulkPutMinGap == 0 {
		m.bulkPutMinGap = defaultBulkPutGap
	}
	m.bulkPutMinGap += 150 * time.Millisecond
	if m.bulkPutMinGap > maxBulkPutGap {
		m.bulkPutMinGap = maxBulkPutGap
	}
}

// errSlowDownBudget is a representative error for worst-case UploadBulk backoff sizing.
var errSlowDownBudget = errors.New(`status code: 503 Service Unavailable: "<Code>SlowDown</Code>"`)

// MaxBulkPutWait is the longest a single UploadBulk may spend in backoff across all attempts.
func MaxBulkPutWait() time.Duration {
	var total time.Duration
	for attempt := 0; attempt < bulkPutMaxAttempts-1; attempt++ {
		total += bulkPutBackoff(attempt, errSlowDownBudget)
	}
	return total
}

// UploadBulkTimeout returns a deadline that fits UploadBulk retries plus one slow Put.
func UploadBulkTimeout() time.Duration {
	return 2*MaxBulkPutWait() + 3*time.Minute
}
