package main

import (
	"errors"
	"net/http"
	"os"
	"slices"
	"strings"
)

// errJoinedNameMissing is returned from a store update when the posted list
// lacks a joined name of the Session: joined names, the Organizer included,
// can't be removed.
var errJoinedNameMissing = errors.New("joined name missing from list")

// needsConfirmation is returned from a store update when a Save would leave no
// valid Draw unless the affected Participants lose Exclusions and the posted
// affected names don't match: the Organizer must confirm exactly these.
type needsConfirmation struct{ affected []string }

func (e *needsConfirmation) Error() string { return "removal needs confirmation" }

// saveNames saves the edit form of the Session page. It posts the complete
// list of names, which Add and Remove apply together: names not in the Session
// yet are appended after the existing ones, which keep their order (the
// Organizer stays first); unjoined names missing from the list are removed.
// Every joined Participant is emailed the added names; nobody is un-submitted
// by an Add. An Add never Draws, a Remove of the last Participant who has not
// submitted does.
// A Remove that leaves no valid Draw saves nothing and shows the names of the
// affected Participants (never their Exclusions) to confirm. The confirmation
// posts the names again with the "affected" names as hidden fields; only if
// they are still exactly the affected ones, the fewest possible Exclusions are
// dropped, those Participants un-submitted and emailed the removed names. A
// Participant affected by both an Add and such a reset gets one combined email.
// Invalid names re-render the page with 422 and the rows kept; after the Draw,
// or when the list lacks a joined name, it is 409; nothing changes.
func (a *app) saveNames(w http.ResponseWriter, r *http.Request) {
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
		a.renderSession(w, r, http.StatusUnprocessableEntity, sess, names, errKeys)
		return
	}

	// Everything is checked on the stored Session, not on the page the
	// Organizer saw. submit strips the removed names from all Exclusions.
	var joined []Participant
	var added, removed, reset []string
	drawn, err := a.submit(sess.Hash, func(s *Session) error {
		kept := s.Participants[:0]
		for _, p := range s.Participants {
			switch {
			case slices.Contains(names, p.Name):
				kept = append(kept, p)
				if p.joined() {
					joined = append(joined, p)
				}
			case p.joined():
				return errJoinedNameMissing
			default:
				removed = append(removed, p.Name)
			}
		}
		s.Participants = kept
		for _, name := range names {
			if s.participantNamed(name) == nil {
				added = append(added, name)
				s.Participants = append(s.Participants, Participant{Name: name})
			}
		}
		if DrawPossible(s.Participants) {
			return nil // also when a confirmed reset is no longer needed
		}
		reset = dropExclusions(s.Participants)
		if !slices.Equal(reset, r.PostForm["affected"]) {
			return &needsConfirmation{reset}
		}
		for _, name := range reset {
			s.participantNamed(name).Submitted = false
		}
		return nil
	})
	var confirm *needsConfirmation
	switch {
	case errors.As(err, &confirm):
		render(w, http.StatusOK, "confirm_removal.html", map[string]any{
			"T": messagesFor(r), "Hash": sess.Hash, "Names": names, "Affected": confirm.affected,
		})
	case errors.Is(err, os.ErrNotExist):
		http.NotFound(w, r)
	case errors.Is(err, errJoinedNameMissing):
		http.Error(w, messagesFor(r)["err_name_joined"], http.StatusConflict)
	case errors.Is(err, errAlreadyDrawn), errors.Is(err, ErrNoDraw):
		http.Error(w, "names cannot be changed", http.StatusConflict)
	case err != nil:
		internalError(w)
	default:
		// The names are saved: attempt every email even if one fails.
		var errs []error
		query := "?saved"
		for _, p := range joined {
			var gone []string // removed names, only for those who lost Exclusions
			if slices.Contains(reset, p.Name) {
				gone = removed
			}
			if len(added) > 0 || gone != nil {
				errs = append(errs, a.notifyNamesChanged(sess, p, added, gone))
				query = "?saved=notified"
			}
		}
		errs = append(errs, a.sendDrawEmails(drawn))
		if err := errors.Join(errs...); err != nil {
			internalError(w)
			return
		}
		http.Redirect(w, r, "/session/"+sess.Hash+query, http.StatusSeeOther)
	}
}
