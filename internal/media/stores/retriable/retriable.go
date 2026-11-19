// Package retriable classifies object-storage Put errors that may clear after backoff.
package retriable

import (
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var statusCodeRE = regexp.MustCompile(`(?i)status code:\s*(\d{3})`)

// Put reports whether repeating a storage Put after backoff may succeed (e.g. S3 SlowDown, HTTP 503).
func Put(err error) bool {
	for err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return true
		}
		if code := httpStatus(err); code > 0 {
			if code == http.StatusTooManyRequests || code == http.StatusServiceUnavailable || code >= http.StatusInternalServerError {
				return true
			}
			return false
		}
		switch strings.ToUpper(s3ErrorCode(err)) {
		case "SLOWDOWN", "REQUESTTIMEOUT", "THROTTLING", "SERVICEUNAVAILABLE":
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

func httpStatus(err error) int {
	m := statusCodeRE.FindStringSubmatch(err.Error())
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func s3ErrorCode(err error) string {
	msg := err.Error()
	i := strings.Index(msg, "<Code>")
	if i < 0 {
		return ""
	}
	msg = msg[i+len("<Code>"):]
	j := strings.Index(msg, "</Code>")
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(msg[:j])
}
