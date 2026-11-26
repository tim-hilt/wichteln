package main

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"net/http"
)

//go:embed web
var webFS embed.FS

var templates = template.Must(template.ParseFS(webFS, "web/templates/*.html"))

type app struct {
	dataDir string
	mailer  Mailer
	baseURL string
	random  io.Reader
}

// NewApp wires the application and returns its root handler. Everything
// environment-specific is injected: where data lives, how email is sent, the
// public base URL for links, and the randomness source for hashes and draws.
func NewApp(dataDir string, mailer Mailer, baseURL string, random io.Reader) http.Handler {
	a := &app{dataDir, mailer, baseURL, random}
	static, _ := fs.Sub(webFS, "web/static")
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /{$}", a.welcome)
	return mux
}

func (a *app) welcome(w http.ResponseWriter, r *http.Request) {
	render(w, "welcome.html", map[string]any{"T": messagesFor(r)})
}

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
