package chargen

// Noble career mechanics (Book 1 chart 11, p. 85; prose pp. 51, 65).
// "To Begin Automatic* (*if Soc B+) / Return & Intrigue C2 C3 C4 C5 /
// Elevation By Intrigue / Continue 7" (chart 11 box A).
//
// The Noble's Risk and Reward are Return and Intrigue, and which one is
// rolled depends on a state the career carries: "Return: Roll only if in
// Exile. Intrigue: Roll only if not in Exile." A failed Intrigue exiles
// the character; a failed Return continues the exile. A successful
// Intrigue earns an Elevation attempt, which is a Roll High: "roll Soc or
// greater to be Elevated to the next higher Noble rank and its increase in
// Social Standing (if any)" (p. 65).
//
// Rank is the Noble ladder itself: "Nobles begin with rank equal to their
// Social Standing" (p. 65). Consecutive rungs may share a Social Standing
// — a character elevated to Soc c is a Baronet, and the next elevation
// keeps Soc 12 while the title becomes Baron (p. 51) — which is what the
// chart's "(if any)" allows for.
//
// Deferred: Land Grant hexes and their economics, and the muster-out
// Proxy column (docs/PRD.md milestone 4); the Base Fame of 1.5 times Soc,
// which nothing in the career consumes (milestone 4 chart F); "A Noble may
// not pursue another career after this one" (career changes, milestone 4).

import (
	"fmt"

	"github.com/philoserf/t5chargen/career"
)

// nobleMechanics is the Noble careerMechanics implementation.
type nobleMechanics struct {
	// rank indexes the chart 11 ladder.
	rank int

	// fluxSpent records the once-per-career Elevation Flux.
	fluxSpent bool
}

// newNoble is the Noble careerRegistry entry.
//
//nolint:ireturn // The registry's function type returns the interface.
func newNoble() (*career.Definition, careerMechanics, error) {
	def, err := career.Noble()
	if err != nil {
		return nil, nil, fmt.Errorf("noble career: %w", err)
	}

	return def, &nobleMechanics{}, nil
}

// begin enters the career at the rung matching the character's Social
// Standing. There is no throw: chart 11 gives "To Begin Automatic* ...
// *if Soc B+" and the asterisk is a prerequisite, which p. 65 places
// before the attempt rather than in it — "Some Careers have requirements
// before a character may attempt to Begin" (interpretation I-28).
//
// The career is therefore only offered to a character who qualifies, and
// reaching here without the prerequisite is an engine fault rather than a
// failed entry, exactly as chart 01's automatic entry is handled.
func (m *nobleMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Noble: To Begin", r.def.Cite)

	if !meetsPrerequisite(r.character, r.def.BeginPrerequisite) {
		return false, fmt.Errorf("%w: %q entered without its prerequisite", errNotImplemented, r.def.Name)
	}

	soc := r.character.Characteristics.Soc

	// "Nobles begin with rank equal to their Social Standing" (p. 65), at
	// the first rung carrying it — a character beginning at Soc 12 is a
	// Baronet, the title the chart calls the initial one.
	m.rank = nobleRankFor(r.def, soc)
	m.setRank(r, r.entryCause)

	return true, nil
}

// nobleRankFor returns the index of the first ladder rung carrying soc, or
// the top rung when soc exceeds the ladder. The first is deliberate: "A
// character elevated to Soc = c (lower case) is initially a Baronet"
// (p. 51), so a Social Standing shared by two rungs enters at the lower
// title (interpretation I-29, ERRATA.md).
func nobleRankFor(def *career.Definition, soc int) int {
	best := 0

	for i, rank := range def.Ranks {
		if rank.Soc <= soc {
			best = i
		}

		if rank.Soc == soc {
			return i
		}
	}

	return best
}

// setRank records the current rung.
func (m *nobleMechanics) setRank(r *careerRun, cause int) {
	rank := r.def.Ranks[m.rank]

	r.record.Rank = rank.ID
	r.record.RankTitle = rank.Title

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceRankSet, Career: r.def.Name, Skill: rank.Title,
	})
}

// resolveTerm rolls Return or Intrigue as the exile state dictates, and on
// a successful Intrigue attempts an Elevation.
func (m *nobleMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	var outcome termOutcome

	outcome.skillRolls = r.def.SkillsPerTerm

	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return outcome, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	if r.record.Exiled {
		outcome.success = m.rollReturn(r, cc, value)

		return outcome, nil
	}

	if !m.rollIntrigue(r, cc, value) {
		return outcome, nil
	}

	outcome.success = true

	elevated, err := m.elevate(r)
	if err != nil {
		return outcome, err
	}

	if elevated {
		// "When Elevated 2" (chart 11 table B).
		outcome.skillRolls += r.def.SkillsPerAdvancement
	}

	return outcome, nil
}

// nobleMods is the chart's Mod, applied to Return as printed and to
// Intrigue with the opposite sign: "Mod= -Successful Intrigues. Mod= +Times
// Exiled." (chart 11; the p. 73 checklist lists both under one roll —
// interpretation I-27, ERRATA.md).
func nobleMods(record *CareerRecord) (int, []Mod) {
	mod := -record.SuccessfulIntrigues + record.TimesExiled

	var mods []Mod

	if record.SuccessfulIntrigues != 0 {
		mods = append(mods, Mod{Name: "successful Intrigues", Value: -record.SuccessfulIntrigues})
	}

	if record.TimesExiled != 0 {
		mods = append(mods, Mod{Name: "times Exiled", Value: record.TimesExiled})
	}

	return mod, mods
}

