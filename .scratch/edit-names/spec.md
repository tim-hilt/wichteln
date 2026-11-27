# Spec: Edit names

Status: ready-for-agent

Builds on `.scratch/secret-santa/spec.md` and `.scratch/join-link/spec.md`. Vocabulary follows `GLOSSARY.md`. Persistence follows ADR-0001.

## Problem Statement

The names of a **Session** are fixed when the **Organizer** creates it. Real groups change:

- Someone was forgotten at creation, and there is no way to add them.
- Someone entered by mistake or no longer takes part can't be taken out, so they block the **Draw** forever (it waits for every **Participant**'s **Submission**).
- A name has a typo. Today the only fix is to start a new Session.

The existing **Release** only undoes a **Join**. It keeps the name, so it doesn't help here.

## Solution

The Organizer's Session page gets an edit form with the full list of names. Names that are already joined are fixed. Names not yet joined can be edited, removed, and new names can be added. The Organizer changes the list freely and presses **Save**, which applies everything at once.

- **Remove** deletes a name that has not been joined. A joined name can't be removed. The workaround is to **Release** it first, then remove it.
- **Add** appends a new name. It appears on the Join page immediately.
- If a removal leaves no valid Draw because of the remaining **Exclusions**, the Organizer is shown which Participants are affected and asked to confirm. On confirmation the minimum number of Exclusions is dropped, the affected Participants must confirm their **Submission** again, and they are emailed.
- After an Add, every joined Participant is emailed that a name was added and that they can still change their Exclusions on their **Personal link**.
- Opening the form of a name that doesn't exist (any more) on the Join link shows "<name> is not part of the session".

## User Stories

1. As an Organizer, I want to add a name to my Session after creating it, so that I can include someone I forgot.
2. As an Organizer, I want to remove a name that has not been joined, so that a person who doesn't take part no longer blocks the Draw.
3. As an Organizer, I want to fix a typo in a name that has not been joined, so that I don't have to start a new Session.
4. As an Organizer, I want to change several names and then press Save once, so that I don't trigger a request and emails per click.
5. As an Organizer, I want adds, removals and renames applied together, so that I can swap one person for another even at exactly 3 Participants.
6. As an Organizer, I want joined names to be fixed in the form (no remove button), so that I don't throw away someone's Submission by accident.
7. As an Organizer, I want to Release a joined name first and then remove it, so that I can still get rid of a Participant who already joined.
8. As an Organizer, I want my own name to stay in the list and not be removable, so that the Session always has an Organizer.
9. As an Organizer, I want to be told when a name is empty or a duplicate (case-insensitive), so that I catch typos before saving.
10. As an Organizer, I want my edits kept when validation fails, so that I don't have to retype them.
11. As an Organizer, I want Save rejected when it would leave fewer than 3 Participants, so that the Draw keeps its secret.
12. As an Organizer, I want a rejection with an explanation when someone joined a name I'm removing in the meantime, so that nothing is lost silently.
13. As an Organizer, I want Save to change nothing after the Draw (409), so that the Assignments stay consistent.
14. As an Organizer, I want removing the last Participant who hasn't submitted to start the Draw automatically, so that the Session doesn't get stuck.
15. As an Organizer, I want to be warned when a removal makes the Draw impossible, so that I understand what I'm about to change before it happens.
16. As an Organizer, I want that warning to list only the affected Participants' names, so that I never learn who excludes whom.
17. As an Organizer, I want to confirm explicitly before any Exclusions are dropped, so that nothing is reset without my consent.
18. As an Organizer, I want the confirmation shown again if the set of affected Participants changed in the meantime, so that I confirm what actually happens.
19. As a Participant, I want Exclusions that name a removed person dropped silently, so that I'm not bothered by something that no longer matters.
20. As a Participant whose Exclusions were dropped because of a removal, I want an email naming the removed person and linking to my Personal link, so that I can review and edit.
21. As a Participant whose Exclusions were dropped, I want to have to confirm my Submission again, so that the Draw doesn't happen before I've seen the change.
22. As a Participant, I want to keep as many of my Exclusions as possible when some must be dropped, so that my wishes are respected where they can be.
23. As a Participant who already joined, I want an email when a name is added, so that I can change my Exclusions regarding the new person.
24. As a Participant who already joined, I want the new name to appear in my Exclusions form, so that I can exclude them.
25. As a Participant affected by both an Add and a reset in one Save, I want a single combined email, so that I'm not spammed.
26. As a Participant, I want my Submission to stay valid after an Add, so that I'm not forced to confirm again.
27. As a not-yet-joined person, I want to see a newly added name on the Join page, so that I can claim it.
28. As a visitor who opened the join form for a name that was removed, I want to see "<name> is not part of the session", so that I understand why joining doesn't work.
29. As an Organizer, I want to re-add a name I removed earlier, so that a mistake can be undone without leftovers.
30. As an Organizer, I want all texts in German and English, so that the app stays consistent.
31. As an Organizer, I want Add to never start the Draw, so that a new Participant always gets a chance to join first.

