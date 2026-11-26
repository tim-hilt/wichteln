# 03: Personal link: Wish and Exclusions

**What to build:** A Participant opens their Personal link and enters a Wish and optional one-way Exclusions (other Participants they must not give to), then submits. They can come back and edit both until the Draw. The Organizer's Session page shows them as submitted, without revealing anything they entered.

**Blocked by:** 02

**Status:** ready-for-agent

Spec: `.scratch/secret-santa/spec.md` · Vocabulary: `GLOSSARY.md` · Persistence: ADR-0001

- [ ] `GET /<hash>` for a known Personal link greets the Participant by name; an unknown hash returns a plain localized 404 page
- [ ] The form has a required free-text Wish (non-empty, max ~2000 characters) and checkboxes for every other Participant to exclude
- [ ] The page explains that Exclusions only restrict whom this Participant gives to
- [ ] Submit stores the Wish and Exclusions, marks the Participant as submitted, and stores the request's language on the Participant
- [ ] Reopening the Personal link shows the saved Wish and Exclusions; editing and resubmitting keeps the Participant submitted
- [ ] A Participant's page never shows Exclusions created by other Participants
- [ ] The Session page shows submitted/pending per Participant and never contains any Wish or Exclusion
- [ ] Seam A tests cover: submit, edit, validation errors, 404 for unknown hash, Session page status, and the privacy rules above
