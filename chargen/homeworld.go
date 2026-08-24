package chargen

// Checklist step B: Determine A Homeworld (chart E1, p. 72; chart B,
// p. 56; prose p. 58). "A character receives one specified skill for each
// Trade Classification or Remark from the homeworld." (p. 58) Grants are
// level 1, per the p. 58 example ("Trader-1 (from Pa), Chef-1 (from Ri,
// selected Chef)").

import (
	"fmt"

	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/world"
)

// rollHomeworld determines the homeworld on chart B's world list:
// "Homeworld. Select or determine a Homeworld" (p. 56). The chart is
// indexed by two dice read in order, D1 then D2, so they are rolled and
// logged separately rather than summed — a 3 and a 4 is not a 4 and a 3.
func rollHomeworld(roller *dice.Roller, log *Log) (world.Homeworld, error) {
	cite := "Book 1 p. 56 chart B (Select a Homeworld: D1 D2)"

	d1 := roller.Roll(1)
	log.Roll(d1, cite)

	d2 := roller.Roll(1)
	log.Roll(d2, cite)

	homeworld, err := world.At(d1.Total, d2.Total)
	if err != nil {
		return world.Homeworld{}, fmt.Errorf("homeworld: %w", err)
	}

	return homeworld, nil
}

// runHomeworld performs checklist step B: validates and records the
// homeworld, then grants its trade-classification skills. The homeworld
// arrives assigned (docs/PRD.md FR2: a supplied UWP or the tool-owned
// default); the assignment is logged as a choice event so the skill
// consequences have a cause and interactive selection has its seam.
func runHomeworld(
	homeworld world.Homeworld, roll bool, roller *dice.Roller,
	log *Log, decider Decider, character *Character,
) error {
	log.Step("Determine A Homeworld", "Book 1 p. 72 chart E1 step B")

	if roll {
		rolled, err := rollHomeworld(roller, log)
		if err != nil {
			return err
		}

		homeworld = rolled
	}

	if err := homeworld.Validate(); err != nil {
		return fmt.Errorf("homeworld: %w", err)
	}

	_, seq, err := choose(log, decider, Choice{
		ID:      ChooseHomeworld,
		Prompt:  "Select a homeworld",
		Options: []string{homeworld.Label()},
		Cite:    "Book 1 p. 58 (as assigned, selected, or random); chart B p. 56",
	})
	if err != nil {
		return err
	}

	character.Homeworld = homeworld

	for _, tc := range homeworld.TradeClassifications {
		if err := grantTC(tc, seq, log, decider, character); err != nil {
			return err
		}
	}

	return nil
}

// grantTC applies one trade classification's chart B grant.
func grantTC(tc string, cause int, log *Log, decider Decider, character *Character) error {
	grant, err := world.GrantFor(tc)
	if err != nil {
		return fmt.Errorf("homeworld: %w", err)
	}

	switch grant.Kind {
	case world.GrantSkill:
		for _, name := range grant.Skills {
			awardSkillAndLog(name, 1, cause, log, character)
		}
	case world.GrantArt, world.GrantTrade:
		return grantSelection(grant, log, decider, character)
	case world.GrantNone: // "(no skill)" rows grant nothing; the TC list is on the homeworld record
	}

	return nil
}

// grantSelection resolves the Ri "One Art (Choose One)" and In "The Trades
// (Choose One)" grants (p. 56); the award is caused by the selecting
// choice event.
func grantSelection(grant world.Grant, log *Log, decider Decider, character *Character) error {
	id := ChooseArt
	prompt := "Choose one Art"
	options, err := world.ArtChoices()

	if grant.Kind == world.GrantTrade {
		id = ChooseTrade
		prompt = "Choose one Trade"
		options, err = world.TradeChoices()
	}

	if err != nil {
		return fmt.Errorf("homeworld: %w", err)
	}

	chosen, seq, err := choose(log, decider, Choice{
		ID:      id,
		Prompt:  prompt + " (TC " + grant.TC + ")",
		Options: options,
		Cite:    "Book 1 p. 56 chart B (" + grant.Label + ")",
	})
	if err != nil {
		return err
	}

	awardSkillAndLog(options[chosen], 1, seq, log, character)

	return nil
}

// awardSkillAndLog awards skill levels (capped at SkillMax, p. 134) and
// emits the matching consequence: skill_awarded, or no_award if the cap
// absorbed the whole receipt. Career-independent; the citizen runner
// delegates here.
func awardSkillAndLog(name string, levels, cause int, log *Log, character *Character) {
	level, applied := character.awardSkill(name, levels)
	if applied == 0 {
		log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceNoAward, Skill: name})

		return
	}

	log.Consequence(ConsequenceEvent{
		Cause: cause,
		Kind:  ConsequenceSkillAwarded,
		Skill: name,
		Delta: applied,
		Value: level,
	})
}

// homeworldOrDefault substitutes the tool-owned default homeworld
// (docs/PRD.md FR2) only when no homeworld at all was supplied — the
// all-zero struct. A partially-populated homeworld (TCs or a name without
// a UWP) falls through to validation and is rejected, never silently
// repaired (FR2).
//
// The deep space mark counts as populated: it is a claim about the world
// (p. 56, interpretation I-97), so a homeworld carrying nothing but the
// mark is a partial deep space birth and is rejected by validateDeepSpace,
// not quietly turned into Regina.
func homeworldOrDefault(homeworld world.Homeworld) (world.Homeworld, error) {
	if homeworld.UWP == "" && homeworld.Name == "" &&
		len(homeworld.TradeClassifications) == 0 && !homeworld.DeepSpace {
		d, err := world.Default()
		if err != nil {
			return world.Homeworld{}, fmt.Errorf("homeworld: %w", err)
		}

		return d, nil
	}

	return homeworld, nil
}
