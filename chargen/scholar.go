package chargen

// Scholar career mechanics (Book 1 chart 02, p. 76; prose pp. 64-66).
// "To Begin (Edu 8+) Automatic / To Begin Edu or Tra / Risk & Reward
// C1 C2 C3 C4 / Promotion (Edu 8+) Int* / Tenure Pub x3 / Continue Edu* or
// Tra*" with "Comment *Mod +Pubs" (chart 02 box A).
//
// The Scholar's Risk and Reward are Research and Publication: "Roll for
// Risk against CC+ Mods ... If Research Complete, roll for Publication
// against CC+ (opposite sign) Mods." Research failure is the shared injury
// (see careerRun.injury); Publication success adds a Publication, and an
// unusually good roll adds two.
//
// Rank is Edu-gated rather than chosen: "A character with Edu 8+ is
// automatically Scholar1 to Begin ... A character with Edu 7 or less is an
// Amateur Scholar. He can resolve Risk and Reward, but cannot be
// Promoted." Promotion past Scholar3 needs Tenure.
//
// Any adverse roll may be waived (chart 02 Waivers; the p. 59 rule, shared
// with education).
//
// Deferred: the non-human C5=Tra Non-Traditional Scholar and the C5=Ins
// exclusion (v1 is human-only, docs/PRD.md); muster-out table D
// (milestone 4).

import (
	"fmt"
	"strconv"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/skill"
)

// The chart 02 rank ladder and its gates.
const (
	scholarAmateur  = "S0" // "0= Edu less than 8"
	scholarLecturer = "S1" // "1= Automatic if Edu 8+"
	scholarTenureAt = "S3" // "3= Eligible for Tenure"

	// scholarPromoteEdu is the Edu at or above which promotion is possible
	// ("Promotion is possible only to those with Edu 8+").
	scholarPromoteEdu = 8

	// scholarTenureEdu is the Edu at or above which Tenure may be applied
	// for ("Scholar with Edu 10+ may apply for Tenure").
	scholarTenureEdu = 10

	// scholarTenureFactor is the Tenure target multiplier ("Tenure Pub x3").
	scholarTenureFactor = 3

	// scholarAwardMargin is how far under the characteristic a Publication
	// roll must land to be Award-Winning ("If Publication Roll is 4 less
	// than Characteristic ... counts as TWO").
	scholarAwardMargin = 4

	// scholarResearchMajor is the Major award for a completed Research
	// ("Research Success Major +2", chart 02 table B).
	scholarResearchMajor = 2
)

// scholarMechanics is the Scholar careerMechanics implementation.
type scholarMechanics struct {
	rank         string
	publications int

	// awardWinning counts only the Award-Winning publications, which
	// publications cannot be read back from: an Award-Winning one "counts
	// as TWO" (chart 02; I-25), so a count of two is either one award or
	// two ordinary papers. Chart M1's TAS Life Membership is the
	// "Award-Winning Scholar's" (p. 67), and needs the distinction.
	awardWinning int

	tenured bool
}

// newScholar is the Scholar careerRegistry entry.
//
//nolint:ireturn // The registry's function type returns the interface.
func newScholar() (*career.Definition, careerMechanics, error) {
	def, err := career.Scholar()
	if err != nil {
		return nil, nil, fmt.Errorf("scholar career: %w", err)
	}

	return def, &scholarMechanics{}, nil
}

// begin enters the career: automatic at Scholar1 for Edu 8+, otherwise a
// To Begin check against Edu entering as an Amateur (chart 02 box A and
// the rank table's gates). Every Scholar then has a Major and a Minor.
func (m *scholarMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Scholar: To Begin", r.def.Cite)

	// The automatic entry rolls nothing, so the career-selection choice is
	// what causes it (docs/PRD.md FR10).
	//
	// The condition is chart 02's "(Edu 8+) Automatic", read from the
	// career data so that the menu and the entry cannot disagree about
	// which characters it covers. scholarPromoteEdu stays where it is: it
	// gates promotion, a different sentence on the same chart that happens
	// to name the same number.
	if entersAutomatically(r.def, r.character) {
		// "A character with Edu 8+ is automatically Scholar1 to Begin."
		if err := m.setRank(r, scholarLecturer, r.entryCause); err != nil {
			return false, err
		}

		return true, m.selectAreas(r)
	}

	throw := r.roller.Check(2, r.character.Characteristics.Edu)
	cause := r.log.Throw(throw, nil, r.def.Cite+" (To Begin vs Edu)")

	begun, err := m.resolveBeginThrow(r, throw.Success, cause)
	if err != nil || !begun {
		return false, err
	}

	if err := m.setRank(r, scholarAmateur, cause); err != nil {
		return false, err
	}

	return true, m.selectAreas(r)
}

