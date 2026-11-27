# 02: Remove names that have not joined

**What to build:** Save also removes unjoined names that are missing from the submitted list. Adds, removals and renames work together in one Save. Joined names and the Organizer can't be removed. Anyone who opens the join form for a removed name is told that it is not part of the session.

**Blocked by:** 01

**Status:** in-review

Spec: `.scratch/edit-names/spec.md`

- [x] An unjoined name missing from the list is removed; a rename works as remove plus add
- [x] The minimum of 3 Participants is checked on the final list, so one Save can swap a name at exactly 3 Participants; fewer than 3 is rejected with its own error text
- [x] A joined name missing from the list returns 409 with an explanation and nothing is saved, including a name that joined while the Organizer was editing (checked inside the update)
- [x] The Organizer can't be removed
- [x] A removed name is stripped silently from every Participant's Exclusions; no email is sent for this
- [x] Re-adding a removed name starts clean: no old Exclusion applies to it
- [x] Removing the last Participant who has not submitted starts the Draw and sends the Draw emails
- [x] The Join page and join form show "<name> is not part of the session" for a name that does not exist
- [x] Release still works as before and never Draws; Release then Remove works for a joined name
- [x] GLOSSARY.md defines **Remove**, noting that Save applies Add and Remove together
- [x] All new texts in DE and EN
