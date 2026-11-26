# 08: Correct email and resend link

**What to build:** Before the Draw, the Organizer can fix a Participant's mistyped email address on the Session page, which re-sends that Participant's Personal link to the new address, or simply resend a Personal link. This way a typo or lost email doesn't block the Session.

**Blocked by:** 06

**Status:** done

Spec: `.scratch/secret-santa/spec.md` · Vocabulary: `GLOSSARY.md` · Persistence: ADR-0001

- [x] The Session page offers per-Participant actions (before the Draw only): correct the email address and resend the Personal link
- [x] Correcting an email persists it and sends the invitation email to the new address; resend sends it to the current address; both use the Organizer's language
- [x] Names and the set of Participants remain unchangeable
- [x] The actions are neither shown nor accepted (POSTs rejected) once the Session is `Drawn`
- [x] Seam A tests: correction, resend, unchanged names, rejection after the Draw
