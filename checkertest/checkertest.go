package checkertest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	junge "github.com/skipper-ad/junge-checkers"
)

type Result struct {
	Code   int
	Stdout string
	Stderr string
}

func Run(t testing.TB, checker junge.Checker, args ...string) Result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := junge.RunWithArgs(checker, args, &stdout, &stderr)
	return Result{
		Code:   code,
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}
}

func Info(t testing.TB, checker junge.Checker) Result {
	t.Helper()
	return Run(t, checker, "info")
}

func Check(t testing.TB, checker junge.Checker, host string) Result {
	t.Helper()
	return Run(t, checker, "check", host)
}

func Put(t testing.TB, checker junge.Checker, host, flagID, flag string, vuln int) Result {
	t.Helper()
	return Run(t, checker, "put", host, flagID, flag, fmt.Sprint(vuln))
}

func Get(t testing.TB, checker junge.Checker, host, flagID, flag string, vuln int) Result {
	t.Helper()
	return Run(t, checker, "get", host, flagID, flag, fmt.Sprint(vuln))
}

func (r Result) RequireStatus(t testing.TB, status junge.Status) {
	t.Helper()
	if r.Code != status.Code() {
		t.Fatalf("code = %d, want %d; stdout=%q stderr=%q", r.Code, status.Code(), r.Stdout, r.Stderr)
	}
}

func (r Result) RequireOK(t testing.TB) {
	t.Helper()
	r.RequireStatus(t, junge.StatusOK)
}

func (r Result) RequireCorrupt(t testing.TB) {
	t.Helper()
	r.RequireStatus(t, junge.StatusCorrupt)
}

func (r Result) RequireMumble(t testing.TB) {
	t.Helper()
	r.RequireStatus(t, junge.StatusMumble)
}

func (r Result) RequireDown(t testing.TB) {
	t.Helper()
	r.RequireStatus(t, junge.StatusDown)
}

func (r Result) RequireCheckFailed(t testing.TB) {
	t.Helper()
	r.RequireStatus(t, junge.StatusCheckFailed)
}

func (r Result) RequirePublic(t testing.TB, want string) {
	t.Helper()
	if r.Stdout != want {
		t.Fatalf("stdout = %q, want %q; code=%d stderr=%q", r.Stdout, want, r.Code, r.Stderr)
	}
}

func (r Result) RequirePrivate(t testing.TB, want string) {
	t.Helper()
	if r.Stderr != want {
		t.Fatalf("stderr = %q, want %q; code=%d stdout=%q", r.Stderr, want, r.Code, r.Stdout)
	}
}

func (r Result) RequirePrivateContains(t testing.TB, want string) {
	t.Helper()
	if !strings.Contains(r.Stderr, want) {
		t.Fatalf("stderr = %q, want to contain %q; code=%d stdout=%q", r.Stderr, want, r.Code, r.Stdout)
	}
}

func (r Result) Info(t testing.TB) junge.CheckerInfo {
	t.Helper()
	var info junge.CheckerInfo
	if err := json.Unmarshal([]byte(r.Stdout), &info); err != nil {
		t.Fatalf("unmarshal info from %q: %v", r.Stdout, err)
	}
	return info
}
