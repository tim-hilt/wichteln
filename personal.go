package main

import (
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
	Wish   string
	Others []otherView
	Errors []string
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
		for _, p := range sess.Participants {
			if p.Hash == hash {
				return sess, p, true
			}
		}
	}
	render(w, http.StatusNotFound, "not_found.html", map[string]any{"T": messagesFor(r)})
	return Session{}, Participant{}, false
}

// personal shows a Participant their own Wish and Exclusions form.
func (a *app) personal(w http.ResponseWriter, r *http.Request) {
	sess, me, ok := a.findByPersonalLink(w, r)
	if !ok {
		return
	}
	renderPersonal(w, r, http.StatusOK, sess, me, nil)
}

// submitPersonal stores the Participant's Wish and Exclusions, marks them as
// submitted and remembers the request's language. It can be repeated to edit.
func (a *app) submitPersonal(w http.ResponseWriter, r *http.Request) {
	sess, me, ok := a.findByPersonalLink(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	me.Wish = strings.TrimSpace(r.PostForm.Get("wish"))
	me.Exclusions = nil
	for _, o := range others(sess, me) {
		if slices.Contains(r.PostForm["exclude"], o) {
			me.Exclusions = append(me.Exclusions, o)
		}
	}
	t := messagesFor(r)
	switch {
	case me.Wish == "":
		renderPersonal(w, r, http.StatusUnprocessableEntity, sess, me, []string{t["err_empty_wish"]})
		return
	case utf8.RuneCountInString(me.Wish) > maxWishLength:
		renderPersonal(w, r, http.StatusUnprocessableEntity, sess, me, []string{t["err_long_wish"]})
		return
	}
	me.Submitted = true
	me.Lang = languageFor(r)
	if _, err := a.store.SaveParticipant(me); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+me.Hash, http.StatusSeeOther)
}

// others returns the names of all Participants except me.
func others(sess Session, me Participant) []string {
	var names []string
	for _, p := range sess.Participants {
		if p.Hash != me.Hash {
			names = append(names, p.Name)
		}
	}
	return names
}

func renderPersonal(w http.ResponseWriter, r *http.Request, status int, sess Session, me Participant, errs []string) {
	view := personalView{T: messagesFor(r), Name: me.Name, Wish: me.Wish, Errors: errs}
	for _, name := range others(sess, me) {
		view.Others = append(view.Others, otherView{name, slices.Contains(me.Exclusions, name)})
	}
	render(w, status, "personal.html", view)
}
