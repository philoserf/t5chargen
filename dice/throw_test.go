package dice_test

import (
	"slices"
	"testing"

	"github.com/philoserf/t5chargen/dice"
)

// TestResolveThrow checks the roll-low comparison ("The Task succeeds if the
// Die Roll is equal or less than the total of all Assets", p. 120) and the
// spectacular flags ("3 ones ... Spectacular Success", "3 sixes ...
// Spectacular Failure", p. 127) on hand-built rolls.
func TestResolveThrow(t *testing.T) {
	tests := []struct {
		name         string
		faces        []int
		target       int
		success      bool
		spectSuccess bool
		spectFailure bool
	}{
		{"equal to target succeeds", []int{3, 4}, 7, true, false, false},
		{"under target succeeds", []int{1, 2}, 7, true, false, false},
		{"over target fails", []int{4, 4}, 7, false, false, false},
		{"three ones flags spectacular success", []int{1, 1, 1}, 2, false, true, false},
		{"three ones among more dice", []int{1, 5, 1, 1}, 20, true, true, false},
		{"two ones are not spectacular", []int{1, 1, 4}, 10, true, false, false},
		{"three sixes flags spectacular failure", []int{6, 6, 6}, 20, true, false, true},
		{"two sixes are not spectacular", []int{6, 6, 2}, 7, false, false, false},
		{"spectacularly interesting: both on 6D", []int{1, 1, 1, 6, 6, 6}, 15, false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total := 0
			for _, f := range tt.faces {
				total += f
			}

			roll := dice.Roll{N: len(tt.faces), Faces: tt.faces, Total: total}
			got := dice.ResolveThrow(roll, tt.target)

			if got.Success != tt.success {
				t.Errorf("Success = %v, want %v", got.Success, tt.success)
			}

			if got.SpectacularSuccess != tt.spectSuccess {
				t.Errorf("SpectacularSuccess = %v, want %v", got.SpectacularSuccess, tt.spectSuccess)
			}

			if got.SpectacularFailure != tt.spectFailure {
				t.Errorf("SpectacularFailure = %v, want %v", got.SpectacularFailure, tt.spectFailure)
			}
		})
	}
}

// TestThrowFromStream verifies Throw consumes exactly n faces from the
// stream and resolves against the recorded roll.
func TestThrowFromStream(t *testing.T) {
	const seed = 42

	throwRoller := dice.New(seed)
	rollRoller := dice.New(seed)

	for n := 1; n <= 8; n++ {
		throw := throwRoller.Throw(n, 10)
		roll := rollRoller.Roll(n)

		if !slices.Equal(throw.Faces, roll.Faces) || throw.Total != roll.Total {
			t.Fatalf("Throw(%d, 10) rolled %v (total %d), want %v (total %d)",
				n, throw.Faces, throw.Total, roll.Faces, roll.Total)
		}

		if throw.Success != (throw.Total <= 10) {
			t.Errorf("Throw(%d, 10) Success = %v with total %d", n, throw.Success, throw.Total)
		}
	}
}

// TestCheckAutomaticFailure verifies the p. 134 rule: "Without regard to
// skill levels, any of the Checks fails on the highest possible roll. 1D
// fails on 6; 2D fails on 12; 3D fails on 18".
func TestCheckAutomaticFailure(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		faces   []int
		target  int
		success bool
	}{
		{name: "2D max fails against a reachable target", n: 2, faces: []int{6, 6}, target: 12, success: false},
		{name: "2D max fails against an unreachable target", n: 2, faces: []int{6, 6}, target: 20, success: false},
		{name: "2D below max succeeds", n: 2, faces: []int{6, 5}, target: 12, success: true},
		{name: "1D max fails", n: 1, faces: []int{6}, target: 6, success: false},
		{name: "1D below max succeeds", n: 1, faces: []int{5}, target: 6, success: true},
		{name: "3D max fails", n: 3, faces: []int{6, 6, 6}, target: 18, success: false},
		{name: "3D below max succeeds", n: 3, faces: []int{6, 6, 5}, target: 18, success: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total := 0
			for _, f := range tt.faces {
				total += f
			}

			got := dice.ResolveCheck(dice.Roll{N: tt.n, Faces: tt.faces, Total: total}, tt.target)
			if got.Success != tt.success {
				t.Errorf("Success = %v, want %v", got.Success, tt.success)
			}
		})
	}
}

// TestHighRollsHigh verifies the p. 66 Roll High procedure: "roll Soc or
// greater to be Elevated". Success is total >= target, and the modifier
// adjusts the roll.
func TestHighRollsHigh(t *testing.T) {
	r := dice.New(1)

	// Seed 1's first 2D is 6+1 = 7 (pinned by TestGoldenFaces).
	got := r.High(2, 7, 0)
	if !got.Success {
		t.Errorf("7 against a target of 7 failed; Roll High succeeds on equal or greater")
	}

	if got.Total != 7 {
		t.Errorf("total = %d, want 7", got.Total)
	}

	r = dice.New(1)
	if got := r.High(2, 8, 0); got.Success {
		t.Error("7 against a target of 8 succeeded")
	}

	r = dice.New(1)

	got = r.High(2, 8, 1)
	if !got.Success {
		t.Error("7 with a +1 mod against a target of 8 failed; the mod adjusts the roll")
	}

	if got.Total != 8 || got.Mod != 1 {
		t.Errorf("total = %d, mod = %d, want 8 and 1", got.Total, got.Mod)
	}
}

// TestResolveUnder enumerates every 2D combination against every target
// that matters, pinning the strict less-than of the Aging Check ("2D <
// Life Stage", p. 89) against the equal-or-less of Throw and Check.
func TestResolveUnder(t *testing.T) {
	for target := range 14 {
		for a := 1; a <= 6; a++ {
			for b := 1; b <= 6; b++ {
				roll := dice.Roll{N: 2, Faces: []int{a, b}, Total: a + b}

				got := dice.ResolveUnder(roll, target)
				if want := roll.Total < target; got.Success != want {
					t.Fatalf("Under(%d vs %d) = %v, want %v", roll.Total, target, got.Success, want)
				}

				// The boundary is the whole point: a roll equal to the
				// target succeeds as a Throw and fails as an Under.
				if roll.Total == target && dice.ResolveThrow(roll, target).Success == got.Success {
					t.Fatalf("Under and Throw agree at the boundary %d", target)
				}
			}
		}
	}
}

// TestUnderKeepsTheMaximumRollAFailure holds the note in Under's doc
// comment: the p. 134 automatic-failure rule is not applied because it
// would change nothing. A 2D roll of 12 is never under a Life Stage,
// which caps at 9.
func TestUnderKeepsTheMaximumRollAFailure(t *testing.T) {
	roll := dice.Roll{N: 2, Faces: []int{6, 6}, Total: 12}
	for target := range 10 {
		if dice.ResolveUnder(roll, target).Success {
			t.Errorf("12 succeeded against Life Stage %d", target)
		}
	}
}
