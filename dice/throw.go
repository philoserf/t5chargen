package dice

// Throw is the recorded outcome of a roll-low target-number throw.
//
// Success is the pure p. 122 comparison ("If the die roll is equal to or
// less than the target number, the task is successful"). The spectacular
// flags (p. 127) are reported separately and do not alter Success; the rule
// that a Spectacular Success succeeds "even if the result would otherwise be
// a failure" is applied by the call sites that use it, with their own cites.
type Throw struct {
	Roll

	Target  int  `json:"target"`
	Success bool `json:"success"`

	// SpectacularSuccess: "Three Ones. If the actual dice roll includes
	// 3 ones (but not possible on 1D or 2D) the result is a Spectacular
	// Success" (p. 127).
	SpectacularSuccess bool `json:"spectacular_success,omitempty"`

	// SpectacularFailure: "Three Sixes. If the actual dice roll includes
	// 3 sixes (not possible on 1D or 2D), the result is a Spectacular
	// Failure." (p. 127)
	SpectacularFailure bool `json:"spectacular_failure,omitempty"`
}

// Throw rolls n dice against target.
//
// "Roll Low: The Task succeeds if the Die Roll is equal or less than the
// total of all Assets." (p. 120) The target passed here is that total: the
// engine sums characteristic, skill, and mods before calling (Mods change
// the Target Number, DMs change the Die Roll — p. 18); the generation
// record keeps the itemized breakdown (docs/PRD.md FR10).
func (r *Roller) Throw(n, target int) Throw {
	return resolveThrow(r.Roll(n), target)
}

// Check rolls n dice against target as a Check: a Throw plus the
// automatic-failure rule. "Automatic Failure. Without regard to skill
// levels, any of the Checks fails on the highest possible roll. 1D fails
// on 6; 2D fails on 12; 3D fails on 18." (p. 134)
//
// Character generation resolves its throws as Checks (chart 10 restates
// the rule beside its starred Risk & Reward and Continue rows: "But, 12 is
// always automatic failure"). This is what guarantees a career can end:
// a Continue target at or above the maximum roll would otherwise never
// fail.
func (r *Roller) Check(n, target int) Throw {
	return resolveCheck(r.Roll(n), target)
}

// resolveCheck applies the p. 134 automatic-failure rule on top of the
// plain comparison. Pure arithmetic, split from the stream for exhaustive
// testing.
func resolveCheck(roll Roll, target int) Throw {
	throw := resolveThrow(roll, target)
	if roll.Total >= maxRoll(roll) {
		throw.Success = false
	}

	return throw
}

// maxRoll is the highest possible total for the dice rolled: all sixes,
// plus the roll's fixed modifier.
func maxRoll(roll Roll) int {
	return roll.N*6 + roll.Mod
}

// resolveThrow applies the p. 120/122 comparison and the p. 127 spectacular
// detection to an already-rolled Roll. Pure arithmetic, split from the
// stream for exhaustive testing.
func resolveThrow(roll Roll, target int) Throw {
	return Throw{
		Roll:               roll,
		Target:             target,
		Success:            roll.Total <= target,
		SpectacularSuccess: countFace(roll.Faces, 1) >= 3,
		SpectacularFailure: countFace(roll.Faces, 6) >= 3,
	}
}

// countFace reports how many dice in faces show the given face.
func countFace(faces []int, face int) int {
	count := 0

	for _, f := range faces {
		if f == face {
			count++
		}
	}

	return count
}
