package junge

import (
	"bytes"
	"strings"
	"testing"
)

func TestContextMethodsFormatPrivateMessages(t *testing.T) {
	checker := Handler{
		CheckFunc: func(c *C, host string) {
			c.Mumblef("Bad response", "host=%s code=%d", host, 418)
		},
	}

	var stdout, stderr bytes.Buffer
	code := RunWithArgs(checker, []string{"check", "svc"}, &stdout, &stderr)
	if code != StatusMumble.Code() {
		t.Fatalf("code = %d, want MUMBLE", code)
	}
	if strings.TrimSpace(stdout.String()) != "Bad response" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.TrimSpace(stderr.String()) != "host=svc code=418" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestStructuredDetailsAreAddedToPrivateMessage(t *testing.T) {
	checker := Handler{
		CheckFunc: func(c *C, host string) {
			c.Detail("host", host)
			c.Detailf("attempt", "%02d", 3)
			c.Corrupt("Flag was corrupted")
		},
	}

	var stdout, stderr bytes.Buffer
	code := RunWithArgs(checker, []string{"check", "svc"}, &stdout, &stderr)
	if code != StatusCorrupt.Code() {
		t.Fatalf("code = %d, want CORRUPT", code)
	}
	want := "Flag was corrupted\nhost=svc\nattempt=03"
	if strings.TrimSpace(stderr.String()) != want {
		t.Fatalf("stderr = %q, want %q", strings.TrimSpace(stderr.String()), want)
	}
}

func TestPrivateDataDoesNotLeakToPublicOutput(t *testing.T) {
	const secret = "SECRET_PRIVATE_TOKEN"
	checker := Handler{
		CheckFunc: func(c *C, host string) {
			c.Detail("token", secret)
			c.Corrupt("Public safe message", "private="+secret)
		},
	}

	var stdout, stderr bytes.Buffer
	code := RunWithArgs(checker, []string{"check", "svc"}, &stdout, &stderr)
	if code != StatusCorrupt.Code() {
		t.Fatalf("code = %d, want CORRUPT", code)
	}
	if strings.Contains(stdout.String(), secret) {
		t.Fatalf("private secret leaked to stdout: %q", stdout.String())
	}
	if strings.TrimSpace(stdout.String()) != "Public safe message" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), secret) {
		t.Fatalf("stderr should contain private secret for platform diagnostics, got %q", stderr.String())
	}
}
