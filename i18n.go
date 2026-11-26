package main

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// Messages maps a message key to its text in one language. Templates read
// them as {{.T.key}}.
type Messages map[string]string

// catalog holds all visible text, per language. Every language must define
// the same keys (enforced by a test).
var catalog = map[string]Messages{
	"en": {
		"title":        "Secret Santa",
		"create":       "Create a new Session",
		"open_session": "Open an existing Session",
		"open_link":    "Open a Personal link",
		"session_hash": "Session hash or link",
		"link_hash":    "Personal link hash or link",
		"go":           "Open",

		"me":              "Me",
		"name":            "Name",
		"email":           "Email",
		"add_participant": "Add participant",
		"remove":          "Remove",
		"submit_session":  "Send invitations",
		"status_pending":  "pending",
		"err_too_few":     "Please enter at least 3 participants.",
		"err_empty_name":  "Names must not be empty.",
		"err_dup_name":    "Names must be unique (ignoring case).",
		"err_bad_email":   "Please enter a valid email address for everyone.",

		"invite_subject": "Invitation to Secret Santa",
		"invite_body":    "Hello %s,\n\nyou have been invited to a Secret Santa round. Your Personal link:\n\n%s\n",
	},
	"de": {
		"title":        "Wichteln",
		"create":       "Neue Sitzung erstellen",
		"open_session": "Bestehende Sitzung öffnen",
		"open_link":    "Persönlichen Link öffnen",
		"session_hash": "Sitzungs-Hash oder -Link",
		"link_hash":    "Hash oder Link des persönlichen Links",
		"go":           "Öffnen",

		"me":              "Ich",
		"name":            "Name",
		"email":           "E-Mail",
		"add_participant": "Teilnehmer hinzufügen",
		"remove":          "Entfernen",
		"submit_session":  "Einladungen senden",
		"status_pending":  "ausstehend",
		"err_too_few":     "Bitte mindestens 3 Teilnehmer eintragen.",
		"err_empty_name":  "Namen dürfen nicht leer sein.",
		"err_dup_name":    "Namen müssen eindeutig sein (Groß-/Kleinschreibung egal).",
		"err_bad_email":   "Bitte für alle eine gültige E-Mail-Adresse eintragen.",

		"invite_subject": "Einladung zum Wichteln",
		"invite_body":    "Hallo %s,\n\ndu wurdest zu einer Wichtelrunde eingeladen. Dein persönlicher Link:\n\n%s\n",
	},
}

// messagesFor returns the catalog matching the request's Accept-Language.
func messagesFor(r *http.Request) Messages { return catalog[languageFor(r)] }

// languageFor returns "de" if German is the top-ranked supported language in
// the request's Accept-Language, otherwise "en".
func languageFor(r *http.Request) string {
	type pref struct {
		lang string
		q    float64
	}
	var prefs []pref
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		q := 1.0
		if v, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			q, _ = strconv.ParseFloat(v, 64)
		}
		lang, _, _ := strings.Cut(strings.ToLower(tag), "-")
		if q > 0 {
			prefs = append(prefs, pref{lang, q})
		}
	}
	sort.SliceStable(prefs, func(i, j int) bool { return prefs[i].q > prefs[j].q })
	for _, p := range prefs {
		if _, ok := catalog[p.lang]; ok {
			return p.lang
		}
	}
	return "en"
}
