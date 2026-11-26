# 06: Automatic Draw and Draw emails

**What to build:** As soon as the last Participant submits, the Draw happens automatically. Each Participant receives an email, in their own language, naming the Participant they give to along with that person's Wish. From then on the Personal link shows that Assignment read-only, and the Session page only shows that the Session is drawn.

**Blocked by:** 03, 04

**Status:** ready-for-agent

Spec: `.scratch/secret-santa/spec.md` · Vocabulary: `GLOSSARY.md` · Persistence: ADR-0001

- [ ] After a Submission is saved, if every Participant is submitted, the Draw runs, Assignments are persisted and the Session becomes `Drawn`, all atomically under the store lock (two simultaneous last Submissions cannot draw twice)
- [ ] Each Participant receives one Draw email with the recipient's name and Wish, in the language from their last Submission
- [ ] After the Draw, the Personal link shows the recipient's name and Wish read-only; Wish and Exclusions can no longer be edited (POSTs are rejected)
- [ ] A Participant never learns who gives to them
- [ ] After the Draw, the Session page shows only "Drawn ✓" (no Wishes, Exclusions or Assignments)
- [ ] Seam A tests: full flow from create to Draw; Draw emails satisfy the invariants (everyone gives once and receives once, no self-Assignment, Exclusions respected) and use each Participant's language; post-Draw pages and locking
