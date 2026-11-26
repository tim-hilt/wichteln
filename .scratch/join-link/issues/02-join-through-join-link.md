# 02: Join a name through the Join link (expand)

**What to build:** Every new Session gets a **Join link** (`/join/<hash>`, independent hash). On the creation form, non-Organizer rows may leave the email empty: such a name is not joined. The Join page asks "select your name" and lists only not-joined names; selecting one opens a form greeting the person by that name and asking for email, Wish and Exclusions. Submitting **Joins** the name (first Submission), redirects to the new Personal link and emails a confirmation with that link in the joiner's language. Rows with an email keep today's behaviour (invited at creation). This is the expand step; ticket 03 removes the email field from non-Organizer rows.

**Blocked by:** 01

**Status:** in-review

Spec: `.scratch/join-link/spec.md`

- [x] Creation accepts non-Organizer rows without email; such names get no Personal link and no email
- [x] The Session page shows the Join link and each name's status (not joined / joined), never Wishes or Exclusions
- [x] `GET /join/<hash>` lists only not-joined names and nothing else about the Session
- [x] Selecting a name shows a form greeting that name; an unknown or already joined name shows the selection again
- [x] The join form validates a plausible email, a non-empty Wish (≤2000 chars) and feasible Exclusions, keeping input on error
- [x] A successful Join redirects to the Personal link and sends a confirmation email with the Personal link in the joiner's browser language
- [x] Joining is one atomic store update: a second Join of the same name gets 409 with a localized "this name was just taken" message
- [x] The automatic Draw fires when the last missing Submission is a Join; Draw emails go to the joined addresses
- [x] After the Draw the Join link shows a plain "joining is closed" page; an unknown Join hash shows the not-found page
- [x] Join link lookups survive a handler rebuilt on the same data directory
- [x] All new texts in DE and EN
