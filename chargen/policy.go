package chargen

// DefaultPolicy is the fixed auto-mode decision table, version
// PolicyVersion; the rules and their rationale live in POLICY.md
// (docs/PRD.md, CLI sketch: the policy is total, deterministic, and
// tie-breaks by first-listed order in Book 1).
type DefaultPolicy struct{}

// Choose applies the POLICY.md rule for the choice point. The policy is
// total, so it never refuses: the error is always nil.
func (p DefaultPolicy) Choose(c Choice) (int, error) {
	return p.decide(c), nil
}

// chooseNamed applies the POLICY.md rules that pick an option by name or
// from a preference list, reporting whether the choice point is one of
// them.
//
//nolint:exhaustive // Deliberately partitioned: the remaining rules are in Choose.
func chooseNamed(c Choice) (int, bool) {
	if c.ID == ChooseCashOut {
		// POLICY.md: keep it. A pension outlives five years of itself
		// for any character who lives, and the record is more useful
		// carrying the stream than the lump.
		return 0, true
	}

	if c.ID == ChooseBenefitDM {
		// POLICY.md: apply the DM in full. The tables run from cheap to
		// dear, so a partial DM only ever reaches a lower row.
		return len(c.Options) - 1, true
	}

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
		return chooseEducationProgram(c), true
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

// declineUnlessAtStake applies POLICY.md's waiver rule for both kinds: the
// engine marks the attempt with 1 where refusing ends the career, the
// process or the admission, and 0 where it costs only a status. The stake
// is carried on the Choice rather than inferred from its prompt, so
// rewording a reason cannot silently change generated characters.
func declineUnlessAtStake(c Choice) int {
	if len(c.Scores) > 0 && c.Scores[0] == 0 {
		return 1
	}

	return 0
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
// excluded on the same criterion: its Officer1 graduation is implemented
// (interpretation I-94), but the policy is choosing for Edu and the
// Academy graduates at Edu=8 — below University, level with the College it
// would displace.
func chooseEducationProgram(c Choice) int {
	// Chart C's rows are all offered now, qualifying or not, so the
	// preference is taken over the ones the character actually meets: the
	// engine marks those with a Score of 1 (education.go, offeredPrograms).
	// Reaching for a row he falls short of would spend a waiver from the
	// pool education and the careers share (I-22) on a program the policy
	// picked only because it was listed.
	qualifying := make([]string, 0, len(c.Options))

	for i, option := range c.Options {
		if i < len(c.Scores) && c.Scores[i] == 1 {
			qualifying = append(qualifying, option)
		}
	}

	// The engine always scores "None" 1, so there is at least one
	// qualifying row; the guard keeps the policy total (it never refuses,
	// and it must never panic) against a Choice that carries no Scores.
	if len(qualifying) == 0 {
		return len(c.Options) - 1
	}

	want := preferredIndex(qualifying, []string{"University", "College", "ED5"}, len(qualifying)-1)

	return indexOrFirst(c.Options, qualifying[want])
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

// decide is the decision table itself, split from Choose so the rules read
// as the plain index-returning switch POLICY.md documents.
//
//nolint:exhaustive // Deliberately partitioned: the name- and preference-based rules are in chooseNamed.
func (DefaultPolicy) decide(c Choice) int {
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
	case ChooseWaiver, ChooseCareerWaiver:
		// POLICY.md: waive what is at stake. Both waiver rules reduce to
		// that — a career waiver is spent only where the un-waived branch
		// ends the career, an educational one only where refusing ends
		// the process or the admission rather than costing a status.
		// Waivers decay for every attempt out of one pool shared with the
		// careers (I-22), so a waiver spent on nothing at risk makes the
		// next real one harder.
		return declineUnlessAtStake(c)
	case ChooseHonors, ChooseRetry, ChooseAdvancement, ChooseSpecialty,
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
	case ChooseCareerChange, ChooseLaterEducation:
		// POLICY.md: decline both (index 0), for the same reason in two
		// shapes — the term or career in hand is worth more than what is
		// offered for it. A career change is offered before the Continue
		// throw is known, so it trades a career for a To Begin that may
		// fail and end resolution outright (p. 66, p. 65). Later
		// Education trades a term of career for a term of school the
		// policy cannot value, and the offer recurs every term, so a
		// policy that accepted would school every character until aging
		// killed him (p. 59).
		return 0
	case ChooseHobby, ChooseHomeworld, ChooseArt, ChooseTrade,
		ChooseService, ChooseMajor, ChooseMinor, ChooseSkill,
		ChooseAssociatedCareer, ChooseBenefitColumn:
		// POLICY.md: first-listed.
		return 0
	}

	return 0
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
