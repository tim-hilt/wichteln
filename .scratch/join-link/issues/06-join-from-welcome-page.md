# 06: "Join a Session" on the welcome page

**What to build:** The welcome page offers a third input, "join a Session", accepting a Join hash or a full Join link, which redirects to `/join/<hash>`.

**Blocked by:** 02

**Status:** ready-for-agent

Spec: `.scratch/join-link/spec.md`

- [ ] A bare Join hash or a full Join link (with query, fragment or trailing slash) redirects to `/join/<hash>`
- [ ] The Join hash of a drawn Session also redirects (to the "joining is closed" page)
- [ ] An unknown hash re-renders the welcome page with an inline "not found" and the input kept
- [ ] The "open a Personal link" input does not accept Join hashes
- [ ] All new texts in DE and EN
