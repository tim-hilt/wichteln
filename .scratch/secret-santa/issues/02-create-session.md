# 02: Create a Session and send invitations

**What to build:** The Organizer clicks "create a new Session", lands on a fresh Session link, enters all Participants (an "Ich"/"Me" row for themselves plus others), and submits. Every Participant receives an email with their Personal link. Afterwards the Session link shows the frozen Participant list with each Participant pending.

**Blocked by:** 01

**Status:** in-review

Spec: `.scratch/secret-santa/spec.md` · Vocabulary: `GLOSSARY.md` · Persistence: ADR-0001

- [x] "Create" redirects to `/session/<hash>` with a fresh 26-character lowercase base32 hash (128 bits from `crypto/rand`)
- [x] `GET /session/<unknown hash>` shows an empty creation form with a pre-labelled "Ich"/"Me" row; nothing is stored before submit
- [x] Participant rows (name + email) can be added and removed via HTMX without full page reloads
- [x] Validation: at least 3 Participants, non-empty names, names unique case-insensitively, plausible email addresses; duplicate emails are allowed; on error, the form re-renders with localized errors and the entered rows kept
- [x] Submit persists the Session in state `Open`, with the Organizer's language, as one JSON file per Session (atomic temp file + rename, behind a mutex); each Participant gets their own Personal link hash
- [x] Submitting to a Session hash that already exists is rejected
- [x] The storage module also supports looking up a Session by Personal link hash (index is internal to storage)
- [x] On submit, every Participant (including the Organizer) receives an invitation email containing their Personal link (base URL + `/<hash>`), in the Organizer's language
- [x] `GET /session/<existing hash>` shows each Participant's name, email and status "pending"; Participants can no longer be added, removed or renamed
- [x] Seam A tests cover: create flow, validation errors, invitation emails captured by a fake Mailer, the Session page after submit, and data surviving a handler rebuilt on the same data directory