// resolveBeginThrow applies a To Begin outcome, offering a waiver on a
// failure ("Position", chart 02 Waivers).
func (*scholarMechanics) resolveBeginThrow(r *careerRun, success bool, cause int) (bool, error) {
	if success {
		return true, nil
	}

	waived, err := r.waive(scholarWaiver("Position: To Begin failed", true), cause)
	if err != nil || waived {
		return waived, err
	}

	// "Each failed attempt ... takes one year" (p. 65).
	if err := r.character.advanceYears(1, r.roller, r.log, cause); err != nil {
		return false, err
	}

	r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceCareerNotBegun, Career: r.def.Name})

	return false, nil
}

// setRank records a rank from the chart 02 table. An id absent from the
// table is a data error, not a rankless Scholar: the constants above are
// the only coupling between this file and scholar.json.
func (m *scholarMechanics) setRank(r *careerRun, id string, cause int) error {
	rank, ok := r.def.RankByID(id)
	if !ok {
		return fmt.Errorf("%w: %q", errUnknownRank, id)
	}

	m.rank = rank.ID
	r.record.Rank = rank.ID
	r.record.RankTitle = rank.Title

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceRankSet, Career: r.def.Name, Skill: rank.Title,
	})

	return nil
}

// selectAreas gives a Scholar without a degree a Major and a Minor: "Every
// Scholar has a Major and a Minor. If no degree (and an associated Major
// and Minor) then select any Skill or Knowledge from the Skills List"
// (chart 02; interpretation I-23, ERRATA.md).
func (m *scholarMechanics) selectAreas(r *careerRun) error {
	if r.character.currentMajor() != "" && r.character.currentMinor() != "" {
		return nil
	}

	names := skill.Names()

	if r.character.currentMajor() == "" {
		chosen, seq, err := choose(r.log, r.decider, Choice{
			ID:      ChooseMajor,
			Prompt:  "Select a Scholar Major",
			Options: names,
			Cite:    r.def.Cite + " (select any Skill or Knowledge from the Skills List)",
		})
		if err != nil {
			return err
		}

		r.record.Major = names[chosen]
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceMajorSet, Skill: names[chosen]})
	}

	if r.minor() != "" {
		return nil
	}

	// "they cannot be the same" (p. 59).
	options := make([]string, 0, len(names))

	for _, name := range names {
		if name != r.major() {
			options = append(options, name)
		}
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseMinor,
		Prompt:  "Select a Scholar Minor",
		Options: options,
		Cite:    r.def.Cite + " (select any Skill or Knowledge from the Skills List)",
	})
	if err != nil {
		return err
	}

	r.record.Minor = options[chosen]
	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceMinorSet, Skill: options[chosen]})

	return nil
}

// resolveTerm runs Research and Publication, then the term's promotion and
// Tenure attempts.
func (m *scholarMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	outcome, err := m.researchAndPublication(r, cc)
	if err != nil || outcome.died {
		return outcome, err
	}

	promoted, err := m.advance(r)
	if err != nil {
		return termOutcome{}, err
	}

	outcome.skillRolls = r.def.SkillsPerTerm
	if promoted {
		outcome.skillRolls += r.def.SkillsPerAdvancement
	}

	return outcome, nil
}

// researchAndPublication runs the chart 02 Risk & Reward box.
func (m *scholarMechanics) researchAndPublication(r *careerRun, cc string) (termOutcome, error) {
	var outcome termOutcome

	mod, err := chooseRiskMod(r, "chart 02 p. 76")
	if err != nil {
		return outcome, err
	}

	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return outcome, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	research := r.roller.Check(2, value+mod)
	seq := r.log.Throw(research, riskMods(mod, 1), r.def.Cite+" (Research vs "+cc+"+Mods)")

	completed := research.Success

	if !completed {
		waived, err := r.waive(scholarWaiver("Research: incomplete with injury", false), seq)
		if err != nil {
			return outcome, err
		}

		completed = waived

		if !waived {
			died, disabled := r.injury(cc, mod, seq,
				"Book 1 p. 76 chart 02 (Research Failure: reduce CC by negative Mods and Flux)")
			outcome.endCause = seq

			if died {
				outcome.died = true

				return outcome, nil
			}

			outcome.endCareer = disabled
		}
	}

	if !completed {
		return outcome, nil
	}

	outcome.success = true

	// "Research Success Major +2" (chart 02 table B).
	if name := r.major(); name != "" {
		r.awardAndLog(name, scholarResearchMajor, seq)
	}

	return outcome, m.publish(r, cc, value, mod)
}

// publish rolls Publication: "If Research Complete, roll for Publication
// against CC+ (opposite sign) Mods" (chart 02).
func (m *scholarMechanics) publish(r *careerRun, cc string, value, mod int) error {
	throw := r.roller.Check(2, value-mod)
	seq := r.log.Throw(throw, riskMods(mod, -1), r.def.Cite+" (Publication vs "+cc+"+ opposite sign Mods)")

	published := throw.Success

	if !published {
		waived, err := r.waive(scholarWaiver("Publication: rejected", false), seq)
		if err != nil {
			return err
		}

		published = waived
	}

	if !published {
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceNoAward})

		return nil
	}

	// "If Publication Roll is 4 less than Characteristic, it is
	// <Award-Winning> and counts as TWO." (chart 02; interpretation I-25.)
	// The margin is read off a roll that carried the Publication on its
	// own: a Caution Mod of +5 or more puts the target below
	// Characteristic-4, so a rejected roll rescued by a Waiver can sit
	// inside the margin while having failed. A waived rejection is a plain
	// Publication (I-25).
	count := 1
	if throw.Success && throw.Total <= value-scholarAwardMargin {
		count = 2
		m.awardWinning++
		r.record.AwardWinningPublications = m.awardWinning
	}

	m.publications += count
	r.record.Publications = m.publications

	r.log.Consequence(ConsequenceEvent{
		Cause: seq, Kind: ConsequencePublication, Delta: count, Value: m.publications,
	})

	return nil
}

