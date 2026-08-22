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
		// POLICY.md: the first present of General, then Exploration (the
		// all-plain-skills columns of the shipped careers); first-listed
		// otherwise.
		return preferredIndex(c.Options, []string{"General", "Exploration", "Business"}, 0)
	case ChooseDuty:
		// POLICY.md: Explorer Duty — the career's point, and the larger
		// skill eligibility (chart 05 table B).
		return indexOrFirst(c.Options, "Explorer Duty")
	case ChooseRiskMod:
		// POLICY.md: No Mod.
		return indexOrFirst(c.Options, "No Mod")
	case ChooseEducation:
		return chooseEducationProgram(c.Options)
	case ChooseBeginTrack:
		// POLICY.md: the highest berth the chart offers — first-listed,
		// which is chart 06's "To Begin 4th Officer".
		return 0
	case ChooseHonors, ChooseWaiver, ChooseRetry, ChooseAdvancement:
		// POLICY.md: always attempt (index 0). Honors failure has no
		// effect (p. 59); waiver attempts burn future waiver odds (Mod
		// minus previous waivers) but the immediate stake outweighs it;
		// the I-8 Reward retry has no stated cost; a commission or
		// promotion attempt has no stated cost either, and rank carries
		// skills and muster-out benefits.
		return 0
	case ChooseCareer, ChooseHobby, ChooseHomeworld, ChooseArt, ChooseTrade,
		ChooseService, ChooseMajor, ChooseMinor, ChooseSkill:
		// POLICY.md: first-listed.
		return 0
	}

	return 0
}

// chooseEducationProgram applies POLICY.md's college track: University,
// then College, then ED5; None (always last) otherwise. Service Academy is
// excluded — its Officer1 graduation links to milestone-3 military
// careers.
func chooseEducationProgram(options []string) int {
	return preferredIndex(options, []string{"University", "College", "ED5"}, len(options)-1)
}

// preferredIndex returns the index of the first present preference, or the
// fallback index.
func preferredIndex(options, preferences []string, fallback int) int {
	for _, want := range preferences {
		for i, option := range options {
			if option == want {
				return i
			}
		}
	}

	return fallback
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
