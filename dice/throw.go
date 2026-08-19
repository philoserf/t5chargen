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
