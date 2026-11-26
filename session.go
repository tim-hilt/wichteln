package main

import (
	"errors"
	"net/http"
	"net/mail"
	"os"
	"strings"
)

// minParticipants keeps the Draw secret: with fewer, assignments are guessable.
const minParticipants = 3

// rowView is one Participant row of the creation form.
type rowView struct {
	T           Messages
	Me          bool // the Organizer's own row: labelled, not removable
	Name, Email string
}

// createSessionLink starts a new Session by redirecting to a fresh Session
// link. Nothing is stored until the creation form is submitted.
func (a *app) createSessionLink(w http.ResponseWriter, r *http.Request) {
	hash, err := newHash(a.random)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/session/"+hash, http.StatusSeeOther)
}

// session shows the status page of an existing Session, or the empty
// creation form if the hash is unknown.
func (a *app) session(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if !validHash(hash) {
		http.NotFound(w, r)
		return
	}
	sess, ok, err := a.store.Load(hash)
	switch {
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	case ok:
		render(w, http.StatusOK, "session.html", map[string]any{"T": messagesFor(r), "Session": sess})
	default:
		renderForm(w, r, http.StatusOK, hash, []rowView{{Me: true}}, nil)
	}
}

// participantRow serves an empty row for HTMX to append to the creation form.
func (a *app) participantRow(w http.ResponseWriter, r *http.Request) {
	render(w, http.StatusOK, "participant_row.html", rowView{T: messagesFor(r)})
}

// submitSession validates the creation form, stores the Session as Open and
// emails every Participant their Personal link in the Organizer's language.
func (a *app) submitSession(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if !validHash(hash) {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	rows := rowsFromForm(r)
	if errKeys := validate(rows); len(errKeys) > 0 {
		renderForm(w, r, http.StatusUnprocessableEntity, hash, rows, errKeys)
		return
	}

	lang := languageFor(r)
	sess := Session{Hash: hash, State: stateOpen, Lang: lang}
	for i, row := range rows {
		personal, err := newHash(a.random)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		sess.Participants = append(sess.Participants, Participant{
			Name: row.Name, Email: row.Email, Hash: personal, Organizer: i == 0,
		})
	}
	switch err := a.store.Create(sess); {
	case errors.Is(err, ErrSessionExists):
		http.Error(w, "session already exists", http.StatusConflict)
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	for _, p := range sess.Participants {
		if err := a.sendInvite(sess, p); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/session/"+hash, http.StatusSeeOther)
}

// sendInvite emails the Participant their Personal link in the Organizer's language.
func (a *app) sendInvite(s Session, p Participant) error {
	t := catalog[s.Lang]
	return a.send(p, t["invite_subject"], mailView{T: t, Intro: t["invite_intro"], Links: []mailLink{{t["invite_button"], a.baseURL + "/" + p.Hash}}})
}

// correctEmail replaces a Participant's email address and sends their Personal
// link to the new address. Only possible before the Draw.
func (a *app) correctEmail(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.PostFormValue("email"))
	if plausibleEmail(email) {
		a.participantAction(w, r, func(p *Participant) { p.Email = email })
		return
	}
	sess, ok, err := a.store.Load(r.PathValue("hash"))
	switch {
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	case !ok:
		http.NotFound(w, r)
	default:
		t := messagesFor(r)
		render(w, http.StatusUnprocessableEntity, "session.html", map[string]any{"T": t, "Session": sess, "Errors": []string{t["err_bad_email"]}})
	}
}

// resendLink sends a Participant's Personal link to their current address.
// Only possible before the Draw.
func (a *app) resendLink(w http.ResponseWriter, r *http.Request) {
	a.participantAction(w, r, func(*Participant) {})
}

// participantAction is the shared flow of the Organizer's per-Participant
// actions: find the Participant named by the "participant" field, apply change
// to them, save, then send them the invitation. Unknown Sessions and
// Participants get 404; once the Session is Drawn, 409 and nothing happens.
func (a *app) participantAction(w http.ResponseWriter, r *http.Request, change func(*Participant)) {
	hash := r.PathValue("hash")
	name := r.PostFormValue("participant")
	var updated Session
	var invitee Participant
	err := a.store.Update(hash, func(s *Session) error {
		if s.State == stateDrawn {
			return errAlreadyDrawn
		}
		p := s.participantNamed(name)
		if p == nil {
			return os.ErrNotExist
		}
		change(p)
		updated, invitee = *s, *p
		return nil
	})
	switch {
	case errors.Is(err, os.ErrNotExist):
		http.NotFound(w, r)
	case errors.Is(err, errAlreadyDrawn):
		http.Error(w, "already drawn", http.StatusConflict)
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		if err := a.sendInvite(updated, invitee); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/session/"+hash, http.StatusSeeOther)
	}
}

// rowsFromForm reads the parallel name/email fields; the first row is the Organizer.
func rowsFromForm(r *http.Request) []rowView {
	names, emails := r.PostForm["name"], r.PostForm["email"]
	var rows []rowView
	for i, name := range names {
		row := rowView{Me: i == 0, Name: strings.TrimSpace(name)}
		if i < len(emails) {
			row.Email = strings.TrimSpace(emails[i])
		}
		rows = append(rows, row)
	}
	return rows
}

// validate returns the message keys of all rule violations in the rows.
func validate(rows []rowView) []string {
	var errKeys []string
	if len(rows) < minParticipants {
		errKeys = append(errKeys, "err_too_few")
	}
	seen := map[string]bool{}
	emptyName, dupName, badEmail := false, false, false
	for _, row := range rows {
		key := strings.ToLower(row.Name)
		switch {
		case row.Name == "":
			emptyName = true
		case seen[key]:
			dupName = true
		}
		seen[key] = true
		badEmail = badEmail || !plausibleEmail(row.Email)
	}
	if emptyName {
		errKeys = append(errKeys, "err_empty_name")
	}
	if dupName {
		errKeys = append(errKeys, "err_dup_name")
	}
	if badEmail {
		errKeys = append(errKeys, "err_bad_email")
	}
	return errKeys
}

// plausibleEmail accepts a bare address with a dotted domain, e.g. a@b.de.
func plausibleEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return false
	}
	_, domain, _ := strings.Cut(s, "@")
	return strings.Contains(domain, ".")
}

func renderForm(w http.ResponseWriter, r *http.Request, status int, hash string, rows []rowView, errKeys []string) {
	t := messagesFor(r)
	var errs []string
	for _, k := range errKeys {
		errs = append(errs, t[k])
	}
	for i := range rows {
		rows[i].T = t
	}
	render(w, status, "session_form.html", map[string]any{"T": t, "Hash": hash, "Rows": rows, "Errors": errs})
}
