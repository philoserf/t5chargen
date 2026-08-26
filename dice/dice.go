// Package dice implements the Traveller5 dice mechanics: xD (±k) rolls,
// Flux and its Good/Bad variants, and roll-low target-number throws.
//
// All rule citations are to Traveller5 Core Rules Book 1, Print Edition 5.1.
// "Traveller uses Six Sided Dice. Only D6 dice are used in Traveller." (p. 18)
//
// Every result carries the individual die faces in roll order so the engine
// can emit complete generation-record events (docs/PRD.md FR10).
//
// Deliberately not implemented (out of scope for character generation):
//   - Many Dice procedures (p. 260) — apply only to rolls of 11D or more.
//   - D/2 half-die (p. 19) — "Rarely used."
//   - Hasty/Cautious difficulty shifts and the This Is Hard! rule
//     (pp. 120, 127) — task-layer adjustments, not dice mechanics; callers
//     adjust the dice count or target before calling into this package.
package dice

import (
	"math/rand/v2"
	"strconv"
)

// Roller draws die faces from a seeded PCG stream. It is the only source
// of randomness in the engine (docs/PRD.md FR9): every roll consumes faces
// from the stream in a fixed, documented order.
//
// A Roller is not safe for concurrent use; the engine consumes it
// sequentially so that replay is deterministic.
type Roller struct {
	rng *rand.Rand
}

// New returns a Roller seeded with seed. The single user-facing seed is
// expanded to the two PCG state words as NewPCG(seed, seed).
//
// This expansion, the PCG algorithm, and the face-consumption order of each
// roll method are version-locked by the replay and provenance contract
// (docs/PRD.md): changing any of them is an engine version bump.
func New(seed uint64) *Roller {
	//nolint:gosec // G404: a deterministic, seeded, non-cryptographic stream is the requirement (docs/PRD.md FR9).
	return &Roller{rng: rand.New(rand.NewPCG(seed, seed))}
}

// Roll rolls n dice and sums them.
//
// "D (Capital D) indicates that a standard six-sided die is used. The number
// in front of the die tells how many of these dice to roll" and "2D. Roll
// two dice: results 2 to 12 (or 8D: Roll eight dice for results 8 to 48)"
// (p. 19).
func (r *Roller) Roll(n int) Roll {
	return r.RollMod(n, 0)
}

// RollMod rolls n dice, sums them, and applies the fixed adjustment mod.
//
// "any addition (or subtraction) after the D indicates how the die roll
// result is changed. ... 2D - 2. Roll two dice and subtract 2; results 0 to
// 10. ... 2D - 7. Roll two dice and subtract 7. This may produce negative
// numbers" (p. 19).
func (r *Roller) RollMod(n, mod int) Roll {
	// A count the rules cannot express rolls nothing rather than
	// panicking in make. Every count reaching here is a constant or a
	// chart field with a floor checked at load, so this is a backstop
	// and not a behaviour — but it is exported, and CLAUDE.md's rule for
	// a value outside the rules' range is that nothing exported panics.
	// An empty roll shows in the record as 0D, which is visibly wrong,
	// where a panic takes the process with it.
	if n < 1 {
		return Roll{N: n, Faces: []int{}, Mod: mod, Total: mod}
	}

	faces := make([]int, n)
	total := mod

	for i := range faces {
		faces[i] = r.face()
		total += faces[i]
	}

	return Roll{N: n, Faces: faces, Mod: mod, Total: total}
}

// face draws one die face, 1 to 6, from the seeded stream.
func (r *Roller) face() int {
	return r.rng.IntN(6) + 1
}

// Roll is the recorded outcome of an xD±k roll.
type Roll struct {
	N     int   `json:"n"`             // number of dice rolled
	Faces []int `json:"faces"`         // individual die faces, in roll order
	Mod   int   `json:"mod,omitempty"` // fixed adjustment (the ±k in "2D-2")
	Total int   `json:"total"`         // sum of Faces plus Mod
}

// Expr renders the roll's dice expression in Book 1 notation: "2D", "2D-2",
// "3D+1" (p. 19).
func (roll Roll) Expr() string {
	expr := strconv.Itoa(roll.N) + "D"

	switch {
	case roll.Mod > 0:
		expr += "+" + strconv.Itoa(roll.Mod)
	case roll.Mod < 0:
		expr += strconv.Itoa(roll.Mod)
	}

	return expr
}
