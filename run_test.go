package junge

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRunWithArgsInfo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := RunWithArgs(Handler{Config: CheckerInfo{Vulns: 2, Timeout: 7, AttackData: true}}, []string{"info"}, &stdout, &stderr)
	if code != 101 {
		t.Fatalf("code = %d, want 101; stderr=%q", code, stderr.String())
	}
	var info CheckerInfo
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &info); err != nil {
		t.Fatalf("unmarshal info: %v; stdout=%q", err, stdout.String())
	}
	if info.Vulns != 2 || info.Timeout != 7 || !info.AttackData {
		t.Fatalf("info = %+v", info)
	}
}

func TestRunWithArgsCheck(t *testing.T) {
	checker := Handler{
		CheckFunc: func(c *C, host string) {
			if host != "127.0.0.1" {
				c.Mumble("bad host")
			}
			c.OK("OK", "checked")
		},
	}

	var stdout, stderr bytes.Buffer
	code := RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != 101 {
		t.Fatalf("code = %d, want 101", code)
	}
	if strings.TrimSpace(stdout.String()) != "OK" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.TrimSpace(stderr.String()) != "checked" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunWithArgsPutAndGetArguments(t *testing.T) {
	var seenPut PutRequest
	var seenGet GetRequest
	checker := Handler{
		PutFunc: func(c *C, req PutRequest) {
			seenPut = req
			c.OK(req.FlagID, "put private")
		},
		GetFunc: func(c *C, req GetRequest) {
			seenGet = req
			c.OK("OK", "get private")
		},
	}

	var stdout, stderr bytes.Buffer
	code := RunWithArgs(checker, []string{"put", "10.0.0.1", "note-42", "FLAG", "3"}, &stdout, &stderr)
	if code != StatusOK.Code() {
		t.Fatalf("put code = %d, want OK; stderr=%q", code, stderr.String())
	}
	if seenPut != (PutRequest{Host: "10.0.0.1", FlagID: "note-42", Flag: "FLAG", Vuln: 3}) {
		t.Fatalf("put request = %+v", seenPut)
	}
	if strings.TrimSpace(stdout.String()) != "note-42" {
		t.Fatalf("put stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = RunWithArgs(checker, []string{"get", "10.0.0.2", "note-43", "FLAG2", "4"}, &stdout, &stderr)
	if code != StatusOK.Code() {
		t.Fatalf("get code = %d, want OK; stderr=%q", code, stderr.String())
	}
	if seenGet != (GetRequest{Host: "10.0.0.2", FlagID: "note-43", Flag: "FLAG2", Vuln: 4}) {
		t.Fatalf("get request = %+v", seenGet)
	}
}

func TestRunWithArgsRejectsBadContractArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing action", args: nil, want: "missing action"},
		{name: "extra check arg", args: []string{"check", "127.0.0.1", "extra"}, want: "check expects 1 argument"},
		{name: "bad vuln", args: []string{"put", "127.0.0.1", "id", "FLAG", "nope"}, want: `invalid vuln number "nope"`},
		{name: "extra info arg", args: []string{"info", "extra"}, want: "info expects no arguments"},
		{name: "unknown action", args: []string{"wat"}, want: "bad action: wat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := RunWithArgs(Handler{}, tt.args, &stdout, &stderr)
			if code != StatusCheckFailed.Code() {
				t.Fatalf("code = %d, want CHECK FAILED; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want to contain %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestRunWithArgsPanicIsCheckFailed(t *testing.T) {
	checker := Handler{
		CheckFunc: func(c *C, host string) {
			panic("boom")
		},
	}

	var stdout, stderr bytes.Buffer
	code := RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != StatusCheckFailed.Code() {
		t.Fatalf("code = %d, want CHECK FAILED", code)
	}
	if strings.TrimSpace(stdout.String()) != "Checker failed" || !strings.Contains(stderr.String(), "panic: boom") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunWithArgsTimeoutIsDown(t *testing.T) {
	checker := Handler{
		Config: CheckerInfo{Timeout: 1},
		CheckFunc: func(c *C, host string) {
			time.Sleep(1500 * time.Millisecond)
			c.OK("late")
		},
	}

	var stdout, stderr bytes.Buffer
	start := time.Now()
	code := RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if elapsed := time.Since(start); elapsed > 1400*time.Millisecond {
		t.Fatalf("timeout returned too late: %s", elapsed)
	}
	if code != StatusDown.Code() {
		t.Fatalf("code = %d, want DOWN; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "Checker timed out" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunWithArgsNilChecker(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := RunWithArgs(nil, []string{"info"}, &stdout, &stderr)
	if code != StatusCheckFailed.Code() {
		t.Fatalf("code = %d, want CHECK FAILED", code)
	}
	if !strings.Contains(stderr.String(), "checker is nil") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

type panicInfoChecker struct{}

func (panicInfoChecker) Info() *CheckerInfo {
	panic("bad config")
}

func (panicInfoChecker) Check(c *C, host string)                       {}
func (panicInfoChecker) Put(c *C, host, flagID, flag string, vuln int) {}
func (panicInfoChecker) Get(c *C, host, flagID, flag string, vuln int) {}

func TestRunWithArgsInfoPanicIsCheckFailed(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := RunWithArgs(panicInfoChecker{}, []string{"info"}, &stdout, &stderr)
	if code != StatusCheckFailed.Code() {
		t.Fatalf("code = %d, want CHECK FAILED", code)
	}
	if !strings.Contains(stderr.String(), "checker info panic: bad config") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
