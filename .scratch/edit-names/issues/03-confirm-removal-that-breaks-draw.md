# 03: Confirm a removal that makes the Draw impossible

**What to build:** When a Save would leave no valid Draw because of the remaining Exclusions, nothing is saved. The Organizer sees which Participants are affected, by name only, and confirms. On confirmation the minimum number of Exclusions is dropped, the affected Participants have to confirm their Submission again, and each gets an email.

**Blocked by:** 02

**Status:** done

Spec: `.scratch/edit-names/spec.md`

- [x] A Save that makes the Draw impossible saves nothing and shows a confirmation page with the affected Participants' names, never who excludes whom
- [x] Confirm posts the full list again with the affected names as hidden fields
- [x] On confirm the affected set is recomputed inside the update; if it differs from the posted one, the confirmation is shown again and nothing is saved
- [x] The Exclusions to drop are chosen deterministically: all dropped, then re-added in Participant order then name order, each kept only if the Draw stays possible; covered by a unit test
- [x] Each affected Participant is un-submitted, so the Draw waits for them
- [x] Each affected Participant gets one email naming the removed person, linking to their Personal link, in their own language
- [x] After they confirm again, the Draw runs as usual
- [x] All new texts in DE and EN
