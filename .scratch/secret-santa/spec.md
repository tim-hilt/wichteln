# Spec: Secret Santa (Wichteln)

Status: ready-for-agent

## Problem Statement

A group of friends, family or colleagues wants to do Secret Santa ("Wichteln"). The organizing itself (budget, date, exchange) happens elsewhere, but the parts that must stay **secret** are awkward to do by hand:

- collecting everyone's **Wish** without revealing it to the others,
- respecting **Exclusions** (e.g. spouses shouldn't draw each other),
- running a fair random **Draw** in which nobody, not even the **Organizer**, learns anyone else's **Assignment**.

Doing this by hand (drawing paper slips, or one person assigning everyone) means that someone either knows everything or the Exclusions get messed up. Existing online tools need accounts, show ads, or aren't in German.

## Solution

A small, self-hosted web app with no accounts. Access is controlled by unguessable secret links:

- The **Organizer** creates a **Session** by entering all **Participants** (name + email), including themselves ("Ich"/"Me"). The Session lives at a secret **Session link** that the Organizer bookmarks.
- On submit, every Participant gets an email with their **Personal link**. There they enter a **Wish** and optional one-way **Exclusions**, then submit. They can edit both until the Draw.
- The Organizer's Session page only shows who has submitted and who is still pending. It never shows Wishes, Exclusions or Assignments.
- When the last Participant submits, the **Draw** happens automatically. Each Participant gets an email naming the Participant they give to, together with that person's Wish. The Personal link shows the same information from then on.
- All visible text, including emails, is in German or English depending on the browser language. There is no language selector.

## User Stories

### Landing page

1. As a visitor, I want a welcome page offering "create a new Session", "open an existing Session" and "open a Personal link", so that I immediately know where to go.
2. As a German-speaking visitor, I want all text in German when my browser prefers German, so that the app feels native.
3. As a visitor whose browser doesn't prefer German, I want all text in English, so that I can still use the app.
4. As a visitor, I don't want a language selector, so that the UI stays minimal.
5. As an Organizer, I want to paste my Session hash *or* the full Session link into the "open Session" field, so that I don't have to cut out the hash myself.
6. As a Participant, I want to paste my Personal link hash *or* the full Personal link into the "open Personal link" field, so that I don't have to cut out the hash myself.
7. As an Organizer who mistyped my Session hash on the welcome page, I want to see "Session not found", so that I don't accidentally create a duplicate Session.
8. As a Participant who mistyped my Personal link hash on the welcome page, I want to see "not found", so that I know the link is wrong.

### Creating a Session

9. As an Organizer, I want "create a new Session" to take me to a fresh Session link (`/session/<hash>`), so that I can bookmark it right away.
10. As an Organizer, I want the form to start with a pre-labelled "Ich"/"Me" row for my own name and email, so that I'm included as a Participant.
11. As an Organizer, I want to add and remove Participant rows (name + email) without full page reloads, so that entering a group is quick.
12. As an Organizer, I want nothing stored until I submit, so that abandoned attempts leave no trace.
13. As an Organizer, I want to be required to enter at least 3 Participants (including me), so that the Draw keeps the secret.
14. As an Organizer, I want an error if two Participants have the same name (case-insensitive), so that names in Exclusions and Assignments are unambiguous.
15. As an Organizer, I want to use the same email address for several Participants (e.g. my kids), so that people without their own email can take part.
16. As an Organizer, I want an error for empty names or invalid-looking email addresses, so that I catch typos before submitting.
17. As an Organizer, I want my entered rows kept when validation fails, so that I don't have to retype everything.
18. As an Organizer, I want every Participant (including me) to get an email with their Personal link when I submit, so that everyone can enter their Wish.
19. As an Organizer, I want the Participant list frozen after submission, so that the Draw is based on a stable group.
20. As an Organizer, I want the invitation emails written in my browser language, so that the group gets emails in the language I set the Session up in.
21. As someone who opens a Session link that doesn't exist on the server, I want to see the empty creation form, so that the link works as a "create here" URL.

### Session page (Organizer, before the Draw)

22. As an Organizer, I want to open my Session link at any time to see each Participant's name, email and status (submitted / pending), so that I know whom to nudge.
23. As an Organizer who is also a Participant, I don't want to see anyone's Wish, Exclusions or Assignment on the Session page, so that the secret stays intact for me too.
24. As an Organizer, I want to correct a Participant's email address before the Draw, which re-sends their Personal link to the new address, so that a typo doesn't block the whole Session.
25. As an Organizer, I want a "resend link" action per Participant before the Draw, so that someone who lost their email can get it again.
26. As an Organizer, I don't want to be able to add, remove or rename Participants after submission, so that the group stays stable.

### Personal link (before the Draw)

27. As a Participant, I want my Personal link (`/<hash>`) to show a page greeting me by name, so that I know it's my page.
28. As a Participant, I want to enter my Wish as free text (links or a description), so that my giver knows what I'd like.
29. As a Participant, I want to be required to enter a non-empty Wish (up to ~2000 characters), so that my giver always has something to go on. ("Überrasch mich" is a valid Wish.)
30. As a Participant, I want to select other Participants I must not give to (one-way Exclusions), so that e.g. I don't draw my spouse.
31. As a Participant, I want to understand that my Exclusions only restrict whom *I* give to, not who gives to me, so that I set them correctly.
32. As a Participant, I don't want to see Exclusions other Participants created, especially ones targeting me, so that nobody's feelings get hurt.
33. As a Participant, I want an error if my Exclusions (combined with everyone else's current Exclusions) would make the Draw impossible, so that the Session never reaches a dead end.
34. As a Participant, I want to submit my Wish and Exclusions together, so that my status flips to "submitted".
35. As a Participant, I want to come back via my Personal link and edit my Wish and Exclusions until the Draw, so that I can change my mind.
36. As a Participant, I want my edits after submitting to keep me "submitted", so that editing doesn't block the Draw.
37. As a Participant, I want my browser language remembered on each submission, so that the Draw email reaches me in my language.
38. As a visitor opening an unknown Personal link, I want a plain "not found" page, so that nothing about other Sessions leaks.

### Draw

39. As a Participant, I want the Draw to happen automatically as soon as the last Participant submits, so that nobody has to trigger it.
40. As a Participant, I want to give to exactly one other Participant and receive from exactly one other Participant, so that everyone gets one present.
41. As a Participant, I never want to be assigned to myself, so that the game works.
42. As a Participant, I want the Draw to respect every Exclusion, so that my constraints are honored.
43. As a Participant, I want the Draw to be random, so that the Assignment is fair and unpredictable.
44. As a Participant, I don't want to learn who gives to me, so that the surprise stays.
45. As a Participant, I want to receive an email after the Draw with the name and Wish of the Participant I give to, written in my own language, so that I can start shopping.
46. As a Participant, I want my Personal link to show my Assignment (recipient name + Wish) read-only after the Draw, so that I can look it up if the email gets lost.
47. As a Participant, I don't want to be able to edit my Wish or Exclusions after the Draw, so that my giver's information stays stable.
48. As an Organizer, I want the Session page to show only "Drawn ✓" after the Draw, so that I know it's done without seeing any Assignments.
49. As an Organizer, I don't want to be able to correct emails or resend invitation links after the Draw, so that the finished Session stays fixed.

### Operations

50. As a self-hoster, I want a single small Docker image (`FROM scratch`) with all assets embedded, so that deployment is trivial.
51. As a self-hoster, I want Sessions persisted to a data directory I can mount as a volume, so that data survives restarts.
52. As a self-hoster, I want emails logged to stdout while SMTP isn't configured, so that I can run and test the app without a mail provider.
53. As a self-hoster, I want the public base URL configurable, so that links in emails point to my domain.
54. As a self-hoster, I want Sessions never deleted, so that there is no cleanup logic to worry about (traffic is very low).

## Implementation Decisions

Vocabulary follows `GLOSSARY.md`. Persistence follows ADR-0001.

- **Stack**: stdlib-only Go, server-side rendered `html/template`, HTMX for partial updates (vendored and embedded together with templates and CSS), multi-stage Docker build ending `FROM scratch`. GitHub Actions builds the image and runs tests.
- **Session lifecycle**: `Open` → `Drawn`. No draft state: a Session first exists on the server when the Organizer submits it.
- **Secret hashes**: 128 bits from `crypto/rand`, encoded as 26-character lowercase base32. Session hashes and Personal link hashes are independent. The base32 alphabet and length mean a hash can never collide with fixed routes.
- **Routes (user-visible contract)**:
  - `GET /`: welcome page with the three actions.
  - "Create" generates a new Session hash and redirects to `/session/<hash>`.
  - `GET /session/<hash>`: empty creation form if unknown; Session status page if `Open`; "Drawn ✓" if `Drawn`.
  - `POST` to the Session link: create the Session (validation: ≥3 Participants, unique names case-insensitive, non-empty names, plausible emails; duplicate emails allowed). Fails if the hash already exists.
  - Session-page actions (before the Draw only): correct a Participant's email address (re-sends their Personal link) and resend a Personal link.
  - `GET /<hash>`: Personal link page (edit form before the Draw, read-only Assignment after); plain 404 page if unknown.
  - `POST` to the Personal link: Submission (Wish + Exclusions).
  - The welcome-page forms accept either a bare hash or a full URL, extract the hash, and check that it exists before redirecting. Otherwise they re-render with a "not found" error.
- **Domain model (persisted per Session)**: Session hash, state, Organizer language, list of Participants, and, once drawn, Assignments. Each Participant has an id, name, email, Personal link hash, whether they are the Organizer, Wish, Exclusions (set of other Participant ids), submitted flag, and language from the last Submission. Assignments map each giver to a recipient.
- **Lookup by Personal link**: the store must find a Session by a Personal link hash. Either keep an in-memory index built at startup from the data directory or use a separate index file. This is an internal choice of the storage module.
- **Storage module (deep module)**: hides the JSON-file-per-Session layout, atomic writes (temp file + rename), a process-wide mutex, and the Personal link index. Its interface exposes domain operations (create Session, load by Session hash, load by Personal link hash, save) and nothing file-related.
- **Draw module (pure)**: input is Participants plus Exclusions; output is Assignments or "impossible". Constraints: a permutation with no fixed points (nobody gives to themselves), and no giver → recipient pair that the giver excluded. Exclusions are one-way: giver A excluding B only forbids A → B. Uses randomized search/backtracking (group sizes are small) so that every valid Assignment set can come out. Random source is `crypto/rand`, or an injectable source for tests.
- **Feasibility check on Submission**: before saving a Participant's Exclusions, run the Draw module against all current Exclusions (with the new ones applied). If it is impossible, reject with a localized error and don't save. Because Exclusions can only become impossible through a Submission, the automatic Draw can always succeed.
- **Automatic Draw**: after saving a Submission, if every Participant is submitted, run the Draw, persist Assignments, set the state to `Drawn`, then send Draw emails. This happens atomically under the store mutex, so two simultaneous last Submissions can't draw twice.
- **Privacy rules (enforced in rendering)**: the Session page never renders Wishes, Exclusions or Assignments. A Personal link page never renders other Participants' Exclusions or who gives to the Participant.
- **i18n**: two locales, DE and EN. For page text, choose DE if the top-ranked supported language in `Accept-Language` is German, otherwise EN.
  - Invitation and resend emails use the Organizer's language, stored when the Session is created.
  - Draw emails use each Participant's language from their last Submission.
  - All strings live in a single message catalog keyed by message id. Templates look them up via a template function.
- **Mailer interface (adapter)**: a single "send email" capability taking a Mail (to, subject, plain-text and HTML version). Two production adapters: SMTP (`net/smtp`; implicit TLS on port 465, STARTTLS otherwise) and stdout, used when no SMTP server is configured. Emails are `multipart/alternative`: plain text plus HTML styled like the website (inline styles, table layout). Links are built from a configurable base URL.
- **Configuration via environment**: listen address, data directory, base URL, and SMTP server, port, credentials and sender address (`MAIL_FROM`). Nothing deployment-specific is built in.
- **Application constructor**: one constructor builds the root `http.Handler` from (data directory, Mailer, base URL, random source). This constructor is the test seam.

## Testing Decisions

- **Good tests** exercise external behavior only: HTTP requests in, HTML and emails out. They never reach into storage files or internal structs, and expected values come from the spec, not recomputed from the implementation. Because Assignments are random, tests assert invariants (each Participant gives once and receives once, no self-Assignment, Exclusions respected), not specific pairings.
- **Seam A: the root `http.Handler`** (primary, agreed with the user). Built with a temp data directory and a recording fake Mailer, driven with `httptest`. Tests follow the user flows end to end: create a Session, read Personal links from the captured emails, submit Wishes and Exclusions, observe the automatic Draw and the Draw emails, and check the Session page and Personal link pages afterwards. Also covers:
  - `Accept-Language` DE/EN switching and email languages,
  - validation errors,
  - unknown hashes (welcome-page "not found", empty form at an unknown Session link, 404 at an unknown Personal link),
  - privacy (the Session page never contains Wish text; the excluded Participant's page never mentions the Exclusion),
  - email correction and resend,
  - locking after the Draw,
  - the infeasible-Exclusion error,
  - persistence across a handler rebuilt on the same data directory.
- **Seam B: the pure Draw function** (secondary, agreed). Property-style repeated runs over Exclusion graphs, e.g.:
  - 3 Participants without Exclusions,
  - someone excluding everyone,
  - Exclusions that leave exactly one valid permutation,
  - one-way vs. two-way Exclusions,
  - impossible configurations reported as impossible.

  Repeated runs should also show that more than one valid outcome occurs when several exist.
- No other seams: storage, the hash generator, i18n and template internals are tested only through Seam A.
- **Prior art**: none. The repo is greenfield, so these tests set the pattern.

## Out of Scope

- Budget, exchange date, Organizer notes, or any other logistics. These are organized outside the app, which only handles what must be secret.
- An Organizer who doesn't participate.
- Draft Sessions / saving before submit.
- Changing the Participant set after submission (only email corrections and resends).
- A manual "Draw now" button or drawing with pending Participants.
- Mutual pairs (A↔B) as a matching mode.
- Deleting Sessions, accounts, authentication beyond secret links, rate limiting.
- A language selector or languages other than DE/EN.
- Email-failure handling beyond reporting the error (no retries or outbox).

## Further Notes

- Security model is "security by obscurity" via 128-bit secret links (see README). Anyone holding a link has full access to that page.
- A Participant probing Exclusions could, in theory, learn something about others' Exclusions from the feasibility error. This is accepted given the audience.
- Concurrency is low, but the automatic Draw must still be atomic (see Implementation Decisions).
