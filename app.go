package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
)

//go:embed web
var webFS embed.FS

var templates = template.Must(template.ParseFS(webFS, "web/templates/*.html"))

type app struct {
	store   *Store
	mailer  Mailer
	baseURL string
	random  io.Reader
}

// NewApp wires the application and returns its root handler. Everything
// environment-specific is injected: where data lives, how email is sent, the
// public base URL for links, and the randomness source for hashes and draws.
func NewApp(dataDir string, mailer Mailer, baseURL string, random io.Reader) http.Handler {
	store, err := NewStore(dataDir)
	if err != nil {
		panic(fmt.Sprintf("cannot open data directory %q: %v", dataDir, err))
	}
	a := &app{store, mailer, baseURL, random}
	static, _ := fs.Sub(webFS, "web/static")
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /{$}", a.welcome)
	mux.HandleFunc("POST /new", a.createSessionLink)
	mux.HandleFunc("GET /session/{hash}", a.session)
	mux.HandleFunc("POST /session/{hash}", a.submitSession)
	mux.HandleFunc("GET /participant-row", a.participantRow)
	mux.HandleFunc("GET /{hash}", a.personal)
	mux.HandleFunc("POST /{hash}", a.submitPersonal)
	return mux
}

func (a *app) welcome(w http.ResponseWriter, r *http.Request) {
	render(w, http.StatusOK, "welcome.html", map[string]any{"T": messagesFor(r)})
}

func render(w http.ResponseWriter, status int, name string, data any) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}
