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
