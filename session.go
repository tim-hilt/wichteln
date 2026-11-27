package main

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
)

// minParticipants keeps the Draw secret: with fewer, assignments are guessable.
const minParticipants = 3

// rowView is one Participant row of the creation or edit form. Only the
// Organizer's row (Me) in the creation form has an Email.
type rowView struct {
	T           Messages
	Me          bool // the Organizer's own row: labelled, not removable
	Fixed       bool // an existing name of the edit form: shown, but neither editable nor removable
	Name, Email string
}

// createSessionLink starts a new Session by redirecting to a fresh Session
// link. Nothing is stored until the creation form is submitted.
func (a *app) createSessionLink(w http.ResponseWriter, r *http.Request) {
	hash, err := newHash(a.random)
	if err != nil {
		internalError(w)
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
		internalError(w)
	case ok:
		a.renderSession(w, r, http.StatusOK, sess, existingRows(sess), nil)
	default:
		renderForm(w, r, http.StatusOK, hash, []rowView{{Me: true}}, nil)
	}
}

// renderSession shows the status page of sess with the edit form filled with
// rows and the messages for errKeys above it.
func (a *app) renderSession(w http.ResponseWriter, r *http.Request, status int, sess Session, rows []rowView, errKeys []string) {
	t := messagesFor(r)
	for i := range rows {
		rows[i].T = t
	}
	render(w, status, "session.html", map[string]any{
		"T": t, "Session": sess, "JoinURL": a.baseURL + "/join/" + sess.JoinHash,
		"Rows": rows, "Errors": errorTexts(t, errKeys),
	})
}

// existingRows returns one Fixed row per name of sess.
func existingRows(sess Session) []rowView {
	var rows []rowView
	for i, p := range sess.Participants {
		rows = append(rows, rowView{Me: i == 0, Fixed: true, Name: p.Name})
	}
	return rows
}

// participantRow serves an empty row for HTMX to append to the creation form.
func (a *app) participantRow(w http.ResponseWriter, r *http.Request) {
	render(w, http.StatusOK, "participant_row.html", rowView{T: messagesFor(r)})
}

// submitSession validates the creation form, stores the Session as Open with
// the Organizer already joined, and emails the Organizer, and only them, their
// links in their language. Everyone else joins later via the Join link.
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

	joinHash, joinErr := newHash(a.random)
	organizerHash, organizerErr := newHash(a.random)
	if errors.Join(joinErr, organizerErr) != nil {
		internalError(w)
		return
	}
	sess := Session{Hash: hash, JoinHash: joinHash, State: stateOpen, Lang: languageFor(r)}
	for i, row := range rows {
		sess.Participants = append(sess.Participants, Participant{Name: row.Name, Email: row.Email, Organizer: i == 0})
	}
	sess.organizer().Hash = organizerHash
	switch err := a.store.Create(sess); {
	case errors.Is(err, ErrSessionExists):
		http.Error(w, "session already exists", http.StatusConflict)
		return
	case err != nil:
		internalError(w)
		return
	}

	if err := a.sendLinks(sess, *sess.organizer()); err != nil {
		internalError(w)
		return
	}
	http.Redirect(w, r, "/session/"+hash, http.StatusSeeOther)
}

// resendLink sends a joined Participant their links again, to the address they
// entered. Unknown Sessions and Participants and names not joined yet get 404;
// once the Session is Drawn, 409 and nothing is sent.
func (a *app) resendLink(w http.ResponseWriter, r *http.Request) {
	sess, ok, err := a.store.Load(r.PathValue("hash"))
	if err != nil {
		internalError(w)
		return
	}
	p := sess.participantNamed(r.PostFormValue("participant"))
	switch {
	case !ok || p == nil || !p.joined():
		http.NotFound(w, r)
	case sess.Drawn():
		http.Error(w, "already drawn", http.StatusConflict)
	default:
		if err := a.sendLinks(sess, *p); err != nil {
			internalError(w)
			return
		}
		http.Redirect(w, r, "/session/"+sess.Hash, http.StatusSeeOther)
	}
}

// rowsFromForm reads the name fields and the Organizer's email; the first row
// is the Organizer's.
func rowsFromForm(r *http.Request) []rowView {
	var rows []rowView
	for i, name := range r.PostForm["name"] {
		row := rowView{Me: i == 0, Name: strings.TrimSpace(name)}
		if row.Me {
			row.Email = strings.TrimSpace(r.PostForm.Get("email"))
		}
		rows = append(rows, row)
	}
	return rows
}

// validate returns the message keys of all rule violations in the creation
// rows: the name rules plus the Organizer's email. Only the Organizer gives
// an email; the others do so when they join.
func validate(rows []rowView) []string {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = row.Name
	}
	errKeys := nameErrors(names)
	if rows[0].Me && !plausibleEmail(rows[0].Email) {
		errKeys = append(errKeys, "err_bad_email")
	}
	return errKeys
}

// nameErrors returns the message keys of all rules the (trimmed) names of a
// Session violate: at least minParticipants, none empty, none equal to
// another regardless of case. Creation and the edit form share them.
func nameErrors(names []string) []string {
	var errKeys []string
	if len(names) < minParticipants {
		errKeys = append(errKeys, "err_too_few")
	}
	seen := map[string]bool{}
	emptyName, dupName := false, false
	for _, name := range names {
		key := strings.ToLower(name)
		switch {
		case name == "":
			emptyName = true
		case seen[key]:
			dupName = true
		}
		seen[key] = true
	}
	if emptyName {
		errKeys = append(errKeys, "err_empty_name")
	}
	if dupName {
		errKeys = append(errKeys, "err_dup_name")
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
	for i := range rows {
		rows[i].T = t
	}
	render(w, status, "session_form.html", map[string]any{"T": t, "Hash": hash, "Rows": rows, "Errors": errorTexts(t, errKeys)})
}

// errorTexts returns the texts of the message keys.
func errorTexts(t Messages, keys []string) []string {
	var texts []string
	for _, k := range keys {
		texts = append(texts, t[k])
	}
	return texts
}
