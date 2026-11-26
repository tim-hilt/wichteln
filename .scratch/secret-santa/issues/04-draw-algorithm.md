# 04: Draw algorithm (pure)

**What to build:** A pure Draw function: given Participants and their one-way Exclusions, it returns random Assignments in which everyone gives exactly once and receives exactly once, nobody gives to themselves, and no Exclusion is violated. If no such Assignment set exists, it reports "impossible".

**Blocked by:** 01 (only for the Go module; can run in parallel with 02 and 03)

**Status:** ready-for-agent

Spec: `.scratch/secret-santa/spec.md` · Vocabulary: `GLOSSARY.md` · Persistence: ADR-0001

- [ ] Input: Participants plus Exclusions (giver → excluded recipient); output: Assignments or "impossible"
- [ ] Exclusions are one-way: A excluding B forbids only A → B
- [ ] Randomized (e.g. shuffled backtracking) so every valid Assignment set can occur; random source is injectable
- [ ] Seam B property-style tests over repeated runs: 3 Participants without Exclusions; someone excluding everyone (impossible); Exclusions leaving exactly one valid permutation; one-way vs two-way Exclusions; impossible configurations reported as impossible
- [ ] A test shows that more than one outcome occurs across runs when several valid Assignment sets exist
- [ ] Every returned result is checked against the invariants (permutation, no self-Assignment, Exclusions respected), not against specific pairings
