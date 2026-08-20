package chargen

// DefaultPolicy is the fixed auto-mode decision table, version
// PolicyVersion; the rules and their rationale live in POLICY.md
// (docs/PRD.md, CLI sketch: the policy is total, deterministic, and
// tie-breaks by first-listed order in Book 1).
type DefaultPolicy struct{}

// Choose applies the POLICY.md rule for the choice point.
func (DefaultPolicy) Choose(c Choice) int {
	switch c.ID {
	case ChooseControllingCharacteristic:
		// POLICY.md: highest-valued available characteristic; ties break
		// to first-listed.
		return maxScoreIndex(c)
	case ChooseSkillColumn:
		// POLICY.md: the General column; first-listed if absent.
		for i, option := range c.Options {
			if option == "General" {
				return i
			}
		}

		return 0
	case ChooseCareer, ChooseHobby, ChooseHomeworld, ChooseArt, ChooseTrade:
		// POLICY.md: first-listed.
		return 0
	}

	return 0
}

// Kind identifies the policy in choice events.
func (DefaultPolicy) Kind() DeciderKind {
	return DeciderPolicy
}

// maxScoreIndex returns the index of the highest score, first-listed on
// ties; index 0 when no scores are provided.
func maxScoreIndex(c Choice) int {
	best := 0

	for i, score := range c.Scores {
		if score > c.Scores[best] {
			best = i
		}
	}

	return best
}
