# 02: Join a name through the Join link (expand)

**What to build:** Every new Session gets a **Join link** (`/join/<hash>`, independent hash). On the creation form, non-Organizer rows may leave the email empty: such a name is not joined. The Join page asks "select your name" and lists only not-joined names; selecting one opens a form greeting the person by that name and asking for email, Wish and Exclusions. Submitting **Joins** the name (first Submission), redirects to the new Personal link and emails a confirmation with that link in the joiner's language. Rows with an email keep today's behaviour (invited at creation). This is the expand step; ticket 03 removes the email field from non-Organizer rows.

**Blocked by:** 01

**Status:** ready-for-agent

Spec: `.scratch/join-link/spec.md`

- [ ] Creation accepts non-Organizer rows without email; such names get no Personal link and no email
- [ ] The Session page shows the Join link and each name's status (not joined / joined), never Wishes or Exclusions
- [ ] `GET /join/<hash>` lists only not-joined names and nothing else about the Session
- [ ] Selecting a name shows a form greeting that name; an unknown or already joined name shows the selection again
- [ ] The join form validates a plausible email, a non-empty Wish (≤2000 chars) and feasible Exclusions, keeping input on error
- [ ] A successful Join redirects to the Personal link and sends a confirmation email with the Personal link in the joiner's browser language
- [ ] Joining is one atomic store update: a second Join of the same name gets 409 with a localized "this name was just taken" message
- [ ] The automatic Draw fires when the last missing Submission is a Join; Draw emails go to the joined addresses
- [ ] After the Draw the Join link shows a plain "joining is closed" page; an unknown Join hash shows the not-found page
- [ ] Join link lookups survive a handler rebuilt on the same data directory
- [ ] All new texts in DE and EN
