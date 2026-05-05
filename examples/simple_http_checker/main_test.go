package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/skipper-ad/junge-checkers/checkertest"
)

func TestExampleCheckerContract(t *testing.T) {
	server := newNotesServer()
	defer server.Close()
	t.Setenv("JUNGE_EXAMPLE_BASE_URL", server.URL)

	info := checkertest.Info(t, checker())
	info.RequireOK(t)
	info.RequirePublic(t, `{"vulns":1,"timeout":10,"attack_data":true,"puts":1,"gets":1}`)
	info.RequirePrivate(t, "")

	check := checkertest.Check(t, checker(), "127.0.0.1")
	check.RequireOK(t)
	check.RequirePublic(t, "OK")
	check.RequirePrivate(t, "OK")

	put := checkertest.Put(t, checker(), "127.0.0.1", "initial-id", "FLAG-123", 1)
	put.RequireOK(t)
	put.RequirePublic(t, "note-1")
	put.RequirePrivate(t, "saved flag_id=note-1 vuln=1")

	get := checkertest.Get(t, checker(), "127.0.0.1", "note-1", "FLAG-123", 1)
	get.RequireOK(t)
	get.RequirePublic(t, "OK")
	get.RequirePrivate(t, "read flag_id=note-1 vuln=1")
}

func newNotesServer() *httptest.Server {
	type note struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	}

	var (
		mu    sync.Mutex
		notes []note
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	mux.HandleFunc("/api/notes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}

		mu.Lock()
		created := note{ID: fmt.Sprintf("note-%d", len(notes)+1), Text: body.Text}
		notes = append(notes, created)
		mu.Unlock()

		_ = json.NewEncoder(w).Encode(createNoteResponse{ID: created.ID})
	})
	mux.HandleFunc("/api/notes/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/notes/")

		mu.Lock()
		defer mu.Unlock()
		for _, saved := range notes {
			if saved.ID == id {
				_ = json.NewEncoder(w).Encode(noteResponse{ID: saved.ID, Text: saved.Text})
				return
			}
		}
		http.NotFound(w, r)
	})
	return httptest.NewServer(mux)
}
