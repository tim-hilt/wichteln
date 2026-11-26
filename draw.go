package main

import (
	"errors"
	"math/rand/v2"
	"slices"
)

// ErrNoDraw reports that no set of Assignments satisfies the Exclusions.
var ErrNoDraw = errors.New("no valid draw exists")

// Draw randomly creates Assignments for n Participants numbered 0..n-1.
// excluded maps a giver to the recipients they must not give to (one-way).
//
// The result maps each giver to their recipient, so it is a permutation of
// 0..n-1 without fixed points that respects all Exclusions. It returns
// ErrNoDraw if none exists. Randomness comes from rng, so tests can seed it.
//
// It is shuffled backtracking: givers pick from a freshly shuffled candidate
// list, so every valid result can occur (not necessarily equally likely).
func Draw(n int, excluded map[int][]int, rng *rand.Rand) ([]int, error) {
	if n < 2 {
		return nil, ErrNoDraw
	}
	recipients := make([]int, n)
	taken := make([]bool, n)

	var assign func(giver int) bool
	assign = func(giver int) bool {
		if giver == n {
			return true
		}
		for _, r := range rng.Perm(n) {
			if taken[r] || r == giver || slices.Contains(excluded[giver], r) {
				continue
			}
			taken[r], recipients[giver] = true, r
			if assign(giver + 1) {
				return true
			}
			taken[r] = false
		}
		return false
	}

	if !assign(0) {
		return nil, ErrNoDraw
	}
	return recipients, nil
}
