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
	},
	"de": {
		"title":        "Wichteln",
		"create":       "Neue Sitzung erstellen",
		"open_session": "Bestehende Sitzung öffnen",
		"open_link":    "Persönlichen Link öffnen",
		"session_hash": "Sitzungs-Hash oder -Link",
		"link_hash":    "Hash oder Link des persönlichen Links",
		"go":           "Öffnen",
	},
}

// messagesFor returns the catalog matching the request's Accept-Language:
// German if it is the top-ranked supported language, otherwise English.
func messagesFor(r *http.Request) Messages {
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
		if m, ok := catalog[p.lang]; ok {
			return m
		}
	}
	return catalog["en"]
}
