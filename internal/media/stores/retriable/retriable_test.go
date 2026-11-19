package retriable

import (
	"fmt"
	"net"
	"testing"
)

func TestPutHTTPStatus(t *testing.T) {
	if !Put(fmt.Errorf(`status code: 503 Service Unavailable: "<Code>SlowDown</Code>"`)) {
		t.Fatal("expected 503 retriable")
	}
	if Put(fmt.Errorf(`status code: 403 Forbidden: "access denied"`)) {
		t.Fatal("expected 403 permanent")
	}
	if !Put(fmt.Errorf(`status code: 429 Too Many Requests: "slow down"`)) {
		t.Fatal("expected 429 retriable")
	}
}

func TestPutS3Code(t *testing.T) {
	if !Put(fmt.Errorf(`other prefix: %q`, `<Error><Code>SlowDown</Code></Error>`)) {
		t.Fatal("expected SlowDown code")
	}
}

func TestPutNetTimeout(t *testing.T) {
	if !Put(&net.DNSError{IsTimeout: true}) {
		t.Fatal("expected timeout retriable")
	}
}
