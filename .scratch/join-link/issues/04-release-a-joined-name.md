# 04: Release a joined name

**What to build:** Before the Draw, the Organizer can **Release** any joined Participant except themselves from the Session page. The name returns to the Join page and can be joined again, e.g. after a mis-click or a mistyped email address.

**Blocked by:** 02

**Status:** in-review

Spec: `.scratch/join-link/spec.md`

- [x] The Session page offers "Release" for each joined non-Organizer Participant before the Draw
- [x] Release clears the Participant's email, Wish, Exclusions and submitted state; their old Personal link returns the not-found page
- [x] The released name is listed on the Join page again and can be joined
- [x] Release sends no email and never triggers the Draw
- [x] Releasing the Organizer or a not-joined name is rejected; after the Draw Release returns 409 and changes nothing
- [x] All new texts in DE and EN
