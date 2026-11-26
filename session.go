package main

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
)

// minParticipants keeps the Draw secret: with fewer, assignments are guessable.
const minParticipants = 3

// rowView is one Participant row of the creation form. Only the Organizer's
// row (Me) has an Email.
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
		render(w, http.StatusOK, "session.html", map[string]any{
			"T": messagesFor(r), "Session": sess, "JoinURL": a.baseURL + "/join/" + sess.JoinHash,
		})
	default:
		renderForm(w, r, http.StatusOK, hash, []rowView{{Me: true}}, nil)
	}
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

// sendLinks emails a joined Participant their links: the Organizer all three
// (Session, Join and Personal link) in the Session's language, anyone else
// their Personal link in their own language.
func (a *app) sendLinks(s Session, p Participant) error {
	if !p.Organizer {
		return a.sendJoinConfirmation(p)
	}
	t := catalog[s.Lang]
	return a.send(p, t["organizer_subject"], mailView{T: t, Intro: t["organizer_intro"], Links: []mailLink{
		{t["session_link"], a.baseURL + "/session/" + s.Hash},
		{t["join_link"], a.baseURL + "/join/" + s.JoinHash},
		{t["personal_link"], a.baseURL + "/" + p.Hash},
	}})
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
	case sess.State == stateDrawn:
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
		// Only the Organizer gives an email; the others do so when they join.
		badEmail = badEmail || row.Me && !plausibleEmail(row.Email)
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
