package media

import (
	"io"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/media/stores/retriable"
)

const (
	bulkPutMaxAttempts = 6
	bulkPutMaxBackoff  = 30 * time.Second
)

// UploadBulk saves a file during bulk import jobs (e.g. Pylon) and retries transient object-store rate limits.
func (m *Manager) UploadBulk(fileName, contentType string, content io.ReadSeeker) (string, string, error) {
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
		wait := time.Duration(1<<attempt) * time.Second
		if wait > bulkPutMaxBackoff {
			wait = bulkPutMaxBackoff
		}
		m.lo.Warn("object store refused upload, retrying", "file_name", fileName, "store", m.store.Name(), "attempt", attempt+1, "wait", wait, "error", err)
		time.Sleep(wait)
	}
}

// MaxBulkPutWait is the longest a single UploadBulk may spend in backoff across all attempts.
func MaxBulkPutWait() time.Duration {
	var total time.Duration
	for attempt := 0; attempt < bulkPutMaxAttempts-1; attempt++ {
		wait := time.Duration(1<<attempt) * time.Second
		if wait > bulkPutMaxBackoff {
			wait = bulkPutMaxBackoff
		}
		total += wait
	}
	return total
}

// UploadBulkTimeout returns a deadline that fits UploadBulk retries plus one slow Put.
func UploadBulkTimeout() time.Duration {
	return 2*MaxBulkPutWait() + 2*time.Minute
}