// advance runs the term's promotion and Tenure attempts, reporting whether
// a rank was gained. Eligibility is tested against the state at the start
// of the phase, so a Scholar promoted to Scholar3 this term does not also
// apply for Tenure, and one tenured this term does not also promote
// (interpretation I-13's rule, applied here).
func (m *scholarMechanics) advance(r *careerRun) (bool, error) {
	entryRank, entryTenured := m.rank, m.tenured

	if m.mayApplyForTenure(r, entryRank, entryTenured) {
		if err := m.applyForTenure(r); err != nil {
			return false, err
		}

		return false, nil
	}

	if !m.mayPromote(r, entryRank, entryTenured) {
		return false, nil
	}

	return m.promote(r)
}

// mayApplyForTenure reports the chart 02 Tenure gate: "Scholar with Edu
// 10+ may apply for Tenure upon reaching Scholar3 and in every Term in
// which the Character is Scholar3".
func (*scholarMechanics) mayApplyForTenure(r *careerRun, rank string, tenured bool) bool {
	return !tenured &&
		rank == scholarTenureAt &&
		r.character.Characteristics.Edu >= scholarTenureEdu
}

// mayPromote reports the chart 02 promotion gates: Edu 8+, a rank above
// it on the ladder, and Tenure to pass Scholar3.
func (m *scholarMechanics) mayPromote(r *careerRun, rank string, tenured bool) bool {
	if r.character.Characteristics.Edu < scholarPromoteEdu {
		// "A character with Edu 7 or less is an Amateur Scholar ... cannot
		// be Promoted." The gate reads against the current Edu, which the
		// Personal column can raise (interpretation I-26, ERRATA.md).
		return false
	}

	if rank == scholarTenureAt && !tenured {
		// "Promotion beyond Scholar3 not possible without Tenure."
		return false
	}

	_, hasNext := r.def.NextRank(rank)

	return hasNext
}

// applyForTenure rolls Tenure against "Pub x3" (chart 02 box A).
func (m *scholarMechanics) applyForTenure(r *careerRun) error {
	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseTenure,
		Prompt:  "Apply for Tenure?",
		Options: []string{"Apply for Tenure", "Decline"},
		Cite:    r.def.Cite + " (Tenure Pub x3)",
	})
	if err != nil || chosen != 0 {
		return err
	}

	// The whole target is the tracked value, so there is no modifier to
	// itemize (as with the Fame Continue target); the cite carries the
	// derivation and the recorded target carries the count.
	target := m.publications * scholarTenureFactor
	throw := r.roller.Check(2, target)
	seq := r.log.Throw(throw, nil,
		r.def.Cite+" (Tenure vs Publications x"+strconv.Itoa(scholarTenureFactor)+")")

	granted := throw.Success

	if !granted {
		waived, err := r.waive(scholarWaiver("Tenure: refused", false), seq)
		if err != nil {
			return err
		}

		granted = waived
	}

	if !granted {
		return nil
	}

	m.tenured = true
	r.record.Tenured = true

	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceTenure, Career: r.def.Name})

	return nil
}

// promote rolls Promotion against "Int*" with "*Mod +Pubs" (chart 02).
func (m *scholarMechanics) promote(r *careerRun) (bool, error) {
	target := r.character.Characteristics.Int + m.publications

	mods := []Mod{}
	if m.publications != 0 {
		mods = append(mods, Mod{Name: "Publications", Value: m.publications})
	}

	throw := r.roller.Check(2, target)
	seq := r.log.Throw(throw, mods, r.def.Cite+" (Promotion vs Int +Pubs)")

	promoted := throw.Success

	if !promoted {
		waived, err := r.waive(scholarWaiver("Promotion: refused", false), seq)
		if err != nil {
			return false, err
		}

		promoted = waived
	}

	if !promoted {
		return false, nil
	}

	next, ok := r.def.NextRank(m.rank)
	if !ok {
		return false, nil
	}

	if err := m.setRank(r, next.ID, seq); err != nil {
		return false, err
	}

	return true, nil
}

// scholarWaiver names a chart 02 waiver-able event. The Continue waiver —
// the sixth event the box lists — is offered by the generic runner, gated
// on the definition's ContinueWaiver (careerrun.go).
func scholarWaiver(reason string, careerEnding bool) waiverPrompt {
	return careerWaiver(reason, "Book 1 p. 76 chart 02", careerEnding)
}
