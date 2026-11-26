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
	mux.HandleFunc("POST /open-session", a.open("/session/", store.Load))
	mux.HandleFunc("POST /open-link", a.open("/", store.LoadByPersonalLink))
	mux.HandleFunc("POST /open-join", a.open("/join/", store.LoadByJoinLink))
	mux.HandleFunc("GET /session/{hash}", a.session)
	mux.HandleFunc("POST /session/{hash}", a.submitSession)
	mux.HandleFunc("POST /session/{hash}/resend", a.resendLink)
	mux.HandleFunc("POST /session/{hash}/release", a.releaseParticipant)
	mux.HandleFunc("GET /join/{hash}", a.joinPage)
	mux.HandleFunc("POST /join/{hash}", a.submitJoin)
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
		internalError(w)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func internalError(w http.ResponseWriter) {
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// renderNotFound shows the 404 page for an unknown Personal or Join link.
func renderNotFound(w http.ResponseWriter, r *http.Request) {
	render(w, http.StatusNotFound, "not_found.html", map[string]any{"T": messagesFor(r)})
}

// hashFrom extracts the hash from what a user pasted: either a bare hash or a
// full link (query, fragment and trailing slash are ignored).
func hashFrom(input string) string {
	input, _, _ = strings.Cut(strings.TrimSpace(input), "#")
	input, _, _ = strings.Cut(input, "?")
	input = strings.TrimRight(input, "/")
	return input[strings.LastIndex(input, "/")+1:]
}

// open returns a handler that extracts the pasted hash and, if load finds it,
// redirects to prefix+hash. An unknown hash re-renders the welcome page with
// an error and the input kept; nothing is ever created.
func (a *app) open(prefix string, load func(hash string) (Session, bool, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		input := r.PostFormValue("hash")
		hash := hashFrom(input)
		_, ok, err := load(hash)
		switch {
		case err != nil:
			internalError(w)
		case !ok:
			t := messagesFor(r)
			render(w, http.StatusNotFound, "welcome.html", map[string]any{"T": t, "Error": t["not_found"], "Input": input, "Form": r.URL.Path})
		default:
			http.Redirect(w, r, prefix+hash, http.StatusSeeOther)
		}
	}
}
