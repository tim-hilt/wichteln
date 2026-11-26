package main

import (
	"errors"
	"net/http"
	"os"
)

// errNotReleasable is returned from a store update when the name is unknown,
// not joined, or the Organizer's.
var errNotReleasable = errors.New("name cannot be released")

// releaseParticipant undoes a Join before the Draw: the name returns to the
// Join page and can be joined again; its email, Personal link, Wish,
// Exclusions and Submission are gone. It sends no email and never Draws.
// Unknown Sessions, the Organizer and names not joined get 404; after the
// Draw, 409 and nothing changes.
func (a *app) releaseParticipant(w http.ResponseWriter, r *http.Request) {
	name := r.PostFormValue("participant")
	err := a.store.Update(r.PathValue("hash"), func(s *Session) error {
		p := s.participantNamed(name)
		switch {
		case p == nil || !p.joined() || p.Organizer:
			return errNotReleasable
		case s.Drawn():
			return errAlreadyDrawn
		}
		// Keep the name; everything the person gave is dropped.
		*p = Participant{Name: p.Name}
		return nil
	})
	switch {
	case errors.Is(err, os.ErrNotExist), errors.Is(err, errNotReleasable):
		http.NotFound(w, r)
	case errors.Is(err, errAlreadyDrawn):
		http.Error(w, "already drawn", http.StatusConflict)
	case err != nil:
		internalError(w)
	default:
		http.Redirect(w, r, "/session/"+r.PathValue("hash"), http.StatusSeeOther)
	}
}
