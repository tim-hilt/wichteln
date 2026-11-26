# 07: Open a Session or Personal link from the welcome page

**What to build:** On the welcome page, the Organizer can paste a Session hash or full Session link to open their Session, and a Participant can paste a Personal link hash or full Personal link to open their page. Unknown hashes show a "not found" error instead of silently creating something.

**Blocked by:** 03

**Status:** done

Spec: `.scratch/secret-santa/spec.md` · Vocabulary: `GLOSSARY.md` · Persistence: ADR-0001

- [x] Both forms accept either a bare hash or a full URL and extract the hash
- [x] An existing Session hash redirects to `/session/<hash>`; an existing Personal link hash redirects to `/<hash>`
- [x] An unknown hash re-renders the welcome page with a localized "not found" error and the input kept
- [x] Opening `/session/<unknown>` directly still shows the empty creation form (unchanged)
- [x] Seam A tests for bare hash, full URL and unknown hash, for both forms
