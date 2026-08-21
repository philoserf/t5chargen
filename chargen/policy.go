package chargen

// DefaultPolicy is the fixed auto-mode decision table, version
// PolicyVersion; the rules and their rationale live in POLICY.md
// (docs/PRD.md, CLI sketch: the policy is total, deterministic, and
// tie-breaks by first-listed order in Book 1).
type DefaultPolicy struct{}

// Choose applies the POLICY.md rule for the choice point.
func (DefaultPolicy) Choose(c Choice) int {
	switch c.ID {
	case ChooseControllingCharacteristic, ChooseCheck:
		// POLICY.md: highest-valued characteristic; ties break to
		// first-listed.
		return maxScoreIndex(c)
	case ChooseSkillColumn:
		// POLICY.md: the General column; first-listed if absent.
		return indexOrFirst(c.Options, "General")
	case ChooseEducation:
		// POLICY.md: the college track — University, then College, then
		// ED5; None otherwise. Service Academy is excluded (its Officer1
		// graduation links to milestone-3 military careers).
		for _, want := range []string{"University", "College", "ED5"} {
			for i, option := range c.Options {
				if option == want {
					return i
				}
			}
		}

		return len(c.Options) - 1 // None, always last
	case ChooseHonors, ChooseWaiver:
		// POLICY.md: always attempt (index 0). Honors failure has no
		// effect (p. 59); waiver attempts burn future waiver odds (Mod
		// minus previous waivers) but the immediate stake outweighs it.
		return 0
	case ChooseCareer, ChooseHobby, ChooseHomeworld, ChooseArt, ChooseTrade,
		ChooseService, ChooseMajor, ChooseMinor, ChooseSkill:
		// POLICY.md: first-listed.
		return 0
	}

	return 0
}

// indexOrFirst returns the index of want in options, or 0.
func indexOrFirst(options []string, want string) int {
	for i, option := range options {
		if option == want {
			return i
		}
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
