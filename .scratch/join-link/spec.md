# Spec: Join link

Status: ready-for-agent

Builds on `.scratch/secret-santa/spec.md`. Vocabulary follows `GLOSSARY.md`. Persistence follows ADR-0001.

## Problem Statement

Today the **Organizer** has to enter every **Participant**'s name **and email address** when creating a **Session**. That is awkward and bad for privacy:

- The Organizer often doesn't know everyone's address and has to collect them first.
- The Organizer sees every Participant's email address, which the Participants never agreed to.
- A typo in someone else's address is the Organizer's problem to find and fix.
- The Organizer has to remember to bookmark the **Session link**. If they lose it, the Session is unreachable.

## Solution

The Organizer only enters the **names** of the Participants, plus their own name and their own email address. The Session gets a second, independent secret URL, the **Join link**, which the Organizer shares with the group (e.g. in a group chat).

- When someone opens the Join link, they see only the names that haven't been joined yet, and pick their own.
- They then land on a page that greets them by that name, so that a mis-click is obvious. There they enter their own email address, **Wish** and **Exclusions**. Submitting **Joins** the name: their first **Submission** happens and the name is claimed.
- They are redirected to their **Personal link** and receive a confirmation email containing it.
- The Organizer's name is joined at creation. Right after creating the Session, the Organizer receives an email with their Session link, the Join link and their Personal link, so nothing can get lost.
- The Organizer's Session page shows the Join link and, per name, whether it has been joined. It never shows email addresses. If a name was joined by the wrong person or with a mistyped address, the Organizer **Releases** it so it can be joined again.
- The **Draw** still happens automatically as soon as every Participant has made a Submission.

The Join link reveals nothing but the not-yet-joined names, and it gives no access to the Session page.

## User Stories

### Creating a Session

1. As an Organizer, I want "create a new Session" to still take me to a fresh Session link (`/session/<hash>`), so that creating works as before.
2. As an Organizer, I want the creation form to start with a pre-labelled "Ich"/"Me" row asking for my name and my email address, so that I'm included as a Participant and can be reached.
3. As an Organizer, I want every other row to ask only for a name, so that I don't have to collect anyone's email address.
4. As an Organizer, I want to add and remove name rows without full page reloads, so that entering a group is quick.
5. As an Organizer, I want to be required to enter at least 3 names (including mine), so that the Draw keeps the secret.
6. As an Organizer, I want an error for empty names, duplicate names (case-insensitive) or an invalid-looking email address of my own, so that I catch typos before submitting.
7. As an Organizer, I want my entered rows kept when validation fails, so that I don't have to retype everything.
8. As an Organizer, I want nothing stored until I submit, so that abandoned attempts leave no trace.
9. As an Organizer, I want nobody but me to be emailed when I create the Session, so that the group only hears about it through the Join link I share.
10. As an Organizer, I want an email right after creation containing my Session link, the Join link and my Personal link, so that I can't lose any of them.
11. As an Organizer, I want that email in my browser language, so that I can read it.
12. As an Organizer, I want my own name to be joined at creation and never offered on the Join page, so that nobody can take my place.
13. As an Organizer, I want the list of names frozen after creation, so that the Draw is based on a stable group.

### Session page (Organizer, before the Draw)

14. As an Organizer, I want the Session page to show the Join link, so that I can copy and share it.
15. As an Organizer, I want to see each name with its status ("not joined" / "joined"), so that I know whom to nudge.
16. As an Organizer, I want my own row to show "pending" until I have saved my Wish, then "joined", so that I don't forget my own Submission.
17. As an Organizer, I don't want to see any Participant's email address, Wish, Exclusions or Assignment, so that their privacy and the secret stay intact.
18. As an Organizer, I want a "Resend link" action for each joined Participant, which emails their Personal link to the address they entered without showing it to me, so that someone who lost their email can get it again.
19. As an Organizer, I want "Resend link" on my own row to send me my complete email again (Session link, Join link, Personal link), so that I can recover all my links at once.
20. As an Organizer, I want a "Release" action for each joined Participant except me, so that a name joined by the wrong person or with a mistyped email address can be joined again.
21. As an Organizer, I want a Release to discard that Participant's email address, Wish and Exclusions and to make their old Personal link stop working, so that whoever joined by mistake has no access left.
22. As an Organizer, I want a Release to send no email, so that a stranger's or a mistyped address isn't contacted again.
23. As an Organizer, I don't want to be able to add, remove or rename names after creation, so that the group stays stable.