// negate flips the sign of each itemized modifier for the Intrigue roll.
func negate(mods []Mod) []Mod {
	out := make([]Mod, len(mods))
	for i, mod := range mods {
		out[i] = Mod{Name: mod.Name, Value: -mod.Value}
	}

	return out
}

// rollReturn rolls Return: "Roll only if in Exile ... Failure: Continued
// Exile; may not attempt Intrigue. Success: Return from Exile" (chart 11).
func (*nobleMechanics) rollReturn(r *careerRun, cc string, value int) bool {
	mod, mods := nobleMods(&r.record)

	throw := r.roller.Check(2, value+mod)
	seq := r.log.Throw(throw, mods, r.def.Cite+" (Return vs "+cc+"+Mods; roll only if in Exile)")

	if !throw.Success {
		// "Continued Exile; may not attempt Intrigue."
		return false
	}

	r.record.Exiled = false
	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceReturned, Career: r.def.Name})

	return true
}

// rollIntrigue rolls Intrigue: "Roll only if not in Exile ... Failure:
// Exile" (chart 11). A success earns the Elevation attempt.
func (*nobleMechanics) rollIntrigue(r *careerRun, cc string, value int) bool {
	mod, mods := nobleMods(&r.record)

	throw := r.roller.Check(2, value-mod)
	seq := r.log.Throw(throw, negate(mods),
		r.def.Cite+" (Intrigue vs "+cc+"+ opposite sign Mods; roll only if not in Exile)")

	if !throw.Success {
		r.record.Exiled = true
		r.record.TimesExiled++

		r.log.Consequence(ConsequenceEvent{
			Cause: seq, Kind: ConsequenceExiled, Career: r.def.Name, Value: r.record.TimesExiled,
		})

		return false
	}

	r.record.SuccessfulIntrigues++
	r.log.Consequence(ConsequenceEvent{
		Cause: seq, Kind: ConsequenceIntrigue, Career: r.def.Name, Value: r.record.SuccessfulIntrigues,
	})

	return true
}

// elevate attempts the Elevation: "Roll Soc or greater (no Mods but
// possibly Flux) to rise to the next higher Noble rank and its increase in
// Social Standing (if any)." (chart 11) This is a Roll High (p. 65).
func (m *nobleMechanics) elevate(r *careerRun) (bool, error) {
	if m.rank >= len(r.def.Ranks)-1 {
		// The top of the ladder: nothing to rise to.
		return false, nil
	}

	mod, err := m.offerFlux(r)
	if err != nil {
		return false, err
	}

	target := r.character.Characteristics.Soc

	throw := r.roller.High(2, target, mod)
	seq := r.log.Throw(throw, nil, r.def.Cite+" (Elevation: Roll High, Soc or greater)")

	if !throw.Success {
		return false, nil
	}

	m.rank++
	m.setRank(r, seq)
	m.raiseSoc(r, seq)

	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceElevated, Career: r.def.Name})

	return true, nil
}

// raiseSoc applies the Elevation's "increase in Social Standing (if any)":
// the rungs that share a Social Standing raise the title alone. Each
// increase that lands awards a Land Grant (chart 11).
func (m *nobleMechanics) raiseSoc(r *careerRun, cause int) {
	rank := r.def.Ranks[m.rank]
	if rank.Soc <= r.character.Characteristics.Soc {
		return
	}

	before := r.character.Characteristics.Soc
	awardCharacteristicAndLog(r.character, r.log, "Soc", 1, cause)

	if r.character.Characteristics.Soc == before {
		// The p. 68 maximum refused the increase; the title still rose.
		return
	}

	m.grantLand(r, cause)
}

// characteristicRaised is the careerRun's notification that a table C
// characteristic award landed. Chart 11 ties the Land Grant to the Social
// Standing and not to the title: "Land Grants. Each increase in Soc during
// CharGen awards a Land Grant." Column 1 line 6 ("C6 +1") raises a Noble's
// Soc mid-career, and that increase is awarded like Elevation's
// (interpretation I-30, ERRATA.md, which records the rank table's
// conflicting "Each noble title confers a Land Grant").
func (m *nobleMechanics) characteristicRaised(r *careerRun, name string, cause int) {
	if name != "Soc" {
		return
	}

	m.grantLand(r, cause)
}

// grantLand records one Land Grant. The hexes and their economics land
// with muster out (docs/PRD.md milestone 4).
func (*nobleMechanics) grantLand(r *careerRun, cause int) {
	r.record.LandGrants++

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceLandGrant, Career: r.def.Name, Value: r.record.LandGrants,
	})
}

// offerFlux offers the once-per-career Elevation Flux: "Once in the Noble
// Career after a successful Intrigue, invoke Flux as a Mod on Elevation
// roll" (chart 11).
func (m *nobleMechanics) offerFlux(r *careerRun) (int, error) {
	if m.fluxSpent {
		return 0, nil
	}

	// The elevation needs Soc or greater on 2D; above Soc 12 it is
	// impossible without the Flux, which is the decision aid the policy
	// reads (POLICY.md).
	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseElevationFlux,
		Prompt:  "Invoke the once-per-career Flux on this Elevation roll?",
		Options: []string{"Invoke Flux", "Save it"},
		Scores:  []int{r.character.Characteristics.Soc, 0},
		Cite:    r.def.Cite + " (Once in the Noble Career, invoke Flux as a Mod on Elevation)",
	})
	if err != nil || chosen != 0 {
		return 0, err
	}

	m.fluxSpent = true

	flux := r.roller.Flux()
	r.log.Flux(flux, r.def.Cite+" (Elevation Flux)")

	return flux.Value, nil
}
