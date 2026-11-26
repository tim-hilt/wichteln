package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"
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
	mux.HandleFunc("POST /open-session", a.openSession)
	mux.HandleFunc("POST /open-link", a.openLink)
	mux.HandleFunc("GET /session/{hash}", a.session)
	mux.HandleFunc("POST /session/{hash}", a.submitSession)
	mux.HandleFunc("POST /session/{hash}/email", a.correctEmail)
	mux.HandleFunc("POST /session/{hash}/resend", a.resendLink)
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

// hashFrom extracts the hash from what a user pasted: either a bare hash or a
// full link (query, fragment and trailing slash are ignored).
func hashFrom(input string) string {
	input, _, _ = strings.Cut(strings.TrimSpace(input), "#")
	input, _, _ = strings.Cut(input, "?")
	input = strings.TrimRight(input, "/")
	return input[strings.LastIndex(input, "/")+1:]
}

// openSession redirects to the Session whose hash or link was pasted.
func (a *app) openSession(w http.ResponseWriter, r *http.Request) {
	a.open(w, r, func(hash string) (string, bool, error) {
		if !validHash(hash) {
			return "", false, nil
		}
		_, ok, err := a.store.Load(hash)
		return "/session/" + hash, ok, err
	})
}

// openLink redirects to the Personal link page whose hash or link was pasted.
func (a *app) openLink(w http.ResponseWriter, r *http.Request) {
	a.open(w, r, func(hash string) (string, bool, error) {
		sess, ok, err := a.store.LoadByPersonalLink(hash)
		return "/" + hash, ok && sess.participant(hash) != nil, err
	})
}

// open extracts the pasted hash and redirects to the path find reports as
// existing. An unknown hash re-renders the welcome page with an error and the
// input kept; nothing is ever created.
func (a *app) open(w http.ResponseWriter, r *http.Request, find func(hash string) (path string, ok bool, err error)) {
	input := r.PostFormValue("hash")
	path, ok, err := find(hashFrom(input))
	switch {
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	case !ok:
		t := messagesFor(r)
		render(w, http.StatusNotFound, "welcome.html", map[string]any{"T": t, "Error": t["not_found"], "Input": input, "Form": r.URL.Path})
	default:
		http.Redirect(w, r, path, http.StatusSeeOther)
	}
}
