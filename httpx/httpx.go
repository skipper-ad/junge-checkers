package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	junge "github.com/skipper-ad/junge-checkers"
	"github.com/skipper-ad/junge-checkers/gen"
	o "github.com/skipper-ad/junge-checkers/require/options"
)

type Client struct {
	C              *junge.C
	BaseURL        string
	HTTP           *http.Client
	Header         http.Header
	Retry          RetryPolicy
	ErrorBodyBytes int
}

type Option func(*Client)

type RetryPolicy struct {
	Attempts int
	Delay    time.Duration
	Statuses map[int]bool
	Methods  map[string]bool
}

type File struct {
	FieldName   string
	FileName    string
	ContentType string
	Reader      io.Reader
}

func NewClient(c *junge.C, baseURL string, opts ...Option) *Client {
	client := &Client{
		C:       c,
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP: &http.Client{
			Timeout: 5 * time.Second,
		},
		Header: http.Header{
			"User-Agent": []string{gen.UserAgent()},
		},
		ErrorBodyBytes: 512,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.HTTP = httpClient
		}
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(client *Client) {
		client.HTTP.Timeout = timeout
	}
}

func WithCookieJar() Option {
	return func(client *Client) {
		jar, err := cookiejar.New(nil)
		if err != nil {
			client.C.CheckFailed("Checker failed", fmt.Sprintf("create cookie jar: %v", err))
		}
		client.HTTP.Jar = jar
	}
}

func WithHeader(key, value string) Option {
	return func(client *Client) {
		client.Header.Set(key, value)
	}
}

func WithBearerToken(token string) Option {
	return WithHeader("Authorization", "Bearer "+token)
}

func WithBasicAuth(username, password string) Option {
	return func(client *Client) {
		req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
		if err != nil {
			client.C.CheckFailed("Checker failed", fmt.Sprintf("build auth header: %v", err))
		}
		req.SetBasicAuth(username, password)
		client.Header.Set("Authorization", req.Header.Get("Authorization"))
	}
}

func WithRetries(attempts int, delay time.Duration, statuses ...int) Option {
	return func(client *Client) {
		if attempts < 1 {
			attempts = 1
		}
		statusSet := make(map[int]bool, len(statuses))
		for _, status := range statuses {
			statusSet[status] = true
		}
		if len(statusSet) == 0 {
			statusSet[http.StatusBadGateway] = true
			statusSet[http.StatusServiceUnavailable] = true
			statusSet[http.StatusGatewayTimeout] = true
		}
		client.Retry = RetryPolicy{
			Attempts: attempts,
			Delay:    delay,
			Statuses: statusSet,
			Methods: map[string]bool{
				http.MethodGet:     true,
				http.MethodHead:    true,
				http.MethodOptions: true,
			},
		}
	}
}

func WithErrorBodySnippet(bytes int) Option {
	return func(client *Client) {
		client.ErrorBodyBytes = bytes
	}
}

func (client *Client) URL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if path == "" {
		return client.BaseURL
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return client.BaseURL + path
}

func (client *Client) Get(path string) *http.Response {
	return client.Do(http.MethodGet, path, nil, "")
}

func (client *Client) PostJSON(path string, body any) *http.Response {
	data, err := json.Marshal(body)
	if err != nil {
		client.C.CheckFailed("Checker failed", fmt.Sprintf("marshal json request: %v", err))
	}
	return client.DoBytes(http.MethodPost, path, data, "application/json")
}

func (client *Client) PostForm(path string, values url.Values) *http.Response {
	return client.DoBytes(http.MethodPost, path, []byte(values.Encode()), "application/x-www-form-urlencoded")
}