### Join link

24. As a Participant, I want the Join link (`/join/<hash>`) to ask "select your name" and list only the names that haven't been joined yet, so that I can find mine quickly.
25. As a Participant, I don't want the Join page to show email addresses, statuses, Wishes or anything else about the Session, so that the Join link reveals as little as possible.
26. As a Participant, I want selecting my name to open a page greeting me by that name, so that I notice immediately if I clicked the wrong one.
27. As a Participant, I want to enter my email address, my Wish and my Exclusions on that page, so that I can join and make my Submission in one step.
28. As a Participant, I want the same validation as for any Submission (non-empty Wish up to ~2000 characters, Exclusions that keep the Draw possible) plus a plausible email address, so that the Session never reaches a dead end.
29. As a Participant, I want nothing stored until I submit that page, so that clicking a name and leaving doesn't block it.
30. As a Participant, I want a clear "this name was just taken" message if someone else joined the same name a moment before me, so that I know what happened.
31. As a Participant, I want to be redirected to my Personal link after joining, so that I can see my saved Wish right away.
32. As a Participant, I want a confirmation email with my Personal link after joining, written in my browser language, so that I don't have to remember to save the link.
33. As a Participant, I want to use the same email address as another Participant (e.g. a parent for their kids), so that people without their own email can take part.
34. As a Participant, I want the Join link to show a plain "joining is closed" page once the Session is drawn, so that I know I'm too late.
35. As a visitor opening an unknown Join link, I want the usual "not found" page, so that nothing about other Sessions leaks.
36. As a Participant, I want to see all other names on my Personal link page (as Exclusion choices), as before, so that I can set Exclusions.

### Personal link (before the Draw)

37. As a Participant, I want to see and change my own email address on my Personal link page, together with my Wish and Exclusions, so that I can fix a typo myself.
38. As a Participant, I want a changed email address to receive the confirmation email with my Personal link again, so that I know the new address works.
39. As a Participant, I want saving without changing my address to send no email, so that I'm not spammed on every edit.
40. As the Organizer, I want a changed own address to receive my complete email (Session link, Join link, Personal link), so that all my links reach the new address.

### Welcome page

41. As a Participant, I want a "join a Session" input on the welcome page that accepts a Join hash or a full Join link, so that I can join even if I only have the hash.
42. As a Participant who mistyped a Join hash on the welcome page, I want to see "not found" inline with my input kept, so that I can fix it.
43. As a Participant entering the Join hash of a drawn Session, I want to be taken to the "joining is closed" page, so that I get the same answer as with the link itself.
44. As a visitor, I don't want the "open a Personal link" input to accept Join hashes, so that the two kinds of links stay distinct.

### Draw

45. As a Participant, I want the Draw to happen automatically once every name has been joined and every Participant (including the Organizer) has made a Submission, so that nobody has to trigger it.
46. As a Participant, I don't want a Release to trigger the Draw, so that the released name gets its chance to be joined again.
47. As a Participant, I want the Draw email sent to the address I entered myself, so that it reaches me.

## Implementation Decisions

- **Persisted Session gains a Join link hash**: an independent 128-bit hash, the same format as all other hashes and generated at creation. The store must find a Session by its Join link hash, just as it does by Personal link hash. This is an internal concern of the storage module, e.g. by extending its in-memory index.
- **Participant state**: a Participant created from a name row has no email, no Personal link hash and is not submitted, so "joined" means "has a Personal link". The Organizer's Participant has an email and a Personal link from creation but is not submitted, which is the "pending" state on the Session page. The Draw condition stays "all submitted". A Participant who hasn't joined can't be submitted, so it implies "all joined".
- **Release** resets a joined non-Organizer Participant to the never-joined state: email, Personal link hash, Wish, Exclusions, submitted flag and language are cleared, and the old Personal link index entry is removed. Other Participants' Exclusions naming them stay, because they are by name. Release is only allowed before the Draw (409 after it). Releasing the Organizer or an unjoined name is rejected.
- **Feasibility is unaffected**: the set of names is fixed at creation, unjoined Participants have no Exclusions, and a Release only removes Exclusions. So the existing feasibility check on Submission still guarantees that the automatic Draw succeeds.
- **Join is one atomic store update**: it checks that the Session is Open and the name is still unjoined, validates the input, runs the feasibility check, assigns a fresh Personal link hash, saves the Submission, and runs the automatic Draw if this was the last missing Submission. All of this happens under the store's lock, so two people joining the same name can't both succeed. The loser gets 409 with a localized "this name was just taken" message.
- **Routes (user-visible contract)**:
  - `GET /join/<hash>`: "select your name" with the unjoined names; "joining is closed" if drawn; not-found page if unknown.
  - Selecting a name opens the join form for that name (name in the URL or query; the exact shape is up to the implementation). An unknown or already joined name shows the selection again.
  - `POST` of the join form: Join (see above); on success, redirect to `/<personal-hash>`.
  - `POST /session/<hash>/release`: Release a Participant by name.
  - `POST /session/<hash>/resend` remains. The Organizer's "correct email" action and route are removed.
  - `POST /open-join` on the welcome page: accepts a hash or a full link and redirects to `/join/<hash>` (also when drawn); inline "not found" for unknown hashes.
