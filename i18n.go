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
		"tagline":      "Draw names, share wishes, keep the secret. No sign-up needed.",
		"create":       "Create a new Session",
		"open_session": "Open an existing Session",
		"open_link":    "Open a Personal link",
		"session_hash": "Session hash or link",
		"link_hash":    "Personal link hash or link",
		"go":           "Open",

		"me":                  "Me",
		"participants":        "Participants",
		"name":                "Name",
		"email":               "Email",
		"add_participant":     "Add participant",
		"remove":              "Remove",
		"submit_session":      "Send invitations",
		"status_pending":      "pending",
		"status_joined":       "joined",
		"join_link":           "Join link",
		"status_not_joined":   "not joined",
		"join_select":         "Select your name",
		"join_closed":         "Joining is closed.",
		"submit_join":         "Join",
		"err_name_taken":      "This name was just taken. Please select another one.",
		"join_subject":        "You joined Secret Santa",
		"join_intro":          "you have joined the Secret Santa round. Your Personal link:",
		"save_email":          "Correct email",
		"resend":              "Resend link",
		"not_found":           "Not found",
		"greeting":            "Hello %s!",
		"wish":                "Your wish",
		"exclusions":          "Whom must you not give to?",
		"exclusions_help":     "Exclusions only restrict whom you give to. They do not affect who gives to you.",
		"submit_wish":         "Save",
		"err_empty_wish":      "Please enter a wish.",
		"err_long_wish":       "The wish must be at most 2000 characters.",
		"err_impossible_draw": "With these exclusions and everyone else's, no valid Draw would be possible. Please remove some exclusions.",
		"err_too_few":         "Please enter at least 3 participants.",
		"err_empty_name":      "Names must not be empty.",
		"err_dup_name":        "Names must be unique (ignoring case).",
		"err_bad_email":       "Please enter a valid email address.",

		"drawn":         "Drawn ✓",
		"draw_gives_to": "You give a present to %s.",
		"draw_wish":     "Their wish",

		"invite_subject": "Invitation to Secret Santa",
		"invite_intro":   "you have been invited to a Secret Santa round. Your Personal link:",
		"invite_button":  "Open your Personal link",
		"draw_subject":   "Your Secret Santa assignment",
		"draw_intro":     "the Draw is done! You give a present to %s.",
		"mail_greeting":  "Hello %s,",
	},
	"de": {
		"title":        "Wichteln",
		"tagline":      "Wichtel auslosen, Wünsche teilen, Geheimnis wahren. Ganz ohne Anmeldung.",
		"create":       "Neue Sitzung erstellen",
		"open_session": "Bestehende Sitzung öffnen",
		"open_link":    "Persönlichen Link öffnen",
		"session_hash": "Sitzungs-Hash oder -Link",
		"link_hash":    "Hash oder Link des persönlichen Links",
		"go":           "Öffnen",

		"me":                  "Ich",
		"participants":        "Teilnehmer",
		"name":                "Name",
		"email":               "E-Mail",
		"add_participant":     "Teilnehmer hinzufügen",
		"remove":              "Entfernen",
		"submit_session":      "Einladungen senden",
		"status_pending":      "ausstehend",
		"status_joined":       "beigetreten",
		"join_link":           "Beitrittslink",
		"status_not_joined":   "nicht beigetreten",
		"join_select":         "Wähle deinen Namen",
		"join_closed":         "Der Beitritt ist geschlossen.",
		"submit_join":         "Beitreten",
		"err_name_taken":      "Dieser Name wurde gerade vergeben. Bitte wähle einen anderen.",
		"join_subject":        "Du bist dem Wichteln beigetreten",
		"join_intro":          "du bist der Wichtelrunde beigetreten. Dein persönlicher Link:",
		"save_email":          "E-Mail korrigieren",
		"resend":              "Link erneut senden",
		"not_found":           "Nicht gefunden",
		"greeting":            "Hallo %s!",
		"wish":                "Dein Wunsch",
		"exclusions":          "Wen darfst du nicht beschenken?",
		"exclusions_help":     "Ausschlüsse beschränken nur, wen du beschenkst. Wer dich beschenkt, bleibt davon unberührt.",
		"submit_wish":         "Speichern",
		"err_empty_wish":      "Bitte einen Wunsch eintragen.",
		"err_long_wish":       "Der Wunsch darf höchstens 2000 Zeichen lang sein.",
		"err_impossible_draw": "Mit diesen und den Ausschlüssen der anderen wäre keine gültige Auslosung möglich. Bitte entferne einige Ausschlüsse.",
		"err_too_few":         "Bitte mindestens 3 Teilnehmer eintragen.",
		"err_empty_name":      "Namen dürfen nicht leer sein.",
		"err_dup_name":        "Namen müssen eindeutig sein (Groß-/Kleinschreibung egal).",
		"err_bad_email":       "Bitte eine gültige E-Mail-Adresse eintragen.",

		"drawn":         "Ausgelost ✓",
		"draw_gives_to": "Du beschenkst %s.",
		"draw_wish":     "Der Wunsch",

		"invite_subject": "Einladung zum Wichteln",
		"invite_intro":   "du wurdest zu einer Wichtelrunde eingeladen. Dein persönlicher Link:",
		"invite_button":  "Deinen persönlichen Link öffnen",
		"draw_subject":   "Deine Wichtel-Zuteilung",
		"draw_intro":     "die Auslosung ist erfolgt! Du beschenkst %s.",
		"mail_greeting":  "Hallo %s,",
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
