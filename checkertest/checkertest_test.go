package checkertest

import (
	"testing"

	junge "github.com/skipper-ad/junge-checkers"
)

func TestCheckertestHelpers(t *testing.T) {
	checker := junge.Handler{
		Config: junge.CheckerInfo{Vulns: 2, Timeout: 5},
		CheckFunc: func(c *junge.C, host string) {
			c.Detail("host", host)
			c.OK("OK")
		},
		PutFunc: func(c *junge.C, req junge.PutRequest) {
			c.OK(req.FlagID)
		},
		GetFunc: func(c *junge.C, req junge.GetRequest) {
			c.Corrupt("Flag was corrupted", "missing "+req.FlagID)
		},
	}

	info := Info(t, checker)
	info.RequireOK(t)
	parsed := info.Info(t)
	if parsed.Vulns != 2 || parsed.Timeout != 5 {
		t.Fatalf("info = %+v", parsed)
	}

	check := Check(t, checker, "127.0.0.1")
	check.RequireOK(t)
	check.RequirePublic(t, "OK")
	check.RequirePrivateContains(t, "host=127.0.0.1")

	put := Put(t, checker, "127.0.0.1", "id-1", "FLAG", 1)
	put.RequireOK(t)
	put.RequirePublic(t, "id-1")

	get := Get(t, checker, "127.0.0.1", "id-1", "FLAG", 1)
	get.RequireCorrupt(t)
	get.RequirePrivate(t, "missing id-1")
}
