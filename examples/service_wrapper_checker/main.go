package main

import (
	"fmt"
	"net/url"
	"time"

	junge "github.com/skipper-ad/junge-checkers"
	"github.com/skipper-ad/junge-checkers/httpx"
	"github.com/skipper-ad/junge-checkers/require"
	o "github.com/skipper-ad/junge-checkers/require/options"
)

type createResponse struct {
	ID string `json:"id"`
}

type flagResponse struct {
	ID   string `json:"id"`
	Flag string `json:"flag"`
}

func main() {
	junge.Main(httpx.Service{
		Config: junge.CheckerInfo{
			Vulns:      2,
			Timeout:    10,
			AttackData: true,
			Puts:       1,
			Gets:       1,
		},
		Port: 8080,
		ClientOptions: []httpx.Option{
			httpx.WithCookieJar(),
			httpx.WithRetries(2, 100*time.Millisecond),
		},
		CheckFunc: check,
		PutFunc:   put,
		GetFunc:   get,
	})
}

func check(c *junge.C, api *httpx.Client) {
	api.ExpectStatus(api.Get("/health"), 200, "Service is unhealthy")
	c.OK("OK")
}

func put(c *junge.C, api *httpx.Client, req junge.PutRequest) {
	c.Detail("vuln", req.Vuln)

	var created createResponse
	api.JSON(api.PostJSON("/api/flags", map[string]any{
		"vuln": req.Vuln,
		"flag": req.Flag,
	}), &created, "Could not save flag", o.Corrupt())
	require.NotEqual(c, "", created.ID, "Could not save flag", o.Corrupt())

	c.OK(created.ID, fmt.Sprintf("saved flag_id=%s", created.ID))
}

func get(c *junge.C, api *httpx.Client, req junge.GetRequest) {
	c.Detail("flag_id", req.FlagID)
	c.Detail("vuln", req.Vuln)

	var saved flagResponse
	api.JSON(api.Get("/api/flags/"+url.PathEscape(req.FlagID)), &saved, "Could not read flag", o.Corrupt())
	require.Equal(c, req.Flag, saved.Flag, "Flag was corrupted", o.Corrupt())

	c.OK("OK")
}
