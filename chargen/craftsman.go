package chargen

// Craftsman career mechanics (Book 1 chart 01, p. 75).
//
// "The focus of a Craftsman's activity is creating Masterpieces within his
// craft. He does not roll Risk and Reward." Chart 01 is the only career
// whose term is neither a Risk & Reward cycle nor a variant of one.
//
// Box A: "To Begin Automatic* / Masterpiece C1 C2 C3 C4 / Continue
// Craftsman x2", with "*if TWO skill-6 and Craftsman-1".
//
// "Master Points. In each Term, the Craftsman totals available Master
// Points (which will be used toward the current Masterpiece). Roll 9D for
// Master Points or less for success in creation. If the Craftsman cannot
// show at least 40 Master Points, he cannot attempt a Masterpiece (treat
// as Failure)."
//
// Deferred: the QREBS allocation, which nothing in generation reads
// (interpretation I-62), and Vintage Masterpiece appreciation, which is
// play.

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/skill"
)

// craftsmanMechanics is the Craftsman careerMechanics implementation.
type craftsmanMechanics struct{}

// newCraftsman is the Craftsman careerRegistry entry.
//
//nolint:ireturn // The registry's function type returns the interface.
func newCraftsman() (*career.Definition, careerMechanics, error) {
	def, err := career.Craftsman()
	if err != nil {
		return nil, nil, fmt.Errorf("craftsman career: %w", err)
	}

	if def.Masterpiece == nil {
		return nil, nil, fmt.Errorf("%w: craftsman career has no Masterpiece box", errNotImplemented)
	}

	return def, &craftsmanMechanics{}, nil
}

// begin resolves "To Begin Automatic* — *if TWO skill-6 and Craftsman-1".
//
// There is no throw. Entry is automatic for a character who qualifies, and
// a character who does not is not offered the career at all rather than
// failing an attempt at it: p. 65's year-per-failed-attempt and its "may
// not be used" both speak of attempts, and there is none to make
// (interpretation I-60, ERRATA.md).
func (*craftsmanMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Craftsman: To Begin (Automatic)", r.def.Cite)

	// The career is only offered to a character who qualifies, so
	// reaching here without the prerequisite is an engine fault, not a
	// failed entry.
	if !meetsPrerequisite(r.character, r.def.BeginPrerequisite) {
		return false, fmt.Errorf("%w: %q entered without its prerequisite", errNotImplemented, r.def.Name)
	}

	return true, nil
}

// resolveTerm creates the term's Masterpiece (chart 01 Creating A
// Masterpiece).
func (m *craftsmanMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	var outcome termOutcome

	box := r.def.Masterpiece

	points, mods := m.masterPoints(r)

	created, cause := m.attempt(r, cc, points, mods)

	// "Per Success 3 +Craftsman-1 / Per Failure 1 +Craftsman-1" (table B).
	// The Craftsman level comes either way; only the eligibility differs.
	outcome.success = created
	outcome.bonusRolls = box.SkillsPerFailure

	if created {
		outcome.bonusRolls = box.SkillsPerSuccess
	}

	if err := r.awardAndLog("Craftsman", 1, cause); err != nil {
		return termOutcome{}, err
	}

	return outcome, nil
}

// masterPoints totals what the Craftsman brings to the current Masterpiece
// (chart 01 Creating A Masterpiece):
//
//	Controlling Characteristics
//	Craftsman Skill
//	Up to FIVE Skills at level 6+ (or Knowledges at level-6) (but not
//	languages)
//
// "Controlling Characteristics" is read as all four the chart names, not
// the one governing this term's Masterpiece — see interpretation I-61.
func (m *craftsmanMechanics) masterPoints(r *careerRun) (int, []Mod) {
	box := r.def.Masterpiece

	total, mods := 0, make([]Mod, 0, 3)

	for _, name := range r.def.ControllingCharacteristics {
		value, _ := characteristicValue(&r.character.Characteristics, name)
		total += value
	}

	mods = append(mods, Mod{Name: "Controlling Characteristics", Value: total})

	craft := r.character.skillLevel("Craftsman")
	total += craft
	mods = append(mods, Mod{Name: "Craftsman", Value: craft})

	bonus := m.bonusSkillPoints(r, box)
	total += bonus

	mods = append(mods, Mod{
		Name:  "Skills at " + strconv.Itoa(box.BonusSkillLevel) + "+",
		Value: bonus,
	})

	return total, mods
}

