package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	junge "github.com/skipper-ad/junge-checkers"
	o "github.com/skipper-ad/junge-checkers/require/options"
)

func TestHTTPXJSONSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Fatalf("missing User-Agent")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	checker := junge.Handler{
		CheckFunc: func(c *junge.C, host string) {
			api := NewClient(c, server.URL)
			var body struct {
				Status string `json:"status"`
			}
			JSON(c, api.Get("/health"), &body, "Bad health")
			if body.Status != "ok" {
				c.Mumblef("Bad health", "status=%q", body.Status)
			}
			c.OK("OK")
		},
	}

	var stdout, stderr bytes.Buffer
	code := junge.RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != junge.StatusOK.Code() {
		t.Fatalf("code = %d, want OK; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestHTTPXServerErrorIsDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := junge.Handler{
		CheckFunc: func(c *junge.C, host string) {
			api := NewClient(c, server.URL)
			CheckResponse(c, api.Get("/"), "Service is down")
			c.OK("unreachable")
		},
	}

	var stdout, stderr bytes.Buffer
	code := junge.RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != junge.StatusDown.Code() {
		t.Fatalf("code = %d, want DOWN", code)
	}
	if !strings.Contains(stderr.String(), "HTTP 500") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestHTTPXClientErrorCanBeCorrupt(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	checker := junge.Handler{
		CheckFunc: func(c *junge.C, host string) {
			api := NewClient(c, server.URL)
			CheckResponse(c, api.Get("/flag"), "Flag was corrupted", o.Corrupt())
			c.OK("unreachable")
		},
	}

	var stdout, stderr bytes.Buffer
	code := junge.RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != junge.StatusCorrupt.Code() {
		t.Fatalf("code = %d, want CORRUPT; stderr=%q", code, stderr.String())
	}
}

func TestResponseURLRemovesCredentials(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://user:pass@example.test/path", nil)
	got := responseURL(&http.Response{Request: req})
	if strings.Contains(got, "user") || strings.Contains(got, "pass") {
		t.Fatalf("responseURL leaked credentials: %q", got)
	}
}

func TestHTTPXServiceWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	checker := Service{
		Config:  junge.CheckerInfo{Vulns: 1, Timeout: 5},
		BaseURL: func(host string) string { return server.URL },
		CheckFunc: func(c *junge.C, api *Client) {
			api.ExpectStatus(api.Get("/health"), http.StatusOK, "Service is unhealthy")
			c.OK("OK")
		},
	}

	var stdout, stderr bytes.Buffer
	code := junge.RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != junge.StatusOK.Code() {
		t.Fatalf("code = %d, want OK; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestHTTPXRetriesAndBodySnippet(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			http.Error(w, "temporary failure", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	checker := junge.Handler{
		CheckFunc: func(c *junge.C, host string) {
			api := NewClient(c, server.URL, WithRetries(2, time.Millisecond, http.StatusServiceUnavailable))
			api.ExpectStatus(api.Get("/"), http.StatusOK, "Service is unhealthy")
			c.OK("OK")
		},
	}

	var stdout, stderr bytes.Buffer
	code := junge.RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != junge.StatusOK.Code() {
		t.Fatalf("code = %d, want OK; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}

	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "flag disappeared", http.StatusNotFound)
	}))
	defer errorServer.Close()
	checker = junge.Handler{
		CheckFunc: func(c *junge.C, host string) {
			api := NewClient(c, errorServer.URL, WithErrorBodySnippet(64))
			api.CheckResponse(api.Get("/flag"), "Flag was corrupted", o.Corrupt())
		},
	}
	stdout.Reset()
	stderr.Reset()
	code = junge.RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != junge.StatusCorrupt.Code() {
		t.Fatalf("code = %d, want CORRUPT; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "body=flag disappeared") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestHTTPXFormsMultipartCookiesAndAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/form":
			if r.Header.Get("Authorization") == "" {
				http.Error(w, "missing auth", http.StatusUnauthorized)
				return
			}
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: "cookie-1"})
			_, _ = fmt.Fprint(w, r.Form.Get("login"))
		case "/multipart":
			if _, err := r.Cookie("sid"); err != nil {
				http.Error(w, "missing cookie", http.StatusUnauthorized)
				return
			}
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			file, _, err := r.FormFile("upload")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer file.Close()
			data, _ := io.ReadAll(file)
			_, _ = w.Write(data)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	checker := junge.Handler{
		CheckFunc: func(c *junge.C, host string) {
			api := NewClient(c, server.URL, WithCookieJar(), WithBearerToken("secret"))
			form := api.Text(api.PostForm("/form", url.Values{"login": []string{"alice"}}), "Bad form")
			if form != "alice" {
				c.Mumblef("Bad form", "form response=%q", form)
			}
			resp := api.PostMultipart("/multipart", map[string]string{"kind": "flag"}, File{
				FieldName:   "upload",
				FileName:    "flag.txt",
				ContentType: "text/plain",
				Reader:      strings.NewReader("FLAG"),
			})
			text := api.Text(resp, "Bad multipart")
			if text != "FLAG" {
				c.Mumblef("Bad multipart", "multipart response=%q", text)
			}
			c.OK("OK")
		},
	}

	var stdout, stderr bytes.Buffer
	code := junge.RunWithArgs(checker, []string{"check", "127.0.0.1"}, &stdout, &stderr)
	if code != junge.StatusOK.Code() {
		t.Fatalf("code = %d, want OK; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
