package main

import (
	"errors"
	"math/rand/v2"
)

// DrawPossible reports whether the Session's Participants, with their current
// Exclusions, admit at least one valid Draw. Exclusions naming unknown
// Participants are ignored.
func DrawPossible(participants []Participant) bool {
	index := map[string]int{}
	for i, p := range participants {
		index[p.Name] = i
	}
	excluded := map[int][]int{}
	for i, p := range participants {
		for _, name := range p.Exclusions {
			if j, ok := index[name]; ok {
				excluded[i] = append(excluded[i], j)
			}
		}
	}
	// Feasibility does not depend on the randomness, so any fixed seed works.
	_, err := Draw(len(participants), excluded, rand.New(rand.NewPCG(0, 0)))
	return err == nil
}

// ErrNoDraw reports that no set of Assignments satisfies the Exclusions.
var ErrNoDraw = errors.New("no valid draw exists")

// Draw randomly creates Assignments for n Participants numbered 0..n-1.
// excluded maps a giver to the recipients they must not give to (one-way).
//
// The result maps each giver to their recipient, so it is a permutation of
// 0..n-1 without fixed points that respects all Exclusions. It returns
// ErrNoDraw if none exists. Randomness comes from rng, so tests can seed it.
//
// Givers pick in turn from a freshly shuffled candidate list and keep the
// first candidate that still leaves every later giver a distinct allowed
// recipient (a perfect matching). So every valid result can occur (not
// necessarily equally likely), and the Draw never backtracks: it stays
// polynomial even for large groups or when no valid result exists, where
// plain backtracking would take exponential time.
func Draw(n int, excluded map[int][]int, rng *rand.Rand) ([]int, error) {
	if n < 2 {
		return nil, ErrNoDraw
	}
	// allowed[g][r] reports whether giver g may give to recipient r.
	allowed := make([][]bool, n)
	for g := range allowed {
		allowed[g] = make([]bool, n)
		for r := range allowed[g] {
			allowed[g][r] = r != g
		}
		for _, r := range excluded[g] {
			allowed[g][r] = false
		}
	}
	if !perfectMatching(allowed) {
		return nil, ErrNoDraw
	}

	recipients := make([]int, n)
	for g := range n {
		for _, r := range rng.Perm(n) {
			if !allowed[g][r] {
				continue
			}
			if fixed := fix(allowed, g, r); perfectMatching(fixed) {
				allowed, recipients[g] = fixed, r
				break
			}
		}
	}
	return recipients, nil
}

// fix returns a copy of allowed in which g gives to r and nobody else does.
func fix(allowed [][]bool, g, r int) [][]bool {
	out := make([][]bool, len(allowed))
	for i, row := range allowed {
		out[i] = make([]bool, len(row))
		if i == g {
			out[i][r] = true
			continue
		}
		copy(out[i], row)
		out[i][r] = false
	}
	return out
}

// perfectMatching reports whether every giver can get a distinct allowed
// recipient (Kuhn's augmenting-path algorithm, O(n³)).
func perfectMatching(allowed [][]bool) bool {
	n := len(allowed)
	giverOf := make([]int, n) // recipient -> matched giver, or -1
	for r := range giverOf {
		giverOf[r] = -1
	}
	var augment func(g int, seen []bool) bool
	augment = func(g int, seen []bool) bool {
		for r, ok := range allowed[g] {
			if ok && !seen[r] {
				seen[r] = true
				if giverOf[r] < 0 || augment(giverOf[r], seen) {
					giverOf[r] = g
					return true
				}
			}
		}
		return false
	}
	for g := range n {
		if !augment(g, make([]bool, n)) {
			return false
		}
	}
	return true
}
