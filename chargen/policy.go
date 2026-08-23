package chargen

// DefaultPolicy is the fixed auto-mode decision table, version
// PolicyVersion; the rules and their rationale live in POLICY.md
// (docs/PRD.md, CLI sketch: the policy is total, deterministic, and
// tie-breaks by first-listed order in Book 1).
type DefaultPolicy struct{}

// Choose applies the POLICY.md rule for the choice point.
//
//nolint:exhaustive // Deliberately partitioned: the name- and preference-based rules are in chooseNamed.
func (DefaultPolicy) Choose(c Choice) int {
	if index, ok := chooseNamed(c); ok {
		return index
	}

	switch c.ID {
	case ChooseControllingCharacteristic, ChooseCheck:
		// POLICY.md: highest-valued characteristic; ties break to
		// first-listed.
		return maxScoreIndex(c)
	case ChooseBranch:
		// POLICY.md: the lowest Branch Mod, then the lowest Branch DM.
		// The Mod is negative against Risk, so the lowest is the least
		// injurious; the DM pushes the Operations roll off the end of its
		// table, collapsing the term's assignments — and with them the
		// skill columns those assignments open.
		return minScoreIndex(c)
	case ChooseElevationFlux, ChooseComeback:
		return chooseOnScore(c)
	case ChooseCareerWaiver:
		// POLICY.md: waive only a career-ending outcome. Waivers share one
		// decaying pool with education (I-22), so spending one on a
		// rejected publication makes the next harder for no lasting gain.
		return declineUnlessCareerEnding(c)
	case ChooseHonors, ChooseWaiver, ChooseRetry, ChooseAdvancement, ChooseSpecialty,
		ChooseOptionalFlux, ChooseBeginTrack, ChooseTenure:
		// POLICY.md: always attempt (index 0). Honors failure has no
		// effect (p. 59); waiver attempts burn future waiver odds (Mod
		// minus previous waivers) but the immediate stake outweighs it;
		// the I-8 Reward retry has no stated cost; a commission or
		// promotion attempt has no stated cost either, and rank carries
		// skills and muster-out benefits. The same index-0 answer covers
		// the first-listed rows: chart 06's "To Begin 4th Officer" is the
		// highest berth on offer, chart 03 ranks its specialties only by
		// die face, and the optional Fame Flux is offered only while
		// another roll can help.
		return 0
	case ChooseSchemeFlux:
		// POLICY.md: as rolled (index 0). Chart 10 ranks its schemes only
		// by Flux, and a richer scheme is not a better one: the Value
		// multiplies a payoff the Reward roll may never earn, while the
		// row itself changes nothing about the odds.
		return 0
	case ChooseCareerChange:
		// POLICY.md: decline. The change is offered before the Continue
		// throw is known, so it trades a career in hand for a To Begin
		// that may fail and end resolution outright (p. 66, p. 65), and
		// aging now bounds the mandatory-continue tail the change exists
		// to avoid.
		return 0
	case ChooseHobby, ChooseHomeworld, ChooseArt, ChooseTrade,
		ChooseService, ChooseMajor, ChooseMinor, ChooseSkill,
		ChooseAssociatedCareer:
		// POLICY.md: first-listed.
		return 0
	}

	return 0
}

// chooseNamed applies the POLICY.md rules that pick an option by name or
// from a preference list, reporting whether the choice point is one of
// them.
//
//nolint:exhaustive // Deliberately partitioned: the remaining rules are in Choose.
func chooseNamed(c Choice) (int, bool) {
	if c.ID == ChooseFameFlux {
		// POLICY.md: invoke only when Flux could reach Fame 19.
		return fameFluxChoice(c), true
	}

	switch c.ID {
	case ChooseSkillColumn:
		// POLICY.md: the first present of General, then Exploration, then
		// Business — the all-plain-skills columns of the shipped careers;
		// first-listed otherwise.
		return preferredIndex(c.Options,
			[]string{"General", "Exploration", "Business", "Combat", "Peacekeeper", "Mission"}, 0), true
	case ChooseDuty:
		// POLICY.md: Explorer Duty — the career's point, and the larger
		// skill eligibility (chart 05 table B).
		return indexOrFirst(c.Options, "Explorer Duty"), true
	case ChooseRiskMod:
		// POLICY.md: No Mod.
		return indexOrFirst(c.Options, "No Mod"), true
	case ChooseEducation:
		return chooseEducationProgram(c.Options), true
	case ChooseCareer:
		// POLICY.md: Citizen by name. The alternatives are listed in
		// chart order, so first-listed would hand the default career to
		// whichever chart number is lowest among those implemented —
		// changing every generated character each time an earlier chart
		// lands.
		return indexOrFirst(c.Options, "Citizen"), true
	default:
		return 0, false
	}
}

// declineUnlessCareerEnding applies POLICY.md's waiver rule: attempt only
// where the un-waived outcome ends the career. The stake is carried on the
// Choice rather than inferred from its prompt, so rewording a reason
// cannot silently change generated characters.
func declineUnlessCareerEnding(c Choice) int {
	if c.CareerEnding {
		return 0
	}

	return 1
}

// chooseOnScore applies the POLICY.md rules that turn on the engine's
// decision aid: the Noble invokes its once-per-career Elevation Flux only
// once the bare 2D can no longer reach Soc, and the Entertainer takes a
// Comeback only below the mean of the 2D that replaces Fame.
func chooseOnScore(c Choice) int {
	if len(c.Scores) == 0 {
		return 1
	}

	take := c.Scores[0] < comebackThreshold
	if c.ID == ChooseElevationFlux {
		take = c.Scores[0] >= maxTwoDice
	}

	if take {
		return 0
	}

	return 1
}

// maxTwoDice is the highest total a 2D Roll High can reach unaided. At or
// above it an Elevation needs the maximum roll or is outright impossible,
// which is where the once-per-career Flux is worth spending (chart 11,
// p. 85).
const maxTwoDice = 12

// comebackThreshold is the Fame below which the default policy takes a
// Comeback: the mean of the 2D that replaces it (chart 03, p. 77).
const comebackThreshold = 7

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

// minScoreIndex returns the index of the lowest score, first-listed on
// ties; index 0 when no scores are provided.
func minScoreIndex(c Choice) int {
	best := 0

	for i, score := range c.Scores {
		if score < c.Scores[best] {
			best = i
		}
	}

	return best
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

// fameFluxThreshold is the Fame that earns an extra muster-out roll: "He
// is allowed one additional roll if Fame 19+" (p. 68).
const fameFluxThreshold = 19

// fluxRange is the most Flux can add or subtract (dice.Flux, p. 19).
const fluxRange = 5

// fameFluxChoice decides chart F's once-per-character gamble. The engine
// passes the Fame so far as a score, so the policy weighs the same number
// a player would see.
func fameFluxChoice(c Choice) int {
	if len(c.Scores) == 0 {
		return 0
	}

	base := c.Scores[0]

	// Already there, or out of reach: the gamble can only lose.
	if base >= fameFluxThreshold || base+fluxRange < fameFluxThreshold {
		return 0
	}

	return 1
}
