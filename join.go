package main

import (
	"errors"
	"net/http"
	"slices"
	"strings"
)

// errNameTaken is returned from a store update that finds the name joined.
var errNameTaken = errors.New("name already joined")

// joinView is the data of the join form: the Personal link form plus the
// Join link it posts to.
type joinView struct {
	personalView
	JoinHash string
}

// findByJoinLink returns the Session behind a Join link hash, or writes the
// 404 / 500 response and returns ok=false.
func (a *app) findByJoinLink(w http.ResponseWriter, r *http.Request) (Session, bool) {
	sess, ok, err := a.store.LoadByJoinLink(r.PathValue("hash"))
	switch {
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	case !ok:
		render(w, http.StatusNotFound, "not_found.html", map[string]any{"T": messagesFor(r)})
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
		renderJoinForm(w, r, http.StatusOK, sess, joinView{personalView: personalView{Name: p.Name}}, nil)
		return
	}
	renderJoinSelection(w, r, http.StatusOK, sess, "")
}

// submitJoin Joins the name sent in the "name" field with the first
// Submission: email, Wish and Exclusions. It stores them, gives the name its
// Personal link, redirects to it and emails the joiner a confirmation in
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
	me := Participant{Name: r.PostForm.Get("name"), Email: strings.TrimSpace(r.PostForm.Get("email")), Wish: strings.TrimSpace(r.PostForm.Get("wish"))}
	if p := sess.participantNamed(me.Name); p == nil || p.joined() {
		renderJoinSelection(w, r, http.StatusConflict, sess, t["err_name_taken"])
		return
	}
	for _, o := range others(sess, me) {
		if slices.Contains(r.PostForm["exclude"], o) {
			me.Exclusions = append(me.Exclusions, o)
		}
	}
	view := joinView{personalView: personalView{Name: me.Name, Email: me.Email, Wish: me.Wish, Others: otherViews(sess, me)}}
	switch {
	case !plausibleEmail(me.Email):
		renderJoinForm(w, r, http.StatusUnprocessableEntity, sess, view, []string{t["err_bad_email"]})
		return
	case wishError(me.Wish) != "":
		renderJoinForm(w, r, http.StatusUnprocessableEntity, sess, view, []string{t[wishError(me.Wish)]})
		return
	}

	var err error
	if me.Hash, err = newHash(a.random); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	drawn, err := a.submit(sess.Hash, func(s *Session) error {
		p := s.participantNamed(me.Name)
		if p == nil || p.joined() {
			return errNameTaken
		}
		me.Organizer, me.Submitted, me.Lang = false, true, languageFor(r)
		*p = me
		return nil
	})
	switch {
	case errors.Is(err, errAlreadyDrawn):
		renderJoinSelection(w, r, http.StatusConflict, sess, "")
	case errors.Is(err, errNameTaken):
		renderJoinSelection(w, r, http.StatusConflict, sess, t["err_name_taken"])
	case errors.Is(err, ErrNoDraw):
		renderJoinForm(w, r, http.StatusUnprocessableEntity, sess, view, []string{t["err_impossible_draw"]})
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		if err := a.sendJoinConfirmation(me); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if drawn != nil {
			if err := a.sendDrawEmails(*drawn); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
		http.Redirect(w, r, "/"+me.Hash, http.StatusSeeOther)
	}
}

// sendJoinConfirmation emails the joiner their Personal link in their language.
func (a *app) sendJoinConfirmation(p Participant) error {
	t := catalog[p.Lang]
	return a.send(p, t["join_subject"], mailView{T: t, Intro: t["join_intro"], Links: []mailLink{{t["personal_link"], a.baseURL + "/" + p.Hash}}})
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

// renderJoinForm shows the form for view.Name, listing everyone else as
// possible Exclusions unless view already does.
func renderJoinForm(w http.ResponseWriter, r *http.Request, status int, sess Session, view joinView, errs []string) {
	view.T, view.Errors, view.JoinHash = messagesFor(r), errs, sess.JoinHash
	if view.Others == nil {
		view.Others = otherViews(sess, Participant{Name: view.Name})
	}
	render(w, status, "join_form.html", view)
}
