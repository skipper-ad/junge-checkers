package require

import (
	"bytes"
	"strings"
	"testing"

	junge "github.com/skipper-ad/junge-checkers"
	o "github.com/skipper-ad/junge-checkers/require/options"
)

func TestRequireDefaultStatusIsMumble(t *testing.T) {
	checker := junge.Handler{
		CheckFunc: func(c *junge.C, host string) {
			Equal(c, "expected", "actual", "Bad response")
		},
	}

	var stdout, stderr bytes.Buffer
	code := junge.RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != junge.StatusMumble.Code() {
		t.Fatalf("code = %d, want MUMBLE", code)
	}
	if strings.TrimSpace(stdout.String()) != "Bad response" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "expected") || !strings.Contains(stderr.String(), "actual") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRequireStatusAndPrivateOptions(t *testing.T) {
	checker := junge.Handler{
		CheckFunc: func(c *junge.C, host string) {
			Contains(c, "abc", "z", "Flag was corrupted", o.Corrupt(), o.Privatef("missing flag_id=%s", "42"))
		},
	}

	var stdout, stderr bytes.Buffer
	code := junge.RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != junge.StatusCorrupt.Code() {
		t.Fatalf("code = %d, want CORRUPT", code)
	}
	if strings.TrimSpace(stdout.String()) != "Flag was corrupted" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.TrimSpace(stderr.String()) != "missing flag_id=42" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