func (client *Client) PostMultipart(path string, fields map[string]string, files ...File) *http.Response {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			client.C.CheckFailed("Checker failed", fmt.Sprintf("write multipart field %q: %v", key, err))
		}
	}
	for _, file := range files {
		if file.FieldName == "" {
			client.C.CheckFailed("Checker failed", "multipart file field name is empty")
		}
		fileName := file.FileName
		if fileName == "" {
			fileName = "file"
		}
		var part io.Writer
		var err error
		if file.ContentType == "" {
			part, err = writer.CreateFormFile(file.FieldName, fileName)
		} else {
			header := make(textproto.MIMEHeader)
			header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(file.FieldName), escapeQuotes(fileName)))
			header.Set("Content-Type", file.ContentType)
			part, err = writer.CreatePart(header)
		}
		if err != nil {
			client.C.CheckFailed("Checker failed", fmt.Sprintf("create multipart file %q: %v", file.FieldName, err))
		}
		if file.Reader == nil {
			client.C.CheckFailed("Checker failed", fmt.Sprintf("multipart file %q reader is nil", file.FieldName))
		}
		if _, err := io.Copy(part, file.Reader); err != nil {
			client.C.CheckFailed("Checker failed", fmt.Sprintf("copy multipart file %q: %v", file.FieldName, err))
		}
	}
	if err := writer.Close(); err != nil {
		client.C.CheckFailed("Checker failed", fmt.Sprintf("close multipart body: %v", err))
	}
	return client.DoBytes(http.MethodPost, path, body.Bytes(), writer.FormDataContentType())
}

func (client *Client) Do(method, path string, body io.Reader, contentType string) *http.Response {
	var data []byte
	if body != nil {
		var err error
		data, err = io.ReadAll(body)
		if err != nil {
			client.C.CheckFailed("Checker failed", fmt.Sprintf("read request body: %v", err))
		}
	}
	return client.DoBytes(method, path, data, contentType)
}

func (client *Client) DoBytes(method, path string, body []byte, contentType string) *http.Response {
	attempts := client.Retry.Attempts
	if attempts < 1 || !client.Retry.Methods[method] {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		resp, err := client.doOnce(method, path, body, contentType)
		if err == nil {
			if attempt < attempts && client.Retry.Statuses[resp.StatusCode] {
				closeBody(resp)
				if !client.sleepBeforeRetry() {
					client.C.Down("Service is down", "retry interrupted by checker timeout")
				}
				continue
			}
			return resp
		}
		lastErr = err
		if attempt < attempts && client.sleepBeforeRetry() {
			continue
		}
		break
	}
	client.C.Down("Service is down", fmt.Sprintf("%s %s: %v", method, client.URL(path), lastErr))
	return nil
}

func (client *Client) doOnce(method, path string, body []byte, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(client.C, method, client.URL(path), bytes.NewReader(body))
	if err != nil {
		client.C.CheckFailed("Checker failed", fmt.Sprintf("build request: %v", err))
	}
	for key, values := range client.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.HTTP.Do(req)
	return resp, err
}

func (client *Client) sleepBeforeRetry() bool {
	if client.Retry.Delay <= 0 {
		return true
	}
	timer := time.NewTimer(client.Retry.Delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-client.C.Done():
		return false
	}
}

func (client *Client) CheckResponse(resp *http.Response, public string, opts ...o.Option) {
	checkResponse(client.C, resp, public, client.ErrorBodyBytes, opts...)
}

func (client *Client) ExpectStatus(resp *http.Response, status int, public string, opts ...o.Option) {
	expectStatus(client.C, resp, status, public, client.ErrorBodyBytes, opts...)
}

func (client *Client) JSON(resp *http.Response, target any, public string, opts ...o.Option) {
	JSON(client.C, resp, target, public, opts...)
}

func (client *Client) Text(resp *http.Response, public string, opts ...o.Option) string {
	return Text(client.C, resp, public, opts...)
}

func CheckResponse(c *junge.C, resp *http.Response, public string, opts ...o.Option) {
	checkResponse(c, resp, public, 0, opts...)
}

func CheckResponseWithBody(c *junge.C, resp *http.Response, public string, snippetBytes int, opts ...o.Option) {
	checkResponse(c, resp, public, snippetBytes, opts...)
}

func checkResponse(c *junge.C, resp *http.Response, public string, snippetBytes int, opts ...o.Option) {
	if resp == nil {
		c.Down(public, "nil http response")
	}
	if resp.StatusCode >= 500 {
		c.Down(public, responseError(resp, snippetBytes, "HTTP %d on %s", resp.StatusCode, responseURL(resp)))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		info := o.GetExitInfo(public, responseError(resp, snippetBytes, "HTTP %d on %s", resp.StatusCode, responseURL(resp)), opts...)
		c.Finish(info.Status, info.Public, info.Private)
	}
}

func ExpectStatus(c *junge.C, resp *http.Response, status int, public string, opts ...o.Option) {
	expectStatus(c, resp, status, public, 0, opts...)
}

