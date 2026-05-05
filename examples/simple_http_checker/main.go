package main

import (
	"fmt"
	"net/url"
	"os"

	junge "github.com/skipper-ad/junge-checkers"
	"github.com/skipper-ad/junge-checkers/httpx"
	"github.com/skipper-ad/junge-checkers/require"
	o "github.com/skipper-ad/junge-checkers/require/options"
)

type createNoteResponse struct {
	ID string `json:"id"`
}

type noteResponse struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func main() {
	junge.Main(checker())
}

func checker() junge.Handler {
	return junge.Handler{
		Config: junge.CheckerInfo{
			Vulns:      1,
			Timeout:    10,
			AttackData: true,
			Puts:       1,
			Gets:       1,
		},
		CheckFunc: check,
		PutFunc:   put,
		GetFunc:   get,
	}
}

func check(c *junge.C, host string) {
	api := httpx.NewClient(c, serviceBaseURL(host))

	resp := api.Get("/health")
	httpx.ExpectStatus(c, resp, 200, "Service is unhealthy")
	c.OK("OK")
}

func put(c *junge.C, req junge.PutRequest) {
	api := httpx.NewClient(c, serviceBaseURL(req.Host))

	var created createNoteResponse
	resp := api.PostJSON("/api/notes", map[string]string{
		"text": req.Flag,
	})
	httpx.JSON(c, resp, &created, "Could not save flag", o.Corrupt())
	require.NotEqual(c, "", created.ID, "Could not save flag", o.Corrupt())

	c.OK(created.ID, fmt.Sprintf("saved flag_id=%s vuln=%d", created.ID, req.Vuln))
}

func get(c *junge.C, req junge.GetRequest) {
	api := httpx.NewClient(c, serviceBaseURL(req.Host))

	var note noteResponse
	path := "/api/notes/" + url.PathEscape(req.FlagID)
	resp := api.Get(path)
	httpx.JSON(c, resp, &note, "Could not read flag", o.Corrupt())
	require.Equal(c, req.Flag, note.Text, "Flag was corrupted", o.Corrupt())

	c.OK("OK", fmt.Sprintf("read flag_id=%s vuln=%d", req.FlagID, req.Vuln))
}

func serviceBaseURL(host string) string {
	if value := os.Getenv("JUNGE_EXAMPLE_BASE_URL"); value != "" {
		return value
	}
	return fmt.Sprintf("http://%s:8080", host)
}
