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
	// prompt is shown to the decider; cite names the governing rule.
	prompt string
	cite   string
}

// educationWaiver is the p. 59 Educational Waiver.
//
// P. 59 names four waiver-able events: "Prerequisite, Application Check,
// Pass/Fail Check, Honors". v1 offers waivers for the Application and
// Pass/Fail checks — the two the process description integrates
// ("Failure terminates the process (but Waiver may result in
// reinstatement...)"). Prerequisite waivers are unreachable while
// chooseProgram offers only qualifying programs (letting an unqualified
// character attempt admission is an interactive-mode concern), and Honors
// waivers are deferred with them; both land with milestone 5.
func educationWaiver(reason string) waiverPrompt {
	return waiverPrompt{
		prompt: "Attempt an Educational Waiver? (" + reason + ")",
		cite:   "Book 1 p. 59 (Educational Waivers)",
	}
}

// offerWaiver offers a waiver and reports whether the adverse outcome is
// waived. A waiver negates the outcome rather than re-rolling it.
func offerWaiver(
	log *Log, decider Decider, roller *dice.Roller, character *Character, p waiverPrompt,
) (bool, error) {
	chosen, _, err := choose(log, decider, Choice{
		ID:      ChooseWaiver,
		Prompt:  p.prompt,
		Options: []string{"Attempt waiver", "Accept the result"},
		Cite:    p.cite,
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
