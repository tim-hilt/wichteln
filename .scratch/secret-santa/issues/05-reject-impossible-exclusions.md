# 05: Reject impossible Exclusions

**What to build:** When a Participant submits Exclusions that, combined with everyone else's current Exclusions, would make the Draw impossible, the Submission is rejected with a clear localized message and nothing is saved, so the Session can never reach a dead end.

**Blocked by:** 03, 04

**Status:** ready-for-agent

Spec: `.scratch/secret-santa/spec.md` · Vocabulary: `GLOSSARY.md` · Persistence: ADR-0001

- [ ] On Submission, the Draw algorithm is run against all current Exclusions with the new ones applied
- [ ] If impossible, the form re-renders with a localized error, keeping the Participant's input, and nothing is persisted
- [ ] Feasible Submissions behave exactly as before
- [ ] Seam A test: a Session where one Participant's Exclusions are rejected because of the others' Exclusions, and the stored state is unchanged (verified via the Personal link page)
