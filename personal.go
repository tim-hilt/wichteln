package main

import (
	"errors"
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

	// Organizer swaps the Join confirmation's text, as they didn't use the
	// Join link and got no confirmation email.
	Organizer bool

	// Joined shows the confirmation of a Join instead of the form.
	Joined bool

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
	switch {
	case err != nil:
		internalError(w)
	case !ok:
		renderNotFound(w, r)
	default:
		return sess, *sess.participant(hash), true
	}
	return Session{}, Participant{}, false
}

// personal shows a Participant their own email, Wish and Exclusions form.
func (a *app) personal(w http.ResponseWriter, r *http.Request) {
	sess, me, ok := a.findByPersonalLink(w, r)
	if !ok {
		return
	}
	// The first visit after the first Submission (made when joining, or by the
	// Organizer on the form) shows the Join confirmation, later ones the form.
	showJoined := me.Submitted && !me.SawJoined
	if showJoined {
		err := a.store.Update(sess.Hash, func(s *Session) error {
			s.participant(me.Hash).SawJoined = true
			return nil
		})
		if err != nil {
			internalError(w)
			return
		}
	}
	renderPersonal(w, r, http.StatusOK, sess, me, nil, showJoined)
}

// submitPersonal stores the Participant's email, Wish and Exclusions, marks
// them as submitted and remembers the request's language. It can be repeated
// to edit. A changed email address is sent the links email again (see
// sendLinks), an unchanged one nothing. Exclusions that would leave no valid
// Draw are rejected and nothing is saved.
func (a *app) submitPersonal(w http.ResponseWriter, r *http.Request) {
	sess, me, ok := a.findByPersonalLink(w, r)
	if !ok {
		return
	}
	if sess.Drawn() {
		http.Error(w, "already drawn", http.StatusConflict)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	t := messagesFor(r)
	me = submissionFrom(r, sess, me)
	if key := submissionError(me); key != "" {
		renderPersonal(w, r, http.StatusUnprocessableEntity, sess, me, []string{t[key]}, false)
		return
	}

	emailChanged := false
	drawn, err := a.submit(sess.Hash, func(s *Session) error {
		p := s.participant(me.Hash)
		if p == nil {
			return errReleased
		}
		emailChanged = p.Email != me.Email
		*p = me
		return nil
	})
	switch {
	case errors.Is(err, errAlreadyDrawn):
		http.Error(w, "already drawn", http.StatusConflict)
	case errors.Is(err, errReleased):
		renderNotFound(w, r)
	case errors.Is(err, ErrNoDraw):
		renderPersonal(w, r, http.StatusUnprocessableEntity, sess, me, []string{t["err_impossible_draw"]}, false)
	case err != nil:
		internalError(w)
	default:
		// The Submission is saved: attempt every email even if one fails.
		var linksErr error
		if emailChanged {
			linksErr = a.sendLinks(sess, me)
		}
		if err := errors.Join(linksErr, a.sendDrawEmails(drawn)); err != nil {
			internalError(w)
			return
		}
		http.Redirect(w, r, "/"+me.Hash, http.StatusSeeOther)
	}
}

// errReleased is returned from a store update that finds the Participant
// Released since their Personal link was loaded.
var errReleased = errors.New("participant released")

// submissionFrom returns me with the email, Wish and Exclusions of the
// submission form (submission_fields.html), marked as submitted in the
// request's language. Exclusions of unknown names and of me are ignored.
func submissionFrom(r *http.Request, sess Session, me Participant) Participant {
	me.Email = strings.TrimSpace(r.PostForm.Get("email"))
	me.Wish = strings.TrimSpace(r.PostForm.Get("wish"))
	me.Submitted, me.Lang = true, languageFor(r)
	me.Exclusions = nil
	for _, o := range others(sess, me) {
		if slices.Contains(r.PostForm["exclude"], o) {
			me.Exclusions = append(me.Exclusions, o)
		}
	}
	return me
}

// submissionError returns the message key of the first rule the Submission
// violates, or "". Feasibility of the Exclusions is checked by submit.
func submissionError(p Participant) string {
	switch {
	case !plausibleEmail(p.Email):
		return "err_bad_email"
	case p.Wish == "":
		return "err_empty_wish"
	case utf8.RuneCountInString(p.Wish) > maxWishLength:
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
		if s.Drawn() {
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

// submissionView is the submission form (submission_fields.html) for me,
// prefilled with what me has entered so far.
func submissionView(r *http.Request, sess Session, me Participant, errs []string) personalView {
	return personalView{T: messagesFor(r), Name: me.Name, Email: me.Email, Wish: me.Wish, Organizer: me.Organizer, Others: otherViews(sess, me), Errors: errs}
}

func renderPersonal(w http.ResponseWriter, r *http.Request, status int, sess Session, me Participant, errs []string, joined bool) {
	view := submissionView(r, sess, me, errs)
	view.Joined = joined && !sess.Drawn()
	if sess.Drawn() {
		view.Drawn, view.Recipient = true, me.Recipient
		view.RecipientsWish = sess.participantNamed(me.Recipient).Wish
	}
	render(w, status, "personal.html", view)
}
