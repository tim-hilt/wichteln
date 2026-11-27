# 01: Add names through the Session page

**What to build:** The Organizer's Session page gets an edit form with the full list of names. Joined names are fixed. The Organizer can add rows (HTMX, like the creation form) and press Save, which appends the new names to the Session. The new names appear on the Join page right away, and every joined Participant gets an email that a name was added, with a link to their Personal link where they can still change their Exclusions.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

Spec: `.scratch/edit-names/spec.md`

- [ ] The Session page shows an edit form with all names; joined names have no remove button and cannot be edited
- [ ] Save posts the complete list of names; new names are appended, the Organizer stays first, existing names keep their order
- [ ] Names are validated with the same rules as creation (trimmed, non-empty, unique case-insensitively); the rules are shared with creation, not copied
- [ ] On a validation error the form is shown again with status 422 and the entered rows kept
- [ ] After the Draw, Save returns 409 and changes nothing
- [ ] An added name is listed on the Join page and can be joined
- [ ] Every joined Participant gets one email about the added name, in their own language, with their Personal link; nobody is un-submitted
- [ ] The new name appears in the joined Participants' Exclusions form
- [ ] An Add never starts the Draw
- [ ] GLOSSARY.md defines **Add** and the Organizer definition no longer says the Organizer named all Participants at creation
- [ ] All new texts in DE and EN
