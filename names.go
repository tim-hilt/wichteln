package main

import (
	"errors"
	"net/http"
	"os"
	"slices"
	"strings"
)

// errNameMissing is returned from a store update when the posted list lacks
// a name of the Session. Removing names is not supported yet.
var errNameMissing = errors.New("name missing from list")

// addNames saves the edit form of the Session page. It posts the complete
// list of names; the ones not in the Session yet are appended after the
// existing ones, which keep their order (the Organizer stays first). Every
// joined Participant is then emailed the added names; nobody is un-submitted,
// and as the new Participants have not submitted, an Add never Draws.
// Invalid names re-render the page with 422 and the rows kept; after the
// Draw or when the list lacks an existing name it is 409; nothing changes.
func (a *app) addNames(w http.ResponseWriter, r *http.Request) {
	sess, ok, err := a.store.Load(r.PathValue("hash"))
	if err != nil {
		internalError(w)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var names []string
	for _, name := range r.PostForm["name"] {
		names = append(names, strings.TrimSpace(name))
	}
	if sess.Drawn() {
		http.Error(w, "already drawn", http.StatusConflict)
		return
	}
	if errKeys := nameErrors(names); len(errKeys) > 0 {
		a.renderSession(w, r, http.StatusUnprocessableEntity, sess, editedRows(sess, names), errKeys)
		return
	}

	// Everything is checked on the stored Session, not on the page the
	// Organizer saw.
	var joined []Participant // who is told, as of before the change
	var added []string
	err = a.store.Update(sess.Hash, func(s *Session) error {
		if s.Drawn() {
			return errAlreadyDrawn
		}
		joined, added = nil, nil
		for _, p := range s.Participants {
			switch {
			case !slices.Contains(names, p.Name):
				return errNameMissing
			case p.joined():
				joined = append(joined, p)
			}
		}
		for _, name := range names {
			if s.participantNamed(name) == nil {
				added = append(added, name)
				s.Participants = append(s.Participants, Participant{Name: name})
			}
		}
		return nil
	})
	switch {
	case errors.Is(err, os.ErrNotExist):
		http.NotFound(w, r)
	case errors.Is(err, errAlreadyDrawn), errors.Is(err, errNameMissing):
		http.Error(w, "names cannot be changed", http.StatusConflict)
	case err != nil:
		internalError(w)
	default:
		// The names are saved: attempt every email even if one fails.
		var errs []error
		if len(added) > 0 {
			for _, p := range joined {
				errs = append(errs, a.notifyNamesAdded(sess, p, added))
			}
		}
		if err := errors.Join(errs...); err != nil {
			internalError(w)
			return
		}
		http.Redirect(w, r, "/session/"+sess.Hash, http.StatusSeeOther)
	}
}

// editedRows returns the rows of the edit form for the posted names: names of
// sess are Fixed, the others editable.
func editedRows(sess Session, names []string) []rowView {
	rows := make([]rowView, len(names))
	for i, name := range names {
		rows[i] = rowView{Me: name == sess.organizer().Name, Fixed: sess.participantNamed(name) != nil, Name: name}
	}
	return rows
}
