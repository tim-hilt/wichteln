# 04: One combined email when a Save adds names and resets Exclusions

**What to build:** A Participant who would get both the "name added" email and the "Exclusions reset" email from one Save gets a single email covering both.

**Blocked by:** 01, 03

**Status:** done

Spec: `.scratch/edit-names/spec.md`

- [x] One Save that adds names and resets a Participant's Exclusions sends that Participant exactly one email, mentioning both
- [x] Participants only affected by the Add get the Add email; only affected by a reset get the reset email
- [x] Mail is sent after the Save succeeded, and every send is attempted even if one fails
- [x] Emails contain only the Participant's own information
- [x] Combined text in DE and EN
