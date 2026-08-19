package dice_test

import (
	"slices"
	"testing"

	"github.com/philoserf/t5chargen/dice"
)

// TestRollProperties checks every xD±k roll shape used by Book 1: n faces
// recorded in roll order, each face 1-6, and Total = sum of faces + Mod
// ("2D - 2. Roll two dice and subtract 2; results 0 to 10", p. 19).
func TestRollProperties(t *testing.T) {
	roller := dice.New(3)

	for range 1000 {
		for n := 1; n <= 10; n++ {
			for _, mod := range []int{0, 2, -2, -7} {
				roll := roller.RollMod(n, mod)

				if roll.N != n || len(roll.Faces) != n {
					t.Fatalf("RollMod(%d, %d) recorded %d faces", n, mod, len(roll.Faces))
				}

				sum := mod

				for _, f := range roll.Faces {
					if f < 1 || f > 6 {
						t.Fatalf("face %d outside 1-6", f)
					}

					sum += f
				}

				if roll.Total != sum {
					t.Fatalf("RollMod(%d, %d) Total = %d, want %d", n, mod, roll.Total, sum)
				}
			}
		}
	}
}

func TestRollExpr(t *testing.T) {
	tests := []struct {
		roll dice.Roll
		want string
	}{
		{dice.Roll{N: 1}, "1D"},
		{dice.Roll{N: 2}, "2D"},
		{dice.Roll{N: 2, Mod: 2}, "2D+2"},
		{dice.Roll{N: 2, Mod: -7}, "2D-7"},
		{dice.Roll{N: 10}, "10D"},
	}

	for _, tt := range tests {
		if got := tt.roll.Expr(); got != tt.want {
			t.Errorf("Expr() = %q, want %q", got, tt.want)
		}
	}
}

// TestDeterminism verifies the replay contract (docs/PRD.md): the same seed
// reproduces the identical face sequence across mixed roll types.
func TestDeterminism(t *testing.T) {
	sequence := func() []int {
		roller := dice.New(99)

		faces := make([]int, 0, 1100)
		for range 100 {
			faces = append(faces, roller.Roll(3).Faces...)

			flux := roller.Flux()
			faces = append(faces, flux.First, flux.Second)

			good := roller.GoodFlux()
			faces = append(faces, good.First, good.Second)

			bad := roller.BadFlux()
			faces = append(faces, bad.First, bad.Second)

			faces = append(faces, roller.Throw(2, 8).Faces...)
		}

		return faces
	}

	if a, b := sequence(), sequence(); !slices.Equal(a, b) {
		t.Fatal("same seed produced different face sequences")
	}
}

// TestGoldenSequence pins the exact face stream for seed 1. This is the
// replay contract made concrete: if this test ever fails, the RNG's seeded
// output has changed and an engine version bump is required (docs/PRD.md,
// Replay and provenance contract).
func TestGoldenSequence(t *testing.T) {
	want := []int{6, 1, 6, 6, 2, 6, 5, 1, 4, 2, 3, 6, 3, 3, 1, 6, 1, 1, 3, 2}

	roller := dice.New(1)

	got := make([]int, 0, len(want))
	for range want {
		got = append(got, roller.Roll(1).Total)
	}

	if !slices.Equal(got, want) {
		t.Fatalf("seed 1 face stream = %v, want %v", got, want)
	}
}
