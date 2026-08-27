package chargen

// Rogue career mechanics (Book 1 chart 10, p. 84; prose pp. 64-66).
// "To Begin CC / Risk & Reward CC* / Continue CC*" with "*Mod +Terms. But,
// 12 is always automatic failure." (chart 10 box A)
//
// "A Rogue selects one Controlling Characteristic (C1 C2 C3 C4 C5 C6)
// which is then used throughout his career (not just in the current
// Term)." Every other career rotates its controlling characteristic; the
// Rogue picks once, and that pick is also his To Begin and Continue
// target.
//
// "A Rogue focuses on a Scheme: a plan to amass at the expense of others.
// In each Term, the Rogue masterminds a Scheme (from the Rogue Schemes
// table) and consults Risk & Reward."
//
// A failed Risk sends him to prison rather than the infirmary: chart 10's
// Risk row prints no characteristic reduction, so a Rogue takes no wound
// badge and cannot die at his own trade (interpretation I-41, ERRATA.md).
//
// Deferred: spending the Payoff, and the muster-out table D (docs/PRD.md
// milestone 4).

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/philoserf/t5chargen/career"
)

// The chart 10 eligibilities and limits.
const (
	rogueFailedScheme     = 1
	rogueSuccessfulScheme = 4
	roguePrisonSkills     = 2
	roguePrisonMaxYears   = 4

	// roguePrisonColumns is the chart's restriction: "In Prison: Prison
	// Skills from the Rogue Skills table column 1 or 2 only".
	roguePrisonColumns = 2

	// rogueRollTheScheme is the option that declines the chart's
	// alternative and rolls Flux as usual. Last in the list, so the
	// policy can name it by position.
	rogueRollTheScheme = "Roll for it"

	// rogueSchemeSelected marks a Scheme taken rather than rolled, so the
	// transcript does not print a Flux that was never thrown.
	rogueSchemeSelected = "selected, not rolled"
)

// rogueMechanics is the Rogue careerMechanics implementation.
type rogueMechanics struct{}

// newRogue is the Rogue careerRegistry entry.
//
//nolint:ireturn // The registry's function type returns the interface.
func newRogue() (*career.Definition, careerMechanics, error) {
	def, err := career.Rogue()
	if err != nil {
		return nil, nil, fmt.Errorf("rogue career: %w", err)
	}

	if def.Schemes == nil {
		return nil, nil, fmt.Errorf("%w: rogue career has no schemes table", errNotImplemented)
	}

	// prisonTerm takes the first two columns unconditionally: "Prison
	// Skills from the Rogue Skills table column 1 or 2 only" (chart 10 B).
	if len(def.SkillColumns) < roguePrisonColumns {
		return nil, nil, fmt.Errorf("%w: rogue career has fewer than %d skill columns",
			errNotImplemented, roguePrisonColumns)
	}

	return def, &rogueMechanics{}, nil
}

// begin selects the career-long controlling characteristic and rolls To
// Begin against it. The selection comes first because the check targets it
// (chart 10 box A; interpretation I-42, ERRATA.md).
func (*rogueMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Rogue: To Begin", r.def.Cite)

	cc, err := r.chooseCC()
	if err != nil {
		return false, err
	}

	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return false, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	throw := r.roller.Check(2, value)
	seq := r.log.Throw(throw, nil, r.def.Cite+" (To Begin vs "+cc+")")

	if throw.Success {
		return true, nil
	}

	return failedToBegin(r, seq)
}

// resolveTerm runs a prison term where one is owed, and otherwise the
// term's Scheme.
func (m *rogueMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	if r.record.PrisonYears > 0 {
		return m.prisonTerm(r), nil
	}

	return m.schemeTerm(r, cc)
}

// prisonTerm serves a sentence: "In Prison: Prison Skills from the Rogue
// Skills table column 1 or 2 only. Receives ONLY Prison Skills (not Term
// or Scheme Skills)." (chart 10 table B) One term serves any sentence, the
// chart's maximum being four years (interpretation I-43, ERRATA.md).
//
// The sentence was set by a Risk failure in an earlier term, so there is
// no throw here for the imprisonment to hang from; it names this step
// instead (interpretation I-87, ERRATA.md).
func (*rogueMechanics) prisonTerm(r *careerRun) termOutcome {
	served := r.record.PrisonYears
	r.record.PrisonYears = 0

	seq := r.log.Step("Rogue: In Prison", r.def.Cite)
	r.log.Consequence(ConsequenceEvent{
		Cause: seq, Kind: ConsequenceImprisoned, Career: r.def.Name, Value: served,
	})

	columns := make([]string, 0, roguePrisonColumns)
	for _, column := range r.def.SkillColumns[:roguePrisonColumns] {
		columns = append(columns, column.Name)
	}

	return termOutcome{skillRolls: roguePrisonSkills, termColumns: columns}
}

