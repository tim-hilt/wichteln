package main

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"
)

// maxWishLength is the longest accepted Wish, in characters.
const maxWishLength = 2000

// personalView is the data of the Personal link page. It only ever holds the
// viewing Participant's own Wish and Exclusions.
type personalView struct {
	T      Messages
	Name   string
	Email  string
	Wish   string
	Others []otherView
	Errors []string

	// After the Draw the page is read-only and shows the Assignment.
	Drawn          bool
	Recipient      string
	RecipientsWish string
}

// otherView is one checkbox: another Participant who can be excluded.
type otherView struct {
	Name     string
	Excluded bool
}

// findByPersonalLink returns the Session and Participant behind a Personal
// link hash, or writes the 404 / 500 response and returns ok=false.
func (a *app) findByPersonalLink(w http.ResponseWriter, r *http.Request) (Session, Participant, bool) {
	hash := r.PathValue("hash")
	sess, ok, err := a.store.LoadByPersonalLink(hash)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return Session{}, Participant{}, false
	}
	if ok {
		if p := sess.participant(hash); p != nil {
			return sess, *p, true
		}
	}
	render(w, http.StatusNotFound, "not_found.html", map[string]any{"T": messagesFor(r)})
	return Session{}, Participant{}, false
}

// personal shows a Participant their own email, Wish and Exclusions form.
func (a *app) personal(w http.ResponseWriter, r *http.Request) {
	sess, me, ok := a.findByPersonalLink(w, r)
	if !ok {
		return
	}
	renderPersonal(w, r, http.StatusOK, sess, me, nil)
}

// submitPersonal stores the Participant's email, Wish and Exclusions, marks
// them as submitted and remembers the request's language. It can be repeated
// to edit. A changed email address is sent the links email again (see
// sendLinks), an unchanged one nothing. Exclusions that would leave no valid Draw are rejected and nothing is saved.
func (a *app) submitPersonal(w http.ResponseWriter, r *http.Request) {
	sess, me, ok := a.findByPersonalLink(w, r)
	if !ok {
		return
	}
	if sess.State == stateDrawn {
		http.Error(w, "already drawn", http.StatusConflict)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	emailChanged := false
	me.Email = strings.TrimSpace(r.PostForm.Get("email"))
	me.Wish = strings.TrimSpace(r.PostForm.Get("wish"))
	me.Lang = languageFor(r)
	me.Exclusions = nil
	for _, o := range others(sess, me) {
		if slices.Contains(r.PostForm["exclude"], o) {
			me.Exclusions = append(me.Exclusions, o)
		}
	}
	t := messagesFor(r)
	if !plausibleEmail(me.Email) {
		renderPersonal(w, r, http.StatusUnprocessableEntity, sess, me, []string{t["err_bad_email"]})
		return
	}
	if key := wishError(me.Wish); key != "" {
		renderPersonal(w, r, http.StatusUnprocessableEntity, sess, me, []string{t[key]})
		return
	}

	drawn, err := a.submit(sess.Hash, func(s *Session) error {
		p := s.participant(me.Hash)
		emailChanged = p.Email != me.Email
		p.Email, p.Wish, p.Exclusions, p.Submitted, p.Lang = me.Email, me.Wish, me.Exclusions, true, me.Lang
		return nil
	})
	switch {
	case errors.Is(err, errAlreadyDrawn):
		http.Error(w, "already drawn", http.StatusConflict)
	case errors.Is(err, ErrNoDraw):
		renderPersonal(w, r, http.StatusUnprocessableEntity, sess, me, []string{t["err_impossible_draw"]})
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		if emailChanged {
			if err := a.sendLinks(sess, me); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
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

// wishError returns the message key of the rule a Wish violates, or "".
func wishError(wish string) string {
	switch {
	case wish == "":
		return "err_empty_wish"
	case utf8.RuneCountInString(wish) > maxWishLength:
		return "err_long_wish"
	}
	return ""
}

// submit stores a Submission, which change applies to the Session, and runs
// the automatic Draw if it was the last missing one; the Session it drew is
// returned, otherwise nil. Everything happens in one store update, so two
// simultaneous Submissions can't each pass the feasibility check on their own
// and together leave no valid Draw, and the last ones can't Draw twice.
// change's error aborts the update; a Drawn Session fails with errAlreadyDrawn
// and exclusions leaving no valid Draw with ErrNoDraw.
func (a *app) submit(sessionHash string, change func(*Session) error) (drawn *Session, err error) {
	err = a.store.Update(sessionHash, func(s *Session) error {
		if s.State == stateDrawn {
			return errAlreadyDrawn
		}
		if err := change(s); err != nil {
			return err
		}
		if !DrawPossible(s.Participants) {
			return ErrNoDraw
		}
		if s.allSubmitted() {
			if err := drawSession(s, a.random); err != nil {
				return err
			}
			drawn = s
		}
		return nil
	})
	return drawn, err
}

// errAlreadyDrawn is returned from a store update that finds the Draw done.
var errAlreadyDrawn = errors.New("session already drawn")

// sendDrawEmails tells every Participant, in the language of their last
// Submission, whom they give to and that person's Wish. A failed email does
// not stop the others: the Draw is already saved and can't be repeated.
func (a *app) sendDrawEmails(s Session) error {
	var errs []error
	for _, p := range s.Participants {
		t := catalog[p.Lang]
		view := mailView{T: t, Intro: fmt.Sprintf(t["draw_intro"], p.Recipient), Wish: s.participantNamed(p.Recipient).Wish}
		errs = append(errs, a.send(p, t["draw_subject"], view))
	}
	return errors.Join(errs...)
}

// others returns the names of all Participants except me.
func others(sess Session, me Participant) []string {
	var names []string
	for _, p := range sess.Participants {
		if p.Name != me.Name {
			names = append(names, p.Name)
		}
	}
	return names
}

// otherViews returns the Exclusion checkboxes for me, ticked as me has set them.
func otherViews(sess Session, me Participant) []otherView {
	var views []otherView
	for _, name := range others(sess, me) {
		views = append(views, otherView{name, slices.Contains(me.Exclusions, name)})
	}
	return views
}

func renderPersonal(w http.ResponseWriter, r *http.Request, status int, sess Session, me Participant, errs []string) {
	view := personalView{T: messagesFor(r), Name: me.Name, Email: me.Email, Wish: me.Wish, Errors: errs}
	if sess.State == stateDrawn {
		view.Drawn, view.Recipient = true, me.Recipient
		view.RecipientsWish = sess.participantNamed(me.Recipient).Wish
	}
	view.Others = otherViews(sess, me)
	render(w, status, "personal.html", view)
}
