package chargen

// Checklist step B: Determine A Homeworld (chart E1, p. 72; chart B,
// p. 56; prose p. 58). "A character receives one specified skill for each
// Trade Classification or Remark from the homeworld." (p. 58) Grants are
// level 1, per the p. 58 example ("Trader-1 (from Pa), Chef-1 (from Ri,
// selected Chef)").

import (
	"fmt"

	"github.com/philoserf/t5chargen/skill"

	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/world"
)

// homeworldOptions is what step B offers. "As assigned, selected, or
// random" (p. 58): assigning and rolling both settle the world before the
// choice, and there is nothing left to decide — the choice is logged all
// the same, so the skills it grants have a cause and a record says who
// picked.
//
// Where the world was neither assigned nor rolled, the choice is a real
// one and the options are chart B's own list. The worlds are returned
// beside the labels because a chosen index has to name a world again.
func homeworldOptions(homeworld world.Homeworld, assigned bool) ([]string, []world.Homeworld, error) {
	if assigned {
		return []string{homeworld.Label()}, nil, nil
	}

	cells, err := world.Selectable()
	if err != nil {
		return nil, nil, fmt.Errorf("homeworld: %w", err)
	}

	options := make([]string, 0, len(cells))
	worlds := make([]world.Homeworld, 0, len(cells))

	for _, cell := range cells {
		w := cell.Homeworld()
		options = append(options, w.Label())
		worlds = append(worlds, w)
	}

	return options, worlds, nil
}

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
	homeworld world.Homeworld, assigned, roll bool, roller *dice.Roller,
	log *Log, decider Decider, character *Character,
) error {
	log.Step("Determine A Homeworld", "Book 1 p. 72 chart E1 step B")

	if roll {
		rolled, err := rollHomeworld(roller, log)
		if err != nil {
			return err
		}

		homeworld = rolled
		assigned = true
	}

	if err := homeworld.Validate(); err != nil {
		return fmt.Errorf("homeworld: %w", err)
	}

	options, worlds, err := homeworldOptions(homeworld, assigned)
	if err != nil {
		return err
	}

	chosen, seq, err := choose(log, decider, Choice{
		ID:      ChooseHomeworld,
		Prompt:  "Select a homeworld",
		Options: options,
		Cite:    "Book 1 p. 58 (as assigned, selected, or random); chart B p. 56",
	})
	if err != nil {
		return err
	}

	if len(worlds) > 0 {
		homeworld = worlds[chosen]
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
			if err := awardSkillAndLog(name, 1, cause, log, decider, character); err != nil {
				return err
			}
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

	if err := awardSkillAndLog(options[chosen], 1, seq, log, decider, character); err != nil {
		return err
	}

	return nil
}

// awardSkillAndLog awards skill levels (capped at SkillMax, p. 134) and
// emits the matching consequence: skill_awarded, or no_award if the cap
// absorbed the whole receipt. Career-independent; the citizen runner
// awardSkillAndLog awards skill levels and records the consequence. It is
// the one funnel every award goes through, which is what lets p. 134's
// Knowledge-Knowledge-Skill sequence apply everywhere a container skill
// is received without each of the thirty award sites knowing about it.
func awardSkillAndLog(name string, levels, cause int, log *Log, decider Decider, character *Character) error {
	handled, err := awardContainer(name, levels, cause, log, decider, character)
	if handled {
		return err
	}

	awardSkillLevels(name, levels, cause, log, character)

	return nil
}

// awardSkillLevels awards levels of a named skill or knowledge and
// records what landed.
//
// The cap follows the name rather than the caller: "Skill, Knowledge, and
// Talent Maximums: Skill-15" and "The maximum level of a Knowledge is 6"
// are both p. 134, and a Knowledge is capped at 6 however it was
// received. Several career tables award one by name — chart 07's
// Starship cells reach Bay Weapons, chart 08's reach Exotics — and
// capping by call site let those past 6.
func awardSkillLevels(name string, levels, cause int, log *Log, character *Character) {
	level, applied := character.awardSkill(name, levels, levelCap(name))
	logAward(name, level, applied, cause, log)
}

// levelCap is the maximum a name may reach: 6 for a Knowledge, 15 for
// everything else (p. 134).
func levelCap(name string) int {
	if entry, ok := skill.Lookup(name); ok && entry.Kind == skill.KindKnowledge {
		return KnowledgeMax
	}

	return SkillMax
}

// logAward records an award, or records that the cap refused it.
func logAward(name string, level, applied, cause int, log *Log) {
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
// supplied reports whether a caller named a homeworld at all. The zero
// value means it did not, which is what leaves the choice to the
// character.
func supplied(homeworld world.Homeworld) bool {
	return homeworld.UWP != "" || homeworld.Name != "" ||
		len(homeworld.TradeClassifications) > 0 || homeworld.DeepSpace
}

func homeworldOrDefault(homeworld world.Homeworld) (world.Homeworld, error) {
	if !supplied(homeworld) {
		d, err := world.Default()
		if err != nil {
			return world.Homeworld{}, fmt.Errorf("homeworld: %w", err)
		}

		return d, nil
	}

	return homeworld, nil
}
