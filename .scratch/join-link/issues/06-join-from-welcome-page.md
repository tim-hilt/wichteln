# 06: "Join a Session" on the welcome page

**What to build:** The welcome page offers a third input, "join a Session", accepting a Join hash or a full Join link, which redirects to `/join/<hash>`.

**Blocked by:** 02

**Status:** done

Spec: `.scratch/join-link/spec.md`

- [x] A bare Join hash or a full Join link (with query, fragment or trailing slash) redirects to `/join/<hash>`
- [x] The Join hash of a drawn Session also redirects (to the "joining is closed" page)
- [x] An unknown hash re-renders the welcome page with an inline "not found" and the input kept
- [x] The "open a Personal link" input does not accept Join hashes
- [x] All new texts in DE and EN
