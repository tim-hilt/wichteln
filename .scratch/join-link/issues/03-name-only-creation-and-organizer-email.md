# 03: Name-only creation and the Organizer email (contract)

**What to build:** The creation form asks for an email only in the Organizer's "Me" row; all other rows are name-only. The Organizer is joined at creation and receives exactly one email containing their Session link, Join link and Personal link. Nobody else is emailed at creation. The Organizer never sees anyone's email address.

**Blocked by:** 02

**Status:** ready-for-agent

Spec: `.scratch/join-link/spec.md`

- [ ] Creation rows other than "Me" have no email field; validation: ≥3 names, non-empty, unique ignoring case, plausible Organizer email
- [ ] Creation sends exactly one email, to the Organizer, in the Organizer's language, with Session link, Join link and Personal link
- [ ] The Organizer's name is never listed on the Join page
- [ ] The Organizer's row shows "pending" until they submit their Wish, then "joined"
- [ ] "Resend link" sends the Organizer their complete email, and other joined Participants their confirmation email in their own language; not offered for not-joined names
- [ ] The Organizer's "correct email" action and route are removed; the Session page shows no email addresses
- [ ] Shared test helpers create Sessions and obtain Personal links via the join flow
- [ ] All new texts in DE and EN
