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
		"create":       "Create a new session",
		"open_session": "Open an existing session",
		"open_link":    "Open your personal link",
		"session_hash": "Paste your session link",
		"link_hash":    "Paste your personal link",
		"open_join":    "Join a session",
		"join_hash":    "Paste the join link",
		"go":           "Open",

		"me":                  "Me",
		"participants":        "Participants",
		"name":                "Name",
		"email":               "Email",
		"add_participant":     "Add participant",
		"remove":              "Remove",
		"submit_session":      "Create session",
		"status_pending":      "pending",
		"status_joined":       "joined",
		"join_link":           "Join link",
		"status_not_joined":   "not joined",
		"join_select":         "Select your name",
		"join_closed":         "The names have already been drawn. You can no longer join.",
		"submit_join":         "Join",
		"err_name_taken":      "Someone else just picked this name. Please select another one.",
		"joined_subject":      "%s joined your Secret Santa session",
		"joined_intro":        "%s has joined your Secret Santa session.",
		"join_subject":        "You joined Secret Santa",
		"join_intro":          "you have joined the Secret Santa session. As soon as everyone has joined, the names are drawn and you get an email telling you whom you give a present to.\n\nUntil then you can change your email address, your wish and whom you must not give to through your personal link:",
		"joined_title":        "You're in, %s!",
		"joined_text":         "You joined the Secret Santa session.",
		"saved_text":          "You added your wish.",
		"joined_email_sent":   "We sent a confirmation email to %s.",
		"save_names":          "Save",
		"names_added_subject": "New names in your Secret Santa session",
		"names_added_intro":   "the Organizer added to the Secret Santa session: %s.\n\nYour submission stays valid. Until the names are drawn you can still change whom you must not give to through your personal link:",
		"resend":              "Resend link",
		"release":             "Release",
		"not_found":           "This link is invalid.",
		"greeting":            "Hello %s!",
		"wish":                "Your wish",
		"exclusions":          "Who must you not give a present to?",
		"submit_wish":         "Save",
		"err_empty_wish":      "Please enter a wish.",
		"err_long_wish":       "The wish must be at most 2000 characters.",
		"err_impossible_draw": "With these exclusions and everyone else's, no draw is possible. Please remove some exclusions.",
		"err_too_few":         "Please enter at least 3 participants.",
		"err_empty_name":      "Names must not be empty.",
		"err_dup_name":        "Each name must be different, regardless of upper or lower case.",
		"err_bad_email":       "Please enter a valid email address.",

		"drawn":         "The names have been drawn.",
		"draw_gives_to": "You give a present to %s.",
		"draw_wish":     "Their wish",

		"organizer_subject": "Your Secret Santa session",
		"organizer_intro":   "your Secret Santa session is ready.\n\n1. Share the join link with the group. Everyone picks their name, enters their email address and wish, and can say whom they must not give a present to.\n\n2. Open your personal link and enter your own wish.\n\n3. Once everyone has joined, the names are drawn automatically and everyone gets an email telling them whom they give a present to.\n\nThe session link shows who has joined so far. Keep this email, it contains all your links.",
		"session_link":      "Session link",
		"personal_link":     "Personal link",
		"draw_subject":      "Your Secret Santa assignment",
		"draw_intro":        "the names have been drawn! You give a present to %s. Keep it to yourself.",
		"mail_greeting":     "Hello %s,",
	},
	"de": {
		"title":        "Wichteln",
		"create":       "Neue Wichtelrunde erstellen",
		"open_session": "Bestehende Wichtelrunde öffnen",
		"open_link":    "Persönlichen Link öffnen",
		"open_join":    "Einer Wichtelrunde beitreten",
		"join_hash":    "Beitrittslink einfügen",
		"session_hash": "Link zur Wichtelrunde einfügen",
		"link_hash":    "Persönlichen Link einfügen",
		"go":           "Öffnen",

		"me":                  "Ich",
		"participants":        "Teilnehmer",
		"name":                "Name",
		"email":               "E-Mail",
		"add_participant":     "Teilnehmer hinzufügen",
		"remove":              "Entfernen",
		"submit_session":      "Wichtelrunde erstellen",
		"status_pending":      "ausstehend",
		"status_joined":       "beigetreten",
		"join_link":           "Beitrittslink",
		"status_not_joined":   "nicht beigetreten",
		"join_select":         "Wähle deinen Namen",
		"join_closed":         "Es wurde bereits ausgelost. Du kannst nicht mehr beitreten.",
		"submit_join":         "Beitreten",
		"err_name_taken":      "Dieser Name wurde gerade vergeben. Bitte wähle einen anderen.",
		"joined_subject":      "%s ist deiner Wichtelrunde beigetreten",
		"joined_intro":        "%s ist deiner Wichtelrunde beigetreten.",
		"join_subject":        "Du bist dem Wichteln beigetreten",
		"join_intro":          "du bist der Wichtelrunde beigetreten. Sobald alle dabei sind, wird ausgelost, und du bekommst eine E-Mail, wen du beschenkst.\n\nBis dahin kannst du über deinen persönlichen Link deine E-Mail-Adresse, deinen Wunsch und deine Ausschlüsse ändern:",
		"joined_title":        "Du bist dabei, %s!",
		"joined_text":         "Du bist der Wichtelrunde beigetreten.",
		"saved_text":          "Du hast deinen Wunsch eingetragen.",
		"joined_email_sent":   "Wir haben eine Bestätigungs-E-Mail an %s geschickt.",
		"save_names":          "Speichern",
		"names_added_subject": "Neue Namen in deiner Wichtelrunde",
		"names_added_intro":   "der Organizer hat der Wichtelrunde hinzugefügt: %s.\n\nDeine Bestätigung bleibt gültig. Bis zur Auslosung kannst du über deinen persönlichen Link noch ändern, wen du nicht beschenken darfst:",
		"resend":              "Link erneut senden",
		"release":             "Freigeben",
		"not_found":           "Dieser Link ist ungültig.",
		"greeting":            "Hallo %s!",
		"wish":                "Dein Wunsch",
		"exclusions":          "Wen darfst du nicht beschenken?",
		"submit_wish":         "Speichern",
		"err_empty_wish":      "Bitte einen Wunsch eintragen.",
		"err_long_wish":       "Der Wunsch darf höchstens 2000 Zeichen lang sein.",
		"err_impossible_draw": "Mit diesen Ausschlüssen und denen der anderen ist keine Auslosung möglich. Bitte entferne einige Ausschlüsse.",
		"err_too_few":         "Bitte mindestens 3 Teilnehmer eintragen.",
		"err_empty_name":      "Namen dürfen nicht leer sein.",
		"err_dup_name":        "Jeder Name darf nur einmal vorkommen, auch bei anderer Groß-/Kleinschreibung.",
		"err_bad_email":       "Bitte eine gültige E-Mail-Adresse eintragen.",

		"drawn":         "Die Namen wurden ausgelost.",
		"draw_gives_to": "Du beschenkst %s.",
		"draw_wish":     "Wunsch",

		"organizer_subject": "Deine Wichtelrunde",
		"organizer_intro":   "deine Wichtelrunde ist erstellt.\n\n1. Teile den Beitrittslink mit der Gruppe. Dort wählt jede Person ihren Namen, trägt E-Mail-Adresse und Wunsch ein und kann angeben, wen sie nicht beschenken darf.\n\n2. Öffne deinen persönlichen Link und trag deinen eigenen Wunsch ein.\n\n3. Sobald alle beigetreten sind, wird automatisch ausgelost. Alle bekommen dann eine E-Mail, wen sie beschenken.\n\nÜber den Link zur Wichtelrunde siehst du, wer schon beigetreten ist. Bewahre diese E-Mail auf, sie enthält alle deine Links.",
		"session_link":      "Link zur Wichtelrunde",
		"personal_link":     "Persönlicher Link",
		"draw_subject":      "Deine Wichtel-Zuteilung",
		"draw_intro":        "die Namen wurden ausgelost! Du beschenkst %s. Verrate es niemandem.",
		"mail_greeting":     "Hallo %s,",
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
