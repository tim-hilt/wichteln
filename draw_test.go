package main

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

const drawRuns = 200

func newRng(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, 1)) }

// checkInvariants fails unless got is a valid Draw result: a permutation of
// 0..n-1, no self-Assignment, no Exclusion violated.
func checkInvariants(t *testing.T, n int, excl map[int][]int, got []int) {
	t.Helper()
	if len(got) != n {
		t.Fatalf("got %d assignments, want %d", len(got), n)
	}
	received := map[int]bool{}
	for giver, recipient := range got {
		if recipient < 0 || recipient >= n || received[recipient] {
			t.Fatalf("not a permutation: %v", got)
		}
		received[recipient] = true
		if giver == recipient {
			t.Fatalf("self-assignment in %v", got)
		}
		for _, x := range excl[giver] {
			if x == recipient {
				t.Fatalf("exclusion %d→%d violated in %v", giver, x, got)
			}
		}
	}
}

func TestDrawValidAssignments(t *testing.T) {
	cases := map[string]struct {
		n    int
		excl map[int][]int
	}{
		"three without exclusions": {3, nil},
		"ten without exclusions":   {10, nil},
		"one-way exclusion":        {3, map[int][]int{0: {1}}},
		"two-way exclusion":        {4, map[int][]int{0: {1}, 1: {0}}},
		"exactly one permutation":  {3, map[int][]int{0: {1}, 1: {2}, 2: {0}}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			for seed := range uint64(drawRuns) {
				got, err := Draw(c.n, c.excl, newRng(seed))
				if err != nil {
					t.Fatalf("seed %d: %v", seed, err)
				}
				checkInvariants(t, c.n, c.excl, got)
			}
		})
	}
}

func TestDrawExclusionsAreOneWay(t *testing.T) {
	// 0 excludes 1, so 0→1 never happens; but 1→0 must still occur.
	excl := map[int][]int{0: {1}}
	sawReverse := false
	for seed := range uint64(drawRuns) {
		got, err := Draw(3, excl, newRng(seed))
		if err != nil {
			t.Fatal(err)
		}
		checkInvariants(t, 3, excl, got)
		sawReverse = sawReverse || got[1] == 0
	}
	if !sawReverse {
		t.Error("1→0 never drawn: exclusion treated as two-way")
	}
}

func TestDrawImpossible(t *testing.T) {
	cases := map[string]struct {
		n    int
		excl map[int][]int
	}{
		"someone excludes everyone": {3, map[int][]int{0: {1, 2}}},
		"two cannot give to one":    {3, map[int][]int{0: {2}, 1: {2}}},
		"single participant":        {1, nil},
		"nobody":                    {0, nil},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := Draw(c.n, c.excl, newRng(1))
			if !errors.Is(err, ErrNoDraw) {
				t.Fatalf("got %v, %v; want ErrNoDraw", got, err)
			}
		})
	}
}

// Large groups must stay fast even when no Draw exists or only few do, since
// the feasibility check runs on every Submission.
func TestDrawScalesToLargeGroups(t *testing.T) {
	const n = 40
	nobodyGivesTo0 := map[int][]int{}
	for g := 1; g < n; g++ {
		nobodyGivesTo0[g] = []int{0}
	}
	if _, err := Draw(n, nobodyGivesTo0, newRng(1)); !errors.Is(err, ErrNoDraw) {
		t.Fatalf("got %v, want ErrNoDraw", err)
	}

	// Everyone may only give to the next one or the one after: few valid Draws.
	tight := map[int][]int{}
	for g := range n {
		for r := range n {
			if r != (g+1)%n && r != (g+2)%n {
				tight[g] = append(tight[g], r)
			}
		}
	}
	for seed := range uint64(20) {
		got, err := Draw(n, tight, newRng(seed))
		if err != nil {
			t.Fatal(err)
		}
		checkInvariants(t, n, tight, got)
	}
}

func TestDrawVaries(t *testing.T) {
	seen := map[string]bool{}
	for seed := range uint64(drawRuns) {
		got, err := Draw(4, nil, newRng(seed))
		if err != nil {
			t.Fatal(err)
		}
		seen[fmt.Sprint(got)] = true
	}
	if len(seen) < 2 {
		t.Errorf("only %d distinct outcome across %d runs", len(seen), drawRuns)
	}
}

func TestDropExclusionsKeepsAsManyAsPossibleDeterministically(t *testing.T) {
	// Dropped in Participant order, then name order: Anna's Exclusion stays,
	// Ben's would clash with it, and of Cleo's only the one on Anna fits.
	mk := func() []Participant {
		return []Participant{
			{Name: "Anna", Exclusions: []string{"Ben"}},
			{Name: "Ben", Exclusions: []string{"Anna"}},
			{Name: "Cleo", Exclusions: []string{"Ben", "Anna"}},
		}
	}
	ps := mk()
	affected := dropExclusions(ps)
	if !DrawPossible(ps) {
		t.Fatal("Draw still impossible")
	}
	if got := strings.Join(affected, ","); got != "Ben,Cleo" {
		t.Errorf("affected %q, want Ben,Cleo", got)
	}
	if got := ps[0].Exclusions; len(got) != 1 || got[0] != "Ben" {
		t.Errorf("Anna's Exclusions %v, want [Ben]", got)
	}
	if len(ps[1].Exclusions) != 0 {
		t.Errorf("Ben's Exclusions %v, want none", ps[1].Exclusions)
	}
	again := mk()
	dropExclusions(again)
	if fmt.Sprint(again) != fmt.Sprint(ps) {
		t.Errorf("not deterministic: %v vs %v", again, ps)
	}
}