func ExpectStatusWithBody(c *junge.C, resp *http.Response, status int, public string, snippetBytes int, opts ...o.Option) {
	expectStatus(c, resp, status, public, snippetBytes, opts...)
}

func expectStatus(c *junge.C, resp *http.Response, status int, public string, snippetBytes int, opts ...o.Option) {
	if resp == nil {
		c.Down(public, "nil http response")
	}
	if resp.StatusCode != status {
		if resp.StatusCode >= 500 {
			c.Down(public, responseError(resp, snippetBytes, "expected HTTP %d, got %d on %s", status, resp.StatusCode, responseURL(resp)))
		}
		info := o.GetExitInfo(public, responseError(resp, snippetBytes, "expected HTTP %d, got %d on %s", status, resp.StatusCode, responseURL(resp)), opts...)
		c.Finish(info.Status, info.Public, info.Private)
	}
}

func JSON(c *junge.C, resp *http.Response, target any, public string, opts ...o.Option) {
	defer closeBody(resp)
	CheckResponse(c, resp, public, opts...)
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		info := o.GetExitInfo(public, fmt.Sprintf("decode json from %s: %v", responseURL(resp), err), opts...)
		c.Finish(info.Status, info.Public, info.Private)
	}
}

func Text(c *junge.C, resp *http.Response, public string, opts ...o.Option) string {
	defer closeBody(resp)
	CheckResponse(c, resp, public, opts...)
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		info := o.GetExitInfo(public, fmt.Sprintf("read body from %s: %v", responseURL(resp), err), opts...)
		c.Finish(info.Status, info.Public, info.Private)
	}
	return string(data)
}

func responseURL(resp *http.Response) string {
	if resp == nil || resp.Request == nil || resp.Request.URL == nil {
		return "<unknown>"
	}
	value := *resp.Request.URL
	value.User = nil
	return value.String()
}

func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

func responseError(resp *http.Response, snippetBytes int, format string, args ...any) string {
	message := fmt.Sprintf(format, args...)
	if snippetBytes <= 0 || resp == nil || resp.Body == nil {
		return message
	}
	snippet := BodySnippet(resp, snippetBytes)
	if snippet == "" {
		return message
	}
	return message + "\nbody=" + snippet
}

func BodySnippet(resp *http.Response, limit int) string {
	if resp == nil || resp.Body == nil || limit <= 0 {
		return ""
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, int64(limit)))
	_ = resp.Body.Close()
	return strings.TrimSpace(string(data))
}

type Service struct {
	Config        junge.CheckerInfo
	Scheme        string
	Port          int
	BaseURL       func(host string) string
	ClientOptions []Option

	CheckFunc func(c *junge.C, api *Client)
	PutFunc   func(c *junge.C, api *Client, req junge.PutRequest)
	GetFunc   func(c *junge.C, api *Client, req junge.GetRequest)
}

func (s Service) Info() *junge.CheckerInfo {
	info := s.Config
	return &info
}

func (s Service) Check(c *junge.C, host string) {
	if s.CheckFunc == nil {
		c.CheckFailed("Checker failed", "service check handler is not implemented")
	}
	s.CheckFunc(c, s.client(c, host))
}

func (s Service) Put(c *junge.C, host, flagID, flag string, vuln int) {
	if s.PutFunc == nil {
		c.CheckFailed("Checker failed", "service put handler is not implemented")
	}
	req := junge.PutRequest{Host: host, FlagID: flagID, Flag: flag, Vuln: vuln}
	s.PutFunc(c, s.client(c, host), req)
}

func (s Service) Get(c *junge.C, host, flagID, flag string, vuln int) {
	if s.GetFunc == nil {
		c.CheckFailed("Checker failed", "service get handler is not implemented")
	}
	req := junge.GetRequest{Host: host, FlagID: flagID, Flag: flag, Vuln: vuln}
	s.GetFunc(c, s.client(c, host), req)
}

func (s Service) client(c *junge.C, host string) *Client {
	if s.BaseURL != nil {
		return NewClient(c, s.BaseURL(host), s.ClientOptions...)
	}
	scheme := s.Scheme
	if scheme == "" {
		scheme = "http"
	}
	if s.Port > 0 {
		return NewClient(c, fmt.Sprintf("%s://%s:%d", scheme, host, s.Port), s.ClientOptions...)
	}
	return NewClient(c, fmt.Sprintf("%s://%s", scheme, host), s.ClientOptions...)
}

func escapeQuotes(value string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(value)
}
