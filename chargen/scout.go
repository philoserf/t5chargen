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
// Deferred: Land Grant values and the Discovery grant economics (chart 05,
// milestone 4 muster out); Sanity decay ("reduce San= -1 for each TWO
// Terms served" — San is deferred with chart A); muster-out table D.

import (
	"fmt"
	"strconv"

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

	throw := r.roller.Throw(2, target)
	seq := r.log.Throw(throw, nil, "Book 1 p. 79 chart 05 (To Begin vs "+name+")")

	if throw.Success {
		return true, nil
	}

	r.character.Age++
	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceYearsElapsed, Value: 1})
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
	mod, err := chooseRiskMod(r)
	if err != nil {
		return termOutcome{}, err
	}

	ccValue, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return termOutcome{}, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	risk := r.roller.Throw(2, ccValue+mod)
	riskSeq := r.log.Throw(risk, riskMods(mod, 1), "Book 1 p. 79 chart 05 (Risk vs "+cc+"+Mods)")

	if !risk.Success {
		died, disabled := m.injury(r, cc, mod, riskSeq)
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

// injury resolves a Risk failure: "Risk Failure: Reduce CC by negative
// Mods and Flux (CC may not be increased). If CC is reduced by 4 or more,
// then he is disabled. Muster Out at Term end with Double Benefits."
// (chart 05, p. 79; wound badge, disabled, and dead per p. 65). Reports
// died, then disabled.
func (*scoutMechanics) injury(r *careerRun, cc string, mod, cause int) (bool, bool) {
	flux := r.roller.Flux()
	r.log.Flux(flux, "Book 1 p. 79 chart 05 (Risk Failure: reduce CC by negative Mods and Flux)")

	delta := min(mod, 0) + flux.Value
	if delta >= 0 {
		// "CC may not be increased" (chart 05); an unreduced CC leaves
		// the Scout unharmed (p. 65).
		return false, false
	}

	value := characteristicAdd(&r.character.Characteristics, cc, delta)
	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceCharacteristicChange,
		Characteristic: cc, Delta: delta, Value: value,
	})

	// "If the Controlling Characteristic is reduced to zero or less, the
	// Character is dead." (p. 65)
	if value <= 0 {
		r.character.Dead = true
		r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceDead, Characteristic: cc})

		return true, false
	}

	r.character.WoundBadges++
	r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceWoundBadge, Value: r.character.WoundBadges})

	if -delta >= 4 {
		r.character.Disabled = true
		r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceDisabled, Characteristic: cc})

		return false, true
	}

	return false, false
}

// reward rolls Reward vs CC with opposite-sign mods; a failure may be
// retried once against C5 ("Retry R&R C5", chart 05; interpretation I-8,
// ERRATA.md). Success is a Discovery with Fame +1 (chart 05).
func (m *scoutMechanics) reward(r *careerRun, cc string, ccValue, mod int) (bool, error) {
	throw := r.roller.Throw(2, ccValue-mod)
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
	throw := r.roller.Throw(2, value-mod)
	seq := r.log.Throw(throw, riskMods(mod, -1),
		"Book 1 p. 79 chart 05 (Retry R&R vs "+r.def.RetryCheck+"; interpretation I-8)")

	return throw.Success, seq, nil
}

// discovery records a Reward success: "The Scout discovers a valuable new
// world or a valuable feature on a known world (a Discovery), receives a
// Land Grant, and Fame +1." (chart 05) Land Grant values are milestone-4
// muster-out material.
func (*scoutMechanics) discovery(r *careerRun, cause int) {
	r.record.Discoveries++
	r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceDiscovery, Value: r.record.Discoveries})

	r.character.Fame++
	r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceFameChange, Delta: 1, Value: r.character.Fame})
}

// chooseCheckCharacteristic presents a check's stated characteristics
// (score-guided, like the education checks) and returns the chosen name
// and roll-low target.
func chooseCheckCharacteristic(r *careerRun, names []string) (string, int, error) {
	name := names[0]

	if len(names) > 1 {
		scores := make([]int, len(names))
		for i, n := range names {
			scores[i], _ = characteristicValue(&r.character.Characteristics, n)
		}

		chosen, _, err := choose(r.log, r.decider, Choice{
			ID:      ChooseCheck,
			Prompt:  "Select the characteristic to check",
			Options: names,
			Scores:  scores,
			Cite:    "Book 1 p. 59 (Check one of the stated Characteristics)",
		})
		if err != nil {
			return "", 0, err
		}

		name = names[chosen]
	}

	value, ok := characteristicValue(&r.character.Characteristics, name)
	if !ok {
		return "", 0, fmt.Errorf("%w: %q", errUnknownCharacteristic, name)
	}

	return name, value, nil
}

// riskModOptions are the chart 05 mod alternatives: "Select Caution,
// Bravery, or No Mod" with "any Cautious Mod +1 through +9 or any Bravery
// Mod -1 to -9" (p. 65).
var riskModOptions = buildRiskModOptions()

// riskModValues parallel riskModOptions.
var riskModValues = buildRiskModValues()

func buildRiskModOptions() []string {
	options := []string{"No Mod"}
	for i := 1; i <= 9; i++ {
		options = append(options, "Caution +"+strconv.Itoa(i))
	}

	for i := 1; i <= 9; i++ {
		options = append(options, "Bravery -"+strconv.Itoa(i))
	}

	return options
}

func buildRiskModValues() []int {
	values := []int{0}
	for i := 1; i <= 9; i++ {
		values = append(values, i)
	}

	for i := 1; i <= 9; i++ {
		values = append(values, -i)
	}

	return values
}

// chooseRiskMod resolves the Caution/Bravery/No Mod selection (p. 65;
// chart 05).
func chooseRiskMod(r *careerRun) (int, error) {
	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseRiskMod,
		Prompt:  "Select Caution, Bravery, or No Mod",
		Options: riskModOptions,
		Cite:    "Book 1 p. 65 (Caution, Bravery, or No Mod); chart 05 p. 79",
	})
	if err != nil {
		return 0, err
	}

	return riskModValues[chosen], nil
}

// riskMods itemizes a non-zero Caution/Bravery mod for a throw event; sign
// flips the mod for the Reward roll ("applied with an opposite sign to the
// Reward roll", p. 65).
func riskMods(mod, sign int) []Mod {
	if mod == 0 {
		return nil
	}

	name := "Caution"
	if mod < 0 {
		name = "Bravery"
	}

	return []Mod{{Name: name, Value: mod * sign}}
}
