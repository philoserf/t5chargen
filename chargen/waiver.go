package chargen

import "github.com/philoserf/t5chargen/dice"

// Waivers: "An adverse die roll or decision ... may be waived. Check Soc
// (2D); Mod minus previous waivers (successful or not)." (Book 1 p. 76
// chart 02; the same rule as the Educational Waiver, p. 59.)
//
// The two are one pool. Neither text says "educational waivers" when
// counting the previous ones, so a waiver spent at university makes the
// next one harder in a career and vice versa (interpretation I-22,
// ERRATA.md).

// waiverPrompt names the waiver-able event in the choice event.
type waiverPrompt struct {
	// id distinguishes the education and career waivers, which share a
	// pool but not a policy; prompt is shown to the decider and cite names
	// the governing rule.
	id     ChoiceID
	prompt string
	cite   string

	// careerEnding marks the outcomes whose un-waived branch ends the
	// career, which is what the auto policy weighs (POLICY.md).
	careerEnding bool

	// statusOnly marks an education waiver whose un-waived branch ends
	// nothing: Honors failure "has no effect" (p. 59), so declining costs
	// only the status. Carried on the Choice like careerEnding, so the
	// policy weighs the stake without reading the prompt text.
	statusOnly bool
}

// atStake reports whether refusing the waiver ends something. A career
// waiver is at stake when the un-waived branch ends the career (chart 02);
// an education waiver is at stake unless it is the Honors one, whose
// failure "has no effect" (p. 59, interpretation I-96).
func (p waiverPrompt) atStake() bool {
	if p.id == ChooseCareerWaiver {
		return p.careerEnding
	}

	return !p.statusOnly
}

// educationWaiver is the p. 59 Educational Waiver.
//
// P. 59 names four waiver-able events — "Prerequisite, Application Check,
// Pass/Fail Check, Honors" — and all four are offered. The Prerequisite is
// a decision rather than a roll: the character does not meet the row's
// minimum and is turned away before he can apply, which is the first thing
// the sentence lists. Honors is the other direction: its failure "has no
// effect" (p. 59), so the waiver buys the status the roll denied rather
// than reinstating a process (interpretation I-96).
func educationWaiver(reason string) waiverPrompt {
	return waiverPrompt{
		id:     ChooseWaiver,
		prompt: "Attempt an Educational Waiver? (" + reason + ")",
		cite:   "Book 1 p. 59 (Educational Waivers)",
	}
}

// honorsWaiverPrompt is the one educational waiver whose refusal ends
// nothing (p. 59: Honors "Failure has no effect").
func honorsWaiverPrompt() waiverPrompt {
	p := educationWaiver("Honors refused")
	p.statusOnly = true

	return p
}

// careerWaiver names a career-chart waiver-able event; chartCite is the
// chart printing the Waivers box (chart 02 p. 76 is the only one in v1).
// careerEnding marks the outcomes that would end the career, which the
// auto policy weighs through the Choice's stake Score rather than by reading
// the prompt text.
func careerWaiver(reason, chartCite string, careerEnding bool) waiverPrompt {
	return waiverPrompt{
		id:           ChooseCareerWaiver,
		prompt:       "Attempt a Waiver? (" + reason + ")",
		cite:         chartCite + " (Waivers: Check Soc, Mod minus previous waivers)",
		careerEnding: careerEnding,
	}
}

// offerWaiver offers a waiver and reports whether the adverse outcome is
// waived. A waiver negates the outcome rather than re-rolling it.
func offerWaiver(
	log *Log, decider Decider, roller *dice.Roller, character *Character, p waiverPrompt,
) (bool, error) {
	chosen, _, err := choose(log, decider, Choice{
		ID:      p.id,
		Prompt:  p.prompt,
		Options: []string{"Attempt waiver", "Accept the result"},
		// What refusing costs: 1 where it ends the career, the process or
		// the admission, 0 where it costs only a status. Both waiver
		// rules turn on this one number (POLICY.md), which is why the
		// stake is carried here rather than read off the prompt.
		Scores:     []int{boolScore(p.atStake()), 0},
		ScoreLabel: "at stake",
		Cite:       p.cite,
	})
	if err != nil || chosen == 1 {
		return false, err
	}

	previous := character.WaiversAttempted
	character.WaiversAttempted++

	target := character.Characteristics.Soc - previous
	throw := roller.Check(2, target)
	log.Throw(throw, []Mod{{Name: "previous waivers", Value: -previous}},
		"Book 1 p. 59 (Waiver: Check Soc, Mod minus previous waivers)")

	return throw.Success, nil
}
