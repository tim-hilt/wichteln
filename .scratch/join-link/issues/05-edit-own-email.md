# 05: Participants edit their own email

**What to build:** Before the Draw, a Participant can change their own email address on their Personal link page, together with Wish and Exclusions. A changed address receives the confirmation email again; for the Organizer it receives the complete Organizer email (Session link, Join link, Personal link).

**Blocked by:** 03

**Status:** done

Spec: `.scratch/join-link/spec.md`

- [x] The Personal link form shows the Participant's current email as a required field, validated as plausible
- [x] Saving with a changed address sends the confirmation email to the new address (the Organizer email for the Organizer)
- [x] Saving without changing the address sends no email
- [x] After the Draw the email can no longer be changed
- [x] All new texts in DE and EN