// bonusSkillPoints totals the best few skills the chart lets a Craftsman
// bring: "Up to FIVE Skills at level 6+ (or Knowledges at level-6) (but
// not languages)". Craftsman itself is counted on its own line, so it is
// not counted twice.
func (m *craftsmanMechanics) bonusSkillPoints(r *careerRun, box *career.Masterpiece) int {
	levels := make([]int, 0, len(r.character.Skills))

	for _, held := range r.character.Skills {
		if held.Level < box.BonusSkillLevel || held.Name == "Craftsman" {
			continue
		}

		// "(but not languages)": the Language skill and its Knowledges.
		if held.Name == box.ExcludedSkill {
			continue
		}

		if entry, ok := skill.Lookup(held.Name); ok && entry.Parent == box.ExcludedSkill {
			continue
		}

		levels = append(levels, held.Level)
	}

	slices.Sort(levels)
	slices.Reverse(levels)

	total := 0
	for _, level := range levels[:min(len(levels), box.MaxBonusSkills)] {
		total += level
	}

	return total
}

// attempt resolves the creation throw and records what it produced.
//
// "Roll 9D for Master Points or less for success in creation. If the
// Craftsman cannot show at least 40 Master Points, he cannot attempt a
// Masterpiece (treat as Failure)". It reports whether one was created and
// the event to hang the term's consequences from.
func (m *craftsmanMechanics) attempt(r *careerRun, cc string, points int, mods []Mod) (bool, int) {
	box := r.def.Masterpiece

	if points < box.MinimumPoints {
		// "he cannot attempt a Masterpiece (treat as Failure)": no throw
		// is made, so the step that established the shortfall is what
		// the consequences hang from (interpretation I-87, ERRATA.md).
		seq := r.log.Step("Craftsman: too few Master Points to attempt a Masterpiece", box.Cite)
		r.log.Consequence(ConsequenceEvent{
			Cause: seq, Kind: ConsequenceNoAward, Career: r.def.Name, Value: points,
		})

		return false, seq
	}

	throw := r.roller.Check(box.Dice, points)
	seq := r.log.Throw(throw, mods, box.Cite+" (Masterpiece vs Master Points, "+cc+" governing)")

	if !throw.Success {
		// "Creation Attempt Fails: Receive Craftsman +1 (considered
		// learning from experience)."
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceNoAward, Career: r.def.Name})

		return false, seq
	}

	perfect := points >= box.PerfectPoints
	r.record.Masterpieces++

	if perfect {
		r.record.PerfectMasterpieces++
	}

	r.log.Consequence(ConsequenceEvent{
		Cause: seq, Kind: ConsequenceMasterpiece, Career: r.def.Name,
		Value: points, Delta: masterpieceValue(box, points),
	})

	return true, seq
}

// masterpieceValue prices a Masterpiece: "The Masterpiece can be sold at
// Cr150,000 plus Cr10,000 per Master Point over 40. A Perfect Masterpiece
// (=55 points or more) sells for Double" (chart 01 p. 75). Spending it
// lands with muster out.
func masterpieceValue(box *career.Masterpiece, points int) int {
	value := box.BaseValue + box.ValuePerPoint*(points-box.MinimumPoints)
	if points >= box.PerfectPoints {
		value *= box.PerfectMultiplier
	}

	return value
}

// meetsPrerequisite reports whether a character already qualifies for a
// career whose entry is automatic but conditional: chart 01's "if TWO
// skill-6 and Craftsman-1" (p. 75) and chart 11's "if Soc B+" (p. 85).
//
// Checked before the career is offered, which is where p. 65 puts it:
// "Pre-Requisites. Some Careers have requirements before a character may
// attempt to Begin".
func meetsPrerequisite(c *Character, p *career.Prerequisite) bool {
	if p == nil {
		return true
	}

	if p.Characteristic != "" {
		value, ok := characteristicValue(&c.Characteristics, p.Characteristic)
		if !ok || value < p.Minimum {
			return false
		}
	}

	if c.skillLevel(p.Skill) < p.SkillLevel {
		return false
	}

	at := 0

	for _, held := range c.Skills {
		if held.Level >= p.SkillsAtLevel {
			at++
		}
	}

	return at >= p.SkillsAtLevelCount
}
