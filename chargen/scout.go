package chargen

// Scout career mechanics (Book 1 chart 05, p. 79; prose pp. 64-65).
// "To Begin C1 or C2 or C3; Risk & Reward C1 C2 C3; Retry R&R C5;
// Continue Int" (chart 05 box A). Scouts have no rank (p. 65).
//
// Per term the Scout chooses a duty: "A Scout may avoid the Risk and
// Reward rolls by volunteering for Courier Duty" (p. 79), with skill
// eligibility "If Courier Duty 4 / If Explorer Duty 8" (chart 05 box B).
// Explorer Duty runs Risk & Reward: "Select Caution, Bravery, or No Mod.
// Roll for Risk against CC+ Mods. ... Roll for Reward against CC+
// (opposite sign) Mods." (chart 05 box)
//
// Sanity decay — "reduce San= -1 for each TWO Terms served" — is recorded
// rather than applied, because no Sanity value is generated during
// character generation (p. 52): careerrun.go's recordSanityMod, driven by
// the career's sanity_per_terms, interpretation I-47 in ERRATA.md.
//
// Deferred: Land Grant values and the Discovery grant economics (chart 05,
// milestone 4 muster out); muster-out table D.

import (
	"fmt"

	"github.com/philoserf/t5chargen/career"
)

// scoutDuties are the chart 05 box B duties in chart order.
var scoutDuties = []string{"Courier Duty", "Explorer Duty"}

// scoutMechanics is the Scout careerMechanics implementation.
type scoutMechanics struct{}

// newScout is the Scout careerRegistry entry.
//
//nolint:ireturn // Registry constructors return the careerMechanics seam by design.
func newScout() (*career.Definition, careerMechanics, error) {
	def, err := career.Scout()
	if err != nil {
		return nil, nil, fmt.Errorf("scout career: %w", err)
	}

	return def, &scoutMechanics{}, nil
}

// begin rolls To Begin: "To Begin C1 or C2 or C3" (chart 05). Scout's box
// lists no Begin retry; a failed attempt costs one year ("Each failed
// attempt (both Begin or Retry) takes one year", p. 65).
func (*scoutMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Scout: To Begin", "Book 1 p. 79 chart 05 (To Begin C1 or C2 or C3)")

	name, target, err := chooseCheckCharacteristic(r, r.def.BeginChecks)
	if err != nil {
		return false, err
	}

	throw := r.roller.Check(2, target)
	seq := r.log.Throw(throw, nil, "Book 1 p. 79 chart 05 (To Begin vs "+name+")")

	if throw.Success {
		return true, nil
	}

	if err := r.character.advanceYears(1, r.roller, r.log, seq); err != nil {
		return false, err
	}

	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceCareerNotBegun, Career: r.def.Name})

	return false, nil
}

// resolveTerm chooses the term's duty and, for Explorer Duty, runs Risk &
// Reward.
func (m *scoutMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseDuty,
		Prompt:  "Select the term's duty",
		Options: scoutDuties,
		Cite:    "Book 1 p. 79 (Courier Duty avoids Risk and Reward; chart 05 table B)",
	})
	if err != nil {
		return termOutcome{}, err
	}

	duty := scoutDuties[chosen]
	outcome := termOutcome{skillRolls: r.def.SkillEligibility[duty]}

	if duty == "Courier Duty" {
		// "A Scout may avoid the Risk and Reward rolls by volunteering
		// for Courier Duty." (p. 79)
		return outcome, nil
	}

	return m.riskAndReward(r, cc, outcome)
}

// riskAndReward runs the chart 05 Risk & Reward box.
func (m *scoutMechanics) riskAndReward(r *careerRun, cc string, outcome termOutcome) (termOutcome, error) {
	mod, err := chooseRiskMod(r, "chart 05 p. 79")
	if err != nil {
		return termOutcome{}, err
	}

	ccValue, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return termOutcome{}, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	risk := r.roller.Check(2, ccValue+mod)
	riskSeq := r.log.Throw(risk, riskMods(mod, 1), "Book 1 p. 79 chart 05 (Risk vs "+cc+"+Mods)")

	if !risk.Success {
		died, disabled := r.injury(cc, mod, riskSeq,
			"Book 1 p. 79 chart 05 (Risk Failure: reduce CC by negative Mods and Flux)")
		outcome.endCause = riskSeq

		if died {
			outcome.died = true

			return outcome, nil
		}

		outcome.endCareer = disabled
	}

	success, err := m.reward(r, cc, ccValue, mod)
	if err != nil {
		return termOutcome{}, err
	}

	outcome.success = success

	return outcome, nil
}

// reward rolls Reward vs CC with opposite-sign mods; a failure may be
// retried once against C5 ("Retry R&R C5", chart 05; interpretation I-8,
// ERRATA.md). Success is a Discovery with Fame +1 (chart 05).
func (m *scoutMechanics) reward(r *careerRun, cc string, ccValue, mod int) (bool, error) {
	throw := r.roller.Check(2, ccValue-mod)
	seq := r.log.Throw(throw, riskMods(mod, -1), "Book 1 p. 79 chart 05 (Reward vs "+cc+"+ opposite sign Mods)")

	if !throw.Success {
		retried, retrySeq, err := m.retryReward(r, mod)
		if err != nil || !retried {
			return false, err
		}

		seq = retrySeq
		throw.Success = true
	}

	m.discovery(r, seq)

	return true, nil
}

// retryReward offers the I-8 Reward retry against the RetryCheck
// characteristic. Reports whether a retry succeeded.
func (*scoutMechanics) retryReward(r *careerRun, mod int) (bool, int, error) {
	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseRetry,
		Prompt:  "Retry the Reward against " + r.def.RetryCheck + "?",
		Options: []string{"Retry", "Accept the failure"},
		Cite:    "Book 1 p. 79 chart 05 (Retry R&R C5); interpretation I-8, ERRATA.md",
	})
	if err != nil || chosen == 1 {
		return false, 0, err
	}

	value, _ := characteristicValue(&r.character.Characteristics, r.def.RetryCheck)
	throw := r.roller.Check(2, value-mod)
	seq := r.log.Throw(throw, riskMods(mod, -1),
		"Book 1 p. 79 chart 05 (Retry R&R vs "+r.def.RetryCheck+"; interpretation I-8)")

	return throw.Success, seq, nil
}

// discovery records a Reward success: "The Scout discovers a valuable new
// world or a valuable feature on a known world (a Discovery), receives a
// Land Grant, and Fame +1." (chart 05)
//
// The grant is recorded as a grant, one per Discovery, rather than left for
// muster out to infer from the Discovery count: p. 79 gives a Scout's grant
// a different shape from a Noble's — "one World Hex on a non-Mainworld" — so
// the two have to be counted in the same currency to be priced at all.
func (*scoutMechanics) discovery(r *careerRun, cause int) {
	r.record.Discoveries++
	r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceDiscovery, Value: r.record.Discoveries})

	r.record.LandGrants++
	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceLandGrant, Career: r.def.Name, Value: r.record.LandGrants,
	})
}