// schemeTerm masterminds a Scheme and consults Risk & Reward.
func (m *rogueMechanics) schemeTerm(r *careerRun, cc string) (termOutcome, error) {
	var outcome termOutcome

	scheme, err := m.scheme(r)
	if err != nil {
		return outcome, err
	}

	caution, err := chooseRiskMod(r, r.def.Cite)
	if err != nil {
		return outcome, err
	}

	// "Select Caution, Bravery, or No Mod and Mod+Terms" (chart 10). The
	// Caution/Bravery selection is kept apart from the Terms mod: only the
	// negative ones sentence a caught Rogue (see imprison).
	terms := len(r.record.Terms)
	mod := caution + terms

	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return outcome, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	caught := false

	risk := r.roller.Check(2, value+mod)
	riskSeq := r.log.Throw(risk, rogueMods(caution, terms, 1), r.def.Cite+" (Risk vs "+cc+"+Mods)")

	if !risk.Success {
		caught = true

		m.imprison(r, caution, riskSeq)
	}

	reward := r.roller.Check(2, value-mod)
	rewardSeq := r.log.Throw(reward, rogueMods(caution, terms, -1),
		r.def.Cite+" (Reward vs "+cc+"+ opposite sign Mods)")

	outcome.skillRolls = r.def.SkillsPerTerm + rogueFailedScheme

	if !reward.Success {
		// Chart 10 table B calls this a Failed Scheme, which is what
		// chart F prices at x3 (p. 91).
		r.record.FailedSchemes++
		r.log.Consequence(ConsequenceEvent{Cause: rewardSeq, Kind: ConsequenceNoAward})

		return outcome, nil
	}

	r.record.SuccessfulSchemes++
	outcome.success = true
	outcome.skillRolls = r.def.SkillsPerTerm + rogueSuccessfulScheme

	m.payoff(r, scheme, value, reward.Total, -mod, caught, rewardSeq)

	return outcome, nil
}

// rogueMods itemizes the two modifiers chart 10 adds to both rolls
// separately, so the log records the Caution or Bravery the character
// selected even where the Terms mod cancels it (docs/PRD.md FR10).
func rogueMods(caution, terms, sign int) []Mod {
	mods := riskMods(caution, sign)
	if terms != 0 {
		mods = append(mods, Mod{Name: "Terms", Value: terms * sign})
	}

	return mods
}

// scheme rolls the term's Scheme and offers the chart's adjustment: "Flux
// may be modified (after roll) plus or minus 1" (chart 10).
func (m *rogueMechanics) scheme(r *careerRun) (career.SchemeRow, error) {
	if scheme, taken, err := m.selectScheme(r); err != nil || taken {
		return scheme, err
	}

	flux := r.roller.Flux()
	seq := r.log.Flux(flux, r.def.Schemes.Cite)

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSchemeFlux,
		Prompt:  "Modify the Scheme Flux?",
		Options: []string{"As rolled", "Minus 1", "Plus 1"},
		Scores:  []int{flux.Value, flux.Value - 1, flux.Value + 1},
		Cite:    r.def.Schemes.Cite + " (Flux may be modified (after roll) plus or minus 1)",
	})
	if err != nil {
		return career.SchemeRow{}, err
	}

	value := []int{flux.Value, flux.Value - 1, flux.Value + 1}[chosen]

	scheme := r.def.Schemes.SchemeAt(value)
	r.record.Scheme = scheme.Career

	r.log.Consequence(ConsequenceEvent{
		Cause: seq, Kind: ConsequenceScheme, Career: r.def.Name, Skill: scheme.Career, Value: value,
	})

	return scheme, nil
}

