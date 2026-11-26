package main

import (
	"errors"
	"fmt"
	"net/http"
)

// errNameTaken is returned from a store update that finds the name joined.
var errNameTaken = errors.New("name already joined")

// findByJoinLink returns the Session behind a Join link hash, or writes the
// 404 / 500 response and returns ok=false.
func (a *app) findByJoinLink(w http.ResponseWriter, r *http.Request) (Session, bool) {
	sess, ok, err := a.store.LoadByJoinLink(r.PathValue("hash"))
	switch {
	case err != nil:
		internalError(w)
	case !ok:
		renderNotFound(w, r)
	}
	return sess, ok && err == nil
}

// joinPage lets a visitor pick their name among the not-joined ones; with a
// "name" query for such a name it shows the join form for it instead. A
// Drawn Session shows only that joining is closed.
func (a *app) joinPage(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.findByJoinLink(w, r)
	if !ok {
		return
	}
	if p := sess.participantNamed(r.URL.Query().Get("name")); p != nil && !p.joined() && sess.State != stateDrawn {
		renderJoinForm(w, r, http.StatusOK, sess, Participant{Name: p.Name}, nil)
		return
	}
	renderJoinSelection(w, r, http.StatusOK, sess, "")
}

// submitJoin Joins the name sent in the "name" field with the first
// Submission: email, Wish and Exclusions. It stores them, gives the name its
// Personal link, redirects to its confirmation and emails the joiner a confirmation in
// their language. Two Joins of one name can't both succeed: the later one
// gets 409.
func (a *app) submitJoin(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.findByJoinLink(w, r)
	if !ok {
		return
	}
	if sess.State == stateDrawn {
		renderJoinSelection(w, r, http.StatusConflict, sess, "")
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	t := messagesFor(r)
	name := r.PostForm.Get("name")
	if p := sess.participantNamed(name); p == nil || p.joined() {
		renderJoinSelection(w, r, http.StatusConflict, sess, t["err_name_taken"])
		return
	}
	me := submissionFrom(r, sess, Participant{Name: name})
	if key := submissionError(me); key != "" {
		renderJoinForm(w, r, http.StatusUnprocessableEntity, sess, me, []string{t[key]})
		return
	}

	var err error
	if me.Hash, err = newHash(a.random); err != nil {
		internalError(w)
		return
	}
	drawn, err := a.submit(sess.Hash, func(s *Session) error {
		p := s.participantNamed(me.Name)
		if p == nil || p.joined() {
			return errNameTaken
		}
		*p = me
		return nil
	})
	switch {
	case errors.Is(err, errAlreadyDrawn):
		renderJoinSelection(w, r, http.StatusConflict, sess, "")
	case errors.Is(err, errNameTaken):
		renderJoinSelection(w, r, http.StatusConflict, sess, t["err_name_taken"])
	case errors.Is(err, ErrNoDraw):
		renderJoinForm(w, r, http.StatusUnprocessableEntity, sess, me, []string{t["err_impossible_draw"]})
	case err != nil:
		internalError(w)
	default:
		// The Join is saved: attempt every email even if one fails.
		if err := errors.Join(a.sendJoinConfirmation(me), a.notifyOrganizer(sess, me), a.sendDrawEmails(drawn)); err != nil {
			internalError(w)
			return
		}
		http.Redirect(w, r, "/"+me.Hash, http.StatusSeeOther)
	}
}

// sendJoinConfirmation emails the joiner their Personal link in their language.
func (a *app) sendJoinConfirmation(p Participant) error {
	t := catalog[p.Lang]
	return a.send(p, t["join_subject"], mailView{T: t, Intro: t["join_intro"], Links: []mailLink{{t["personal_link"], a.baseURL + "/" + p.Hash}}})
}

// notifyOrganizer emails the Organizer, in their language, who has joined.
func (a *app) notifyOrganizer(sess Session, joiner Participant) error {
	t := catalog[sess.Lang]
	return a.send(*sess.organizer(), fmt.Sprintf(t["joined_subject"], joiner.Name), mailView{T: t, Intro: fmt.Sprintf(t["joined_intro"], joiner.Name)})
}

func renderJoinSelection(w http.ResponseWriter, r *http.Request, status int, sess Session, errMsg string) {
	var names []string
	for _, p := range sess.Participants {
		if !p.joined() {
			names = append(names, p.Name)
		}
	}
	render(w, status, "join_select.html", map[string]any{
		"T": messagesFor(r), "Hash": sess.JoinHash, "Names": names, "Error": errMsg, "Closed": sess.State == stateDrawn,
	})
}

// renderJoinForm shows the join form greeting me, prefilled with what me
// has entered so far. It posts back to the Join link it is shown on.
func renderJoinForm(w http.ResponseWriter, r *http.Request, status int, sess Session, me Participant, errs []string) {
	render(w, status, "join_form.html", submissionView(r, sess, me, errs))
}
