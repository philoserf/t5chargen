package dice_test

import (
	"testing"

	"github.com/philoserf/t5chargen/dice"
)

// TestARollTheRulesCannotExpress pins that a non-positive dice count rolls
// nothing rather than panicking in make. Every count the engine passes is
// a constant or a chart field floored at load, so this is a backstop —
// but RollMod is exported, and CLAUDE.md's rule is that nothing exported
// panics on a value outside the rules' range.
func TestARollTheRulesCannotExpress(t *testing.T) {
	r := dice.New(1)

	for _, n := range []int{0, -1, -9} {
		roll := r.RollMod(n, 3)
		if len(roll.Faces) != 0 || roll.Total != 3 {
			t.Errorf("RollMod(%d, 3) = %d faces totalling %d, want 0 faces totalling 3",
				n, len(roll.Faces), roll.Total)
		}
	}
}

// TestUnderDoesNotReportRollLowSpectaculars pins that an Under throw
// leaves the p. 127 flags clear, as High does and for the same reason:
// they read the dice the way a roll-low Throw does, and on an Under the
// caller wants the throw to fail.
func TestUnderDoesNotReportRollLowSpectaculars(t *testing.T) {
	// Sweep until three ones come up on 3D; that is the roll that would
	// otherwise be reported as a Spectacular Success.
	for seed := range uint64(400) {
		throw := dice.New(seed).Under(3, 9)
		if throw.Total != 3 {
			continue
		}

		if throw.SpectacularSuccess || throw.SpectacularFailure {
			t.Fatalf("seed %d: three ones under 9 reported spectacular %v/%v",
				seed, throw.SpectacularSuccess, throw.SpectacularFailure)
		}

		return
	}

	t.Error("no seed in 400 rolled three ones on 3D; widen the sweep")
}
