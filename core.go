package junge

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type checkerFinished struct{}

type detail struct {
	key   string
	value string
}

// C is the checker context. It is intentionally close to testing.T:
// pass it to helpers, use c.Context for timeouts, and finish with c.OK/c.Down/etc.
type C struct {
	context.Context

	mu       sync.Mutex
	status   Status
	public   string
	private  string
	details  []detail
	finished bool
}

func newC(ctx context.Context) *C {
	return &C{
		Context: ctx,
		status:  StatusCheckFailed,
	}
}

func (c *C) SetPublic(message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.public = message
}

func (c *C) SetPrivate(message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.private = message
}

func (c *C) Public() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.public
}

func (c *C) Private() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.private
}

func (c *C) Detail(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.details = append(c.details, detail{key: key, value: fmt.Sprint(value)})
}

func (c *C) Detailf(key, format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.details = append(c.details, detail{key: key, value: fmt.Sprintf(format, args...)})
}

func (c *C) Details() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make(map[string]string, len(c.details))
	for _, item := range c.details {
		result[item.key] = item.value
	}
	return result
}

// Finish completes the checker action immediately.
func (c *C) Finish(status Status, public, private string, privateArgs ...any) {
	c.mu.Lock()
	c.status = status
	c.public = public
	c.private = privateWithDetails(privateMessage(public, private, privateArgs...), c.details)
	c.finished = true
	c.mu.Unlock()

	panic(checkerFinished{})
}

func (c *C) OK(public string, private ...string) {
	c.Finish(StatusOK, public, firstPrivate(public, private))
}

func (c *C) OKf(public, privateFormat string, privateArgs ...any) {
	c.Finish(StatusOK, public, privateFormat, privateArgs...)
}

func (c *C) Corrupt(public string, private ...string) {
	c.Finish(StatusCorrupt, public, firstPrivate(public, private))
}

func (c *C) Corruptf(public, privateFormat string, privateArgs ...any) {
	c.Finish(StatusCorrupt, public, privateFormat, privateArgs...)
}

func (c *C) Mumble(public string, private ...string) {
	c.Finish(StatusMumble, public, firstPrivate(public, private))
}

func (c *C) Mumblef(public, privateFormat string, privateArgs ...any) {
	c.Finish(StatusMumble, public, privateFormat, privateArgs...)
}

func (c *C) Down(public string, private ...string) {
	c.Finish(StatusDown, public, firstPrivate(public, private))
}

func (c *C) Downf(public, privateFormat string, privateArgs ...any) {
	c.Finish(StatusDown, public, privateFormat, privateArgs...)
}

func (c *C) CheckFailed(public string, private ...string) {
	c.Finish(StatusCheckFailed, public, firstPrivate(public, private))
}

func (c *C) CheckFailedf(public, privateFormat string, privateArgs ...any) {
	c.Finish(StatusCheckFailed, public, privateFormat, privateArgs...)
}

func firstPrivate(public string, values []string) string {
	if len(values) == 0 {
		return public
	}
	return values[0]
}

func privateMessage(public, private string, args ...any) string {
	if len(args) == 0 {
		return private
	}
	return fmt.Sprintf(private, args...)
}

func privateWithDetails(private string, details []detail) string {
	if len(details) == 0 {
		return private
	}
	var b strings.Builder
	b.WriteString(private)
	for _, item := range details {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(item.key)
		b.WriteString("=")
		b.WriteString(item.value)
	}
	return b.String()
}