// selectScheme offers the chart's alternative to rolling: "A Rogue may
// select for his Scheme (rather than roll) any previous career" (chart
// 10, p. 84). It reports the scheme and whether one was taken.
//
// Selection replaces the roll rather than steering it, so a Rogue who
// takes a previous career consumes no Flux and the "Flux may be modified
// (after roll)" clause never reaches him — it scopes itself to a roll
// that was made (interpretation I-109, ERRATA.md).
//
// The offer is made only where there is something to take, so a Rogue who
// has served nothing else is asked nothing. Chart 10 prints a row for
// every career, so a selection always names one of them and the Value is
// the chart's own rather than an invented one.
func (*rogueMechanics) selectScheme(r *careerRun) (career.SchemeRow, bool, error) {
	previous := previousCareers(r.character)
	if len(previous) == 0 {
		return career.SchemeRow{}, false, nil
	}

	options := make([]string, 0, len(previous)+1)
	options = append(options, previous...)
	options = append(options, rogueRollTheScheme)

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSchemeCareer,
		Prompt:  "Select a previous career as the Scheme?",
		Options: options,
		Cite: r.def.Schemes.Cite +
			" (A Rogue may select for his Scheme (rather than roll) any previous career)",
	})
	if err != nil {
		return career.SchemeRow{}, false, err
	}

	if options[chosen] == rogueRollTheScheme {
		return career.SchemeRow{}, false, nil
	}

	scheme, ok := r.def.Schemes.SchemeFor(options[chosen])
	if !ok {
		// Unreachable while previousCareers draws on the same career
		// registry the Schemes table names, and stated rather than
		// assumed: a career with no row would otherwise pay Cr0.
		return career.SchemeRow{}, false, fmt.Errorf("%w: chart 10 prints no Scheme row for %q",
			errNotImplemented, options[chosen])
	}

	r.record.Scheme = scheme.Career

	r.log.Consequence(ConsequenceEvent{
		Cause: seq, Kind: ConsequenceScheme, Career: r.def.Name,
		Skill: scheme.Career, Detail: rogueSchemeSelected,
	})

	return scheme, true, nil
}

// previousCareers lists the careers the character has served, in the
// order he served them and without repeats.
//
// Served, not attempted: a career whose To Begin failed was never held,
// which is the same reading I-54 takes of "this career may not be used"
// (p. 65). The career now in progress is not among them — its record
// joins the character only when the run ends — so a Rogue cannot take
// the stint he is standing in, though an earlier Rogue stint counts and
// chart 10 prints a Rogue row for it.
func previousCareers(c *Character) []string {
	names := make([]string, 0, len(c.Careers))

	for _, record := range c.Careers {
		if record.Began && !slices.Contains(names, record.Career) {
			names = append(names, record.Career)
		}
	}

	return names
}

// imprison applies a Risk failure: "Prison for (sum of negative Mods +
// Flux) years at the start of the next Term (may be zero; maximum 4).
// Fame +1 (actually Infamy)" (chart 10).
//
// caution is the Caution/Bravery selection alone, not the total Risk mod:
// "Reduce the Controlling Characteristic by all negative Mods; ignore any
// positive Mods" (p. 65), so chart 10's positive "+Terms" mod may not net
// against a negative Bravery mod before the sum is taken — the same rule
// negativeMods applies to the Armed Forces Branch and Operations mods.
func (*rogueMechanics) imprison(r *careerRun, caution, cause int) {
	flux := r.roller.Flux()
	r.log.Flux(flux, r.def.Cite+" (Prison for the sum of negative Mods and Flux)")

	years := min(max(-(min(caution, 0)+flux.Value), 0), roguePrisonMaxYears)
	r.record.PrisonYears = years

	// "Fame +1 (actually Infamy)" (chart 10). Chart F prices schemes, not
	// imprisonments, so this is the career's own tracked Fame — the same
	// field chart 03 uses — and chart F adds it (interpretation I-67).
	r.record.Fame++
	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceFameChange, Delta: 1, Value: r.record.Fame,
	})

	if years == 0 {
		return
	}

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceSentenced, Career: r.def.Name, Value: years,
	})
}

// payoff applies a Reward success: "Payoff= V x (1+CC-R+Mods)", halved
// where the Risk also failed (chart 10; interpretation I-44, ERRATA.md).
func (*rogueMechanics) payoff(
	r *careerRun, scheme career.SchemeRow, cc, reward, mods int, caught bool, cause int,
) {
	multiplier := max(1+cc-reward+mods, 0)

	switch {
	case scheme.ShipShares > 0:
		shares := scheme.ShipShares * multiplier
		if caught {
			shares /= 2
		}

		r.record.ShipShares += shares

		r.log.Consequence(ConsequenceEvent{
			Cause: cause, Kind: ConsequenceShipShares, Delta: shares, Value: r.record.ShipShares,
		})
	default:
		payoff := scheme.Credits * multiplier
		if caught {
			payoff /= 2
		}

		r.record.SchemePayoff += payoff

		r.log.Consequence(ConsequenceEvent{
			Cause: cause, Kind: ConsequencePayoff, Career: r.def.Name,
			Delta: payoff, Value: r.record.SchemePayoff,
			Skill: "x" + strconv.Itoa(multiplier),
		})
	}
}
