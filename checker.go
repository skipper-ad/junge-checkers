package junge

// CheckerInfo is returned by the "info" action.
type CheckerInfo struct {
	Vulns      int  `json:"vulns"`
	Timeout    int  `json:"timeout"`
	AttackData bool `json:"attack_data"`
	Puts       int  `json:"puts,omitempty"`
	Gets       int  `json:"gets,omitempty"`
}

func DefaultInfo() CheckerInfo {
	return CheckerInfo{
		Vulns:   1,
		Timeout: 10,
		Puts:    1,
		Gets:    1,
	}
}

func normalizeInfo(info *CheckerInfo) *CheckerInfo {
	base := DefaultInfo()
	if info == nil {
		return &base
	}
	if info.Vulns <= 0 {
		info.Vulns = base.Vulns
	}
	if info.Timeout <= 0 {
		info.Timeout = base.Timeout
	}
	if info.Puts < 0 {
		info.Puts = 0
	}
	if info.Gets < 0 {
		info.Gets = 0
	}
	return info
}

// Checker implements the immutable Skipper checker action contract.
type Checker interface {
	Info() *CheckerInfo
	Check(c *C, host string)
	Put(c *C, host, flagID, flag string, vuln int)
	Get(c *C, host, flagID, flag string, vuln int)
}

type PutRequest struct {
	Host   string
	FlagID string
	Flag   string
	Vuln   int
}

type GetRequest = PutRequest

// Handler lets a checker be described with functions instead of a custom type.
type Handler struct {
	Config    CheckerInfo
	CheckFunc func(c *C, host string)
	PutFunc   func(c *C, req PutRequest)
	GetFunc   func(c *C, req GetRequest)
}

func (h Handler) Info() *CheckerInfo {
	info := h.Config
	return normalizeInfo(&info)
}

func (h Handler) Check(c *C, host string) {
	if h.CheckFunc == nil {
		c.CheckFailed("Checker failed", "check handler is not implemented")
	}
	h.CheckFunc(c, host)
}

func (h Handler) Put(c *C, host, flagID, flag string, vuln int) {
	if h.PutFunc == nil {
		c.CheckFailed("Checker failed", "put handler is not implemented")
	}
	h.PutFunc(c, PutRequest{Host: host, FlagID: flagID, Flag: flag, Vuln: vuln})
}

func (h Handler) Get(c *C, host, flagID, flag string, vuln int) {
	if h.GetFunc == nil {
		c.CheckFailed("Checker failed", "get handler is not implemented")
	}
	h.GetFunc(c, GetRequest{Host: host, FlagID: flagID, Flag: flag, Vuln: vuln})
}
