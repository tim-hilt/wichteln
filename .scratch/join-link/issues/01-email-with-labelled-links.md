# 01: Email view with several labelled links

**What to build:** Prefactor: an email can carry several labelled links (e.g. Session link, Join link, Personal link), each rendered in both the plain-text and the HTML version. Existing emails (invitation, resend, Draw) look exactly as before.

**Blocked by:** None (can start immediately)

**Status:** done

Spec: `.scratch/join-link/spec.md`

- [x] An email can contain zero, one or several links, each with a label, in both the text and HTML parts
- [x] Existing invitation and Draw emails are unchanged (existing tests stay green)
- [x] No user-visible behaviour change
