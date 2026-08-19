package dice_test

import (
	"testing"

	"github.com/philoserf/t5chargen/dice"
)

// TestFluxDistributions enumerates all 36 two-die combinations for each Flux
// variant and checks the value counts against the printed probability tables
// of Book 1 Appendix 8 (p. 261).
func TestFluxDistributions(t *testing.T) {
	tests := []struct {
		kind dice.FluxKind
		want map[int]int // value -> count out of 36, per the p. 261 tables
	}{
		{dice.FluxStandard, map[int]int{-5: 1, -4: 2, -3: 3, -2: 4, -1: 5, 0: 6, 1: 5, 2: 4, 3: 3, 4: 2, 5: 1}},
		{dice.FluxGood, map[int]int{0: 6, 1: 10, 2: 8, 3: 6, 4: 4, 5: 2}},
		{dice.FluxBad, map[int]int{0: 6, -1: 10, -2: 8, -3: 6, -4: 4, -5: 2}},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			got := map[int]int{}

			for first := 1; first <= 6; first++ {
				for second := 1; second <= 6; second++ {
					roll := dice.NewFlux(tt.kind, first, second)
					if roll.First != first || roll.Second != second {
						t.Errorf("NewFlux(%v, %d, %d) recorded faces (%d, %d)",
							tt.kind, first, second, roll.First, roll.Second)
					}

					got[roll.Value]++
				}
			}

			for value, want := range tt.want {
				if got[value] != want {
					t.Errorf("value %+d: got %d/36, want %d/36", value, got[value], want)
				}
			}

			for value := range got {
				if _, ok := tt.want[value]; !ok {
					t.Errorf("value %+d occurred but is outside the printed range", value)
				}
			}
		})
	}
}

// TestFluxQuickRollEquivalence verifies "Flux can be quick rolled as 2D - 7.
// The results and the probabilities are the same" (p. 261): the standard
// Flux distribution over all 36 combinations equals the 2D-7 distribution.
func TestFluxQuickRollEquivalence(t *testing.T) {
	flux := map[int]int{}
	twoDMinus7 := map[int]int{}

	for a := 1; a <= 6; a++ {
		for b := 1; b <= 6; b++ {
			flux[dice.NewFlux(dice.FluxStandard, a, b).Value]++
			twoDMinus7[a+b-7]++
		}
	}

	for value := -5; value <= 5; value++ {
		if flux[value] != twoDMinus7[value] {
			t.Errorf("value %+d: Flux %d/36, 2D-7 %d/36", value, flux[value], twoDMinus7[value])
		}
	}
}

// TestFluxStreamOrder verifies the documented face-consumption order: the
// light die is drawn from the stream before the dark die ("always subtract
// dark from light", p. 19).
func TestFluxStreamOrder(t *testing.T) {
	const seed = 7

	for range 50 {
		fluxRoller := dice.New(seed)
		faceRoller := dice.New(seed)

		for range 20 {
			got := fluxRoller.Flux()
			light := faceRoller.Roll(1).Total
			dark := faceRoller.Roll(1).Total

			if got.First != light || got.Second != dark || got.Value != light-dark {
				t.Fatalf("Flux() = (%d, %d, %+d), want light %d then dark %d",
					got.First, got.Second, got.Value, light, dark)
			}
		}
	}
}

func TestFluxRollExpr(t *testing.T) {
	tests := []struct {
		kind dice.FluxKind
		want string
	}{
		{dice.FluxStandard, "Flux"},
		{dice.FluxGood, "Good Flux"},
		{dice.FluxBad, "Bad Flux"},
	}

	for _, tt := range tests {
		if got := dice.NewFlux(tt.kind, 3, 4).Expr(); got != tt.want {
			t.Errorf("Expr() for %v = %q, want %q", tt.kind, got, tt.want)
		}
	}
}