- **Creation form**: the first row has name and email; the other rows have name only. Validation: at least 3 rows, non-empty names, unique names ignoring case, plausible email for the Organizer only.
- **Personal link form** gains an email field (required, plausible). If the address changed, the confirmation email is sent to the new address after saving.
- **Emails**:
  - *Organizer email*: Session link, Join link and Personal link. Sent at creation, on Resend for the Organizer, and when the Organizer changes their address. Language: the Organizer's language.
  - *Confirmation email*: Personal link. Sent on Join, on Resend, and on an address change. Language: the Participant's language from their last Submission (the Join counts as one).
  - The email view must support several labelled links; today it has a single optional link.
  - The Draw email is unchanged.
- **i18n**: new catalog keys for the Join page, the join form greeting, "joining is closed", "name just taken", the statuses "not joined" / "joined" / "pending", Release, the Join link label on the Session page, the welcome-page join input, and the Organizer email texts (DE and EN).
- **Existing Sessions on disk** are not migrated. They have no Join link hash, so their Session page shows no Join link. Everything else keeps working because all their Participants are already joined. There is no live data to protect.

## Testing Decisions

- **Good tests** exercise external behavior only: HTTP requests in, HTML and emails out. They never inspect storage files or internal structs. Expected values come from the specs. Because the Draw is random, tests assert invariants, not pairings.
- **Single seam: the root `http.Handler`** built by the application constructor, with a temp data directory and the recording fake Mailer, driven with `httptest`. Links are taken from the captured emails: the Organizer email yields the Join link and the Organizer's Personal link; confirmation emails yield the Personal links. Covers:
  - creation validation (name-only rows, Organizer email required, ≥3 names),
  - the Organizer email (exactly one email at creation, containing all three links),
  - the Join page listing only unjoined names and never the Organizer's,
  - the join form's greeting, validation and feasibility error,
  - two joins of the same name (second gets "just taken"),
  - the redirect and confirmation email after joining,
  - the Session page statuses and the absence of any email address on it,
  - Resend for the Organizer and other Participants,
  - Release (name reappears on the Join page, old Personal link returns 404, no email sent, rejected for the Organizer and after the Draw),
  - email change on the Personal link (email only when changed),
  - the automatic Draw after the last Join / Submission, and "joining is closed" afterwards,
  - the welcome-page join input (hash, full link, unknown, drawn),
  - DE/EN for the new pages and emails,
  - persistence of Join link lookups across a handler rebuilt on the same data directory.
- The pure Draw function's tests are unchanged. No new seams.
- **Prior art**: the existing handler-level tests for Session creation, the Personal link, the Draw flow, opening links from the welcome page and resend. The shared test helpers that open a Session and collect Personal links from emails need to switch to the join flow.

## Out of Scope

- Adding, removing or renaming names after creation.
- Revoking or regenerating the Join link.
- A manual "Draw now" or "close joining" action.
- The Organizer seeing or editing anyone's email address.
- Notifying the Organizer when someone joins.
- Migrating Sessions created before this change.
- Protection against someone deliberately joining a wrong name beyond Release (no codes, no email verification).

## Further Notes

- The Join link page does show the names that haven't been joined yet. This is accepted: every Participant sees all names on their Personal link anyway.
- Anyone holding the Join link can join any unjoined name. Release is the remedy, and the Organizer sees which names are joined.
- This supersedes these stories of the secret-santa spec: #10, #15–#16 (emails now entered by Participants), #18, #20 and #24 (no Organizer-entered emails, no invitations at creation, no Organizer email correction), and #22 (emails no longer shown).