## Implementation Decisions

- **Edit and Save.** The Session page shows an edit form: joined names fixed, unjoined names editable and removable, with rows addable via HTMX (like the creation form). Save posts the complete desired list of names. The server diffs it against the stored names: names missing are removals, new names are additions. The Organizer stays first, existing names keep their order, new names are appended.
- **Atomic application.** The whole Save happens in one store update, using the same submission path as Join and personal submissions (including the Draw check and automatic Draw). Everything it checks is evaluated inside the update, not on the page the Organizer saw.
- **Validation.** Reuse the creation rules for names: trimmed, non-empty, unique case-insensitively, at least the minimum number of Participants counted on the final list. The name validation is shared with creation rather than duplicated. The Organizer's email field does not apply here.
- **Rejections.** After the Draw: 409. A joined name missing from the list (including one that joined while the Organizer was editing): 409 with an explanation, nothing saved. Removing the Organizer: rejected. Validation errors re-render the form with 422 and keep the entered rows.
- **Silent cleanup.** Removed names are stripped from every Participant's Exclusions without notice, so a later Add of the same name starts clean.
- **Removal that makes the Draw impossible.**
  - Nothing is saved. The Organizer gets a confirmation page listing only the affected Participants' names, with a Confirm action that posts the full list again together with the affected names as hidden fields.
  - On confirm, the affected set is recomputed inside the update. If it differs from the posted one, the confirmation is shown again and nothing is saved.
  - To pick the Exclusions to drop: start with all Exclusions dropped, then re-add them one by one in a fixed order (Participant order, then name order), keeping each one only if the Draw stays possible. The result is deterministic and keeps as many Exclusions as possible.
  - Each affected Participant is un-submitted, so the Draw waits for them. This is required, since otherwise the Draw could fire before they see the change.
- **Add.**
  - Adding never makes the Draw impossible and never starts it (the new Participant has not submitted).
  - Existing Submissions stay valid and nobody is un-submitted. The Draw can fire when the new Participant joins as the last one; this is accepted and mitigated by the email.
  - Every joined Participant is emailed that a name was added, with their Personal link, in their own language.
- **Emails.** One email per Participant per Save. If a Participant is affected by both an Add and a reset, the content is combined. Emails name the added and removed people and contain only the Participant's own information, so nothing about others' Exclusions leaks. Emails are sent after the Save succeeded, and every send is attempted even if one fails, as in Join.
- **Draw trigger.** A Save that adds no name and resets no Submission may complete the Draw (the last blocker was removed). Draw emails are then sent as for any Draw.
- **Join page.** New names are listed immediately. Submitting or opening the form for a name that does not exist shows "<name> is not part of the session" on both the selection and the form.
- **Release** stays unchanged and still never Draws.
- **Glossary.** Add the terms **Add** and **Remove** (Organizer adds or deletes a name before the Draw, applied together by Save; avoid "Invite", "Delete", "Kick"), and loosen the **Organizer** definition, since names may now come after creation.
- **i18n.** All new texts (form labels, errors, confirmation page, both emails, Join page message) in DE and EN.

## Testing Decisions

- A good test checks external behaviour through the HTTP interface: status codes, rendered pages, emails sent and what a Personal link or the Join page shows afterwards. It doesn't assert on stored files or internal functions.
- The single seam is the existing HTTP seam via the test client with the recording mailer. Prior art: the Release tests (`release_test.go`), Join tests (`join_test.go`) and the Draw flow tests (`draw_flow_test.go`).
- The Draw algorithm keeps its existing unit tests. The only direct unit test is for choosing which Exclusions to drop (deterministic, minimal, keeps as many as possible), next to the Draw tests.
- Cases to cover (in TDD order):
  1. Remove an unjoined name; Add a name; both in one Save; a rename.
  2. Rejections: joined name, Organizer, after the Draw, fewer than 3, empty or duplicate names, a name that joined in the meantime.
  3. Removing the last blocker starts the Draw and sends Draw emails; Add never does.
  4. Stale Exclusions are stripped; a re-added name starts clean.
  5. The Draw-impossible confirmation (names only), confirm, hidden-field mismatch, un-submit, one email per Participant, combined email.
  6. The Add email to all joined Participants, the new name in their Exclusions form, Submissions stay valid.
  7. The Join page lists new names and shows "<name> is not part of the session" for removed ones.
  8. DE and EN texts.

## Out of Scope

- Removing a joined Participant in one step (Release first).
- Removing or renaming the Organizer.
- Changes after the Draw.
- Un-submitting Participants on Add.
- Protection against lost updates between two browser tabs.
- Participant limits, reordering names or bulk import.

## Further Notes

- The choice not to un-submit on Add trades some safety for less friction. A Draw can fire right after the new Participant joins, before others re-check their Exclusions. If this turns out to be a problem, un-submitting all joined Participants on Add is the follow-up.
- The Confirm step could instead recompute and apply without comparing hidden fields. Comparing was chosen so the Organizer confirms what actually happens.
- Tickets are to be created from this spec in a separate step.
