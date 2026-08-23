package chargen

// The generic career runner: the term loop, controlling-characteristic
// rotation, career skills table, and Continue throw shared by every career
// (Book 1 chart D p. 64; prose pp. 65-66). Career-specific exceptional
// mechanics (docs/PRD.md, Architecture notes) plug in through the
// careerMechanics interface; Citizen's live in citizen.go.

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/skill"
)

// errUnknownCharacteristic reports a data-file characteristic name outside
// the six standard abbreviations.
var errUnknownCharacteristic = errors.New("unknown characteristic")

// errNotImplemented reports a table cell whose resolution lands in a later
// milestone (docs/PRD.md milestones 2-3).
var errNotImplemented = errors.New("not implemented until education/skill milestones")

// errUnknownRank reports a rank id absent from the career's rank table.
var errUnknownRank = errors.New("unknown rank")

// errUnknownAdvancementTarget reports an advancement target form the
// engine does not implement.
var errUnknownAdvancementTarget = errors.New("unknown advancement target")

// errUnregisteredCareer reports a career present in career.Available but
// missing from careerRegistry — an internal wiring bug, distinct from the
// user-facing ErrUnknownCareer (which the CLI maps to a usage exit).
var errUnregisteredCareer = errors.New("career has no registered mechanics")

// careerMechanics is one career's exceptional mechanics. The interface is
// unexported and grows with the careers that need more seams (rank,
// commission, muster out land with milestones 3-4).
type careerMechanics interface {
	// begin resolves career entry: automatic for Citizen (chart 04), a
	// To Begin throw for most careers (chart D p. 64; p. 65). It reports
	// whether the career began; a failed attempt costs a year (p. 65).
	begin(r *careerRun) (bool, error)

	// resolveTerm runs the career's Risk/Reward variant for the term
	// (p. 65: Citizen Life for Citizens) and applies its awards.
	resolveTerm(r *careerRun, cc string) (termOutcome, error)
}

// termOutcome is a term's Risk/Reward-variant result.
type termOutcome struct {
	// success is the variant's outcome (Citizen Life success, a Scout
	// Discovery).
	success bool

	// skillRolls overrides the definition's SkillsPerTerm when non-zero
	// (chart 05 table B splits eligibility by duty).
	skillRolls int

	// endCareer ends the career after the term completes, without a
	// Continue roll — a disabled character "Musters Out at Term end"
	// (chart 05 p. 79; muster out itself is milestone 4).
	endCareer bool

	// termColumns restricts the term's skill rolls to named columns, for
	// the Armed Forces: "Term skills (but not commission, promotion, or
	// other skill eligibilities) may be taken on a column of the Skills
	// table corresponding to an Operations result received in the Term"
	// (p. 65). bonusRolls are the unrestricted eligibilities.
	termColumns []string
	bonusRolls  int

	// died ends the term and the career immediately: no skills, no
	// Continue ("the Character is dead", p. 65).
	died bool

	// endCause is the throw that ended the career, and so the cause of
	// the years the term still elapses. Both career-ending paths skip the
	// Continue roll, which is what carries the cause on an ordinary term
	// (docs/PRD.md FR10: a consequence names the throw behind it, never a
	// step). Read only when died or endCareer is set.
	endCause int
}

// careerRegistry maps canonical career names to their definition and
// mechanics. Its key set must match career.Available (tested).
var careerRegistry = map[string]func() (*career.Definition, careerMechanics, error){
	"Citizen":     newCitizen,
	"Scholar":     newScholar,
	"Noble":       newNoble,
	"Soldier":     newSoldier,
	"Spacer":      newSpacer,
	"Marine":      newMarine,
	"Agent":       newAgent,
	"Rogue":       newRogue,
	"Entertainer": newEntertainer,
	"Scout":       newScout,
	"Merchant":    newMerchant,
}

// careerRun is the shared state of one career resolution.
type careerRun struct {
	def       *career.Definition
	mechanics careerMechanics
	roller    *dice.Roller
	log       *Log
	decider   Decider
	character *Character

	// availableCCs rotates the controlling characteristic: "This
	// Controlling Characteristic cannot be used again until all of the
	// others in the sequence have been used." (p. 65)
	availableCCs []string

	// entryLevels are the skill levels held when the career began; the
	// first-receipt rule counts receipts against this baseline
	// (interpretation I-2, ERRATA.md).
	entryLevels map[string]int

	// entryCause is the sequence of the career-selection choice — the
	// cause for consequences of entering a career that needs no throw
	// (chart 02's automatic Scholar entry; docs/PRD.md FR10).
	entryCause int

	record CareerRecord
}

// runCareerByName resolves one career through the registry, reporting
// whether the career began (a failed To Begin leaves a began:false record
// and the caller offers the remaining careers, p. 65).
func runCareerByName(
	name string, entryCause int, roller *dice.Roller, log *Log, decider Decider, character *Character,
) (bool, error) {
	entry, ok := careerRegistry[name]
	if !ok {
		return false, fmt.Errorf("%w: %q", errUnregisteredCareer, name)
	}

	def, mechanics, err := entry()
	if err != nil {
		return false, err
	}

	// The baseline is captured before mechanics.begin deliberately: a
	// skill granted during career entry (a To Begin outcome, milestone 3)
	// is a career receipt under interpretation I-2 (ERRATA.md) and demotes
	// a later Job/Hobby determination of the same skill to a later receipt.
	entryLevels := make(map[string]int, len(character.Skills))
	for _, held := range character.Skills {
		entryLevels[held.Name] = held.Level
	}

	run := &careerRun{
		def:         def,
		entryCause:  entryCause,
		mechanics:   mechanics,
		roller:      roller,
		log:         log,
		decider:     decider,
		character:   character,
		record:      CareerRecord{Career: def.Name},
		entryLevels: entryLevels,
	}

	began, err := mechanics.begin(run)
	if err != nil {
		return false, err
	}

	run.record.Began = began

	if !began {
		character.Careers = append(character.Careers, run.record)

		return false, nil
	}

	for {
		continued, err := run.term(len(run.record.Terms) + 1)
		if err != nil {
			return false, err
		}

		if !continued {
			break
		}
	}

	run.recordSanityMod(entryCause)

	character.Careers = append(character.Careers, run.record)

	return true, nil
}

// elapseTerm advances the clock by the term's four years, resolving any
// Aging Check the span crosses (p. 66; chart A p. 89).
func (r *careerRun) elapseTerm(cause int) error {
	return r.character.advanceYears(TermYears, r.roller, r.log, cause)
}

// closeTerm ends the term and reports whether the career continues.
//
// A disabled character "Musters Out at Term end" (chart 05 p. 79): the
// term completes and its years pass, but there is no Continue throw, which
// is what elapses them on an ordinary term.
func (r *careerRun) closeTerm(outcome termOutcome) (bool, error) {
	if outcome.endCareer {
		return false, r.elapseTerm(outcome.endCause)
	}

	return r.continueRoll()
}

// term resolves one 4-year term and reports whether the career continues.
func (r *careerRun) term(number int) (bool, error) {
	r.log.Step(r.def.Name+": Term "+strconv.Itoa(number), r.def.Cite)

	cc, err := r.chooseCC()
	if err != nil {
		return false, err
	}

	outcome, err := r.mechanics.resolveTerm(r, cc)
	if err != nil {
		return false, err
	}

	if outcome.died {
		// "the Character is dead" (p. 65): the term ends at the injury —
		// no skills, no Continue. The years still pass: the Continue
		// throw is what elapses them on an ordinary term, and this path
		// never reaches it.
		if err := r.elapseTerm(outcome.endCause); err != nil {
			return false, err
		}

		r.record.Terms = append(r.record.Terms, TermRecord{
			Term: number, ControllingCharacteristic: cc,
		})

		return false, nil
	}

	if err := r.termSkills(outcome.skillRolls, outcome.termColumns); err != nil {
		return false, err
	}

	if outcome.bonusRolls > 0 {
		// Commission and promotion eligibilities are not restricted to
		// the term's assignments (p. 65).
		if err := r.termSkills(outcome.bonusRolls, nil); err != nil {
			return false, err
		}
	}

	continued, err := r.closeTerm(outcome)
	if err != nil {
		return false, err
	}

	// Aging resolves as the term's years pass, and can kill: "The second
	// time three characteristics are reduced to 0, the character dies"
	// (chart A p. 89). A Continue throw already rolled cannot bring the
	// character back to serve it.
	if r.character.Dead {
		continued = false
	}

	r.record.Terms = append(r.record.Terms, TermRecord{
		Term:                      number,
		ControllingCharacteristic: cc,
		Success:                   outcome.success,
		Continued:                 continued,
	})

	return continued, nil
}

// chooseCC selects the term's controlling characteristic: "The player
// picks one of these Characteristics (any one anywhere in the sequence)
// ... This Controlling Characteristic cannot be used again until all of
// the others in the sequence have been used" (p. 65).
func (r *careerRun) chooseCC() (string, error) {
	// "A Rogue selects one Controlling Characteristic ... which is then
	// used throughout his career (not just in the current Term)"
	// (chart 10, p. 84): chosen once, then reused without a further
	// choice event.
	if r.def.CCFixed && r.record.ControllingCharacteristic != "" {
		return r.record.ControllingCharacteristic, nil
	}

	// Chart 03's "Risk & Reward Talent" names no series, so the
	// Entertainer rotates nothing and the term has no controlling
	// characteristic.
	if len(r.def.ControllingCharacteristics) == 0 {
		return "", nil
	}

	if len(r.availableCCs) == 0 {
		r.availableCCs = slices.Clone(r.def.ControllingCharacteristics)
	}

	scores := make([]int, len(r.availableCCs))
	for i, name := range r.availableCCs {
		scores[i], _ = characteristicValue(&r.character.Characteristics, name)
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseControllingCharacteristic,
		Prompt:  "Select the term's controlling characteristic",
		Options: slices.Clone(r.availableCCs),
		Scores:  scores,
		Cite:    "Book 1 p. 65 (Risk and Reward: Select the CC)",
	})
	if err != nil {
		return "", err
	}

	cc := r.availableCCs[chosen]

	if r.def.CCFixed {
		r.record.ControllingCharacteristic = cc

		return cc, nil
	}

	r.availableCCs = slices.Delete(r.availableCCs, chosen, chosen+1)

	return cc, nil
}

// firstReceiptLevels applies the first-receipt rule: the stated level on
// first receipt, Skill-1 thereafter (docs/PRD.md FR5; for Citizen, "with
// Skill-4 (later receipts are Skill-1)", p. 78). A skill already received
// during this career counts as held, so the determination is a later
// receipt: +1. Pre-career levels (homeworld grants, education) are not
// career receipts and do not demote the award — interpretation I-2,
// ERRATA.md.
func (r *careerRun) firstReceiptLevels(name string, firstReceipt int) int {
	if r.character.skillLevel(name) > r.entryLevels[name] {
		return 1
	}

	return firstReceipt
}

// awardAndLog awards skill levels via the career-independent
// awardSkillAndLog.
func (r *careerRun) awardAndLog(name string, levels, cause int) {
	awardSkillAndLog(name, levels, cause, r.log, r.character)
}

// groupCells maps the chart's open-selection cells to the Master Skill
// List group they select from: "One Art", "One Trade", "One Science", and
// "Starship Skill" (charts 04-06; p. 132 chart MS). Chart B's own Art and
// Trade lists (p. 56) are the same six and ten names.
var groupCells = map[career.EntryKind]struct {
	prompt string
	names  func() []string
}{
	career.EntryArt: {
		prompt: "Select One Art",
		names:  func() []string { return skill.InGroup(skill.GroupArts) },
	},
	career.EntryTrade: {
		prompt: "Select One Trade",
		names:  func() []string { return skill.InGroup(skill.GroupTrades) },
	},
	career.EntryScience: {
		prompt: "Select One Science",
		names:  func() []string { return skill.UnderParent(skill.ParentSciences) },
	},
	career.EntryStarship: {
		prompt: "Select a Starship Skill",
		names:  func() []string { return skill.InGroup(skill.GroupStarship) },
	},
	career.EntrySoldier: {
		prompt: "Select a Soldier Skill",
		names:  func() []string { return skill.InGroup(skill.GroupSoldier) },
	},

	// Chart 09's Vocation column prints "Any Knowledge" (p. 83) and chart
	// 13's General column "Any Skill*** ... from Citizen Life Skills and
	// Knowledges" (p. 87); both are open selections over a whole list
	// rather than a Master Skill List group.
	career.EntryAnyKnowledge: {
		prompt: "Select Any Knowledge",
		names:  allKnowledges,
	},
	career.EntryAnySkill: {
		prompt: "Select Any Skill from Citizen Life Skills and Knowledges",
		names:  citizenLifeSkills,
	},
}

// citizenLifeSkills is chart 13's "Any Skill*** from Citizen Life Skills
// and Knowledges" (p. 87): every table E entry of chart 04, in chart
// order. Chart 09's Undercover Assignment reads the same list.
func citizenLifeSkills() []string {
	def, err := career.Citizen()
	if err != nil {
		return nil
	}

	return def.HobbyChoices()
}

// allKnowledges is chart 09's "Any Knowledge" cell (p. 83): every
// Knowledge on the Master Skill List (p. 132 chart MS), in name order.
func allKnowledges() []string {
	var names []string

	for _, name := range skill.Names() {
		if entry, ok := skill.Lookup(name); ok && entry.Kind == skill.KindKnowledge {
			names = append(names, name)
		}
	}

	return names
}

// article returns the indefinite article for a career name, so the prompt
// reads "an Entertainer" rather than "a Entertainer".
func article(name string) string {
	if name == "" {
		return "a"
	}

	if strings.ContainsRune("AEIOU", rune(name[0])) {
		return "an"
	}

	return "a"
}

// waive offers a waiver on an adverse outcome and records the negation
// when it succeeds (chart 02 Waivers, p. 76; the p. 59 rule).
func (r *careerRun) waive(prompt waiverPrompt, cause int) (bool, error) {
	waived, err := offerWaiver(r.log, r.decider, r.roller, r.character, prompt)
	if err != nil || !waived {
		return false, err
	}

	r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceWaived})

	return true, nil
}

// major and minor report the character's current Major and Minor,
// preferring any the career in progress selected for itself (chart 02's
// degreeless Scholar) over the education records, since its record is not
// on the character until the career ends.
func (r *careerRun) major() string {
	if r.record.Major != "" {
		return r.record.Major
	}

	return r.character.currentMajor()
}

func (r *careerRun) minor() string {
	if r.record.Minor != "" {
		return r.record.Minor
	}

	return r.character.currentMinor()
}

// awardOpenCell resolves the cells that select from a Master Skill List
// group, and rejects the ones still waiting on later milestones.
func (r *careerRun) awardOpenCell(kind career.EntryKind) error {
	if kind == career.EntryCapital {
		// "Capital*** = World Knowledge (of world of highest held noble
		// Land Grant)" (chart 11 p. 85): the Land Grant worlds land with
		// muster out (docs/PRD.md milestone 4).
		return fmt.Errorf("%w: Capital cell needs the Land Grant worlds", errNotImplemented)
	}

	return r.awardFromGroup(kind)
}

// awardFromGroup resolves an open-selection cell by choice and awards the
// selected skill, caused by the selecting choice event (docs/PRD.md FR10).
func (r *careerRun) awardFromGroup(kind career.EntryKind) error {
	cell, ok := groupCells[kind]
	if !ok {
		return fmt.Errorf("%w: %q cell", errNotImplemented, kind)
	}

	options := cell.names()
	if len(options) == 0 {
		return fmt.Errorf("%w: %q cell has no alternatives", errNotImplemented, kind)
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSkill,
		Prompt:  cell.prompt,
		Options: options,
		Cite:    r.def.Cite + " table C; Book 1 p. 132 chart MS",
	})
	if err != nil {
		return err
	}

	r.awardAndLog(options[chosen], 1, seq)

	return nil
}

// resolveSkillName maps a career chart cell to its Master Skill List name.
// Most cells name one entry. Two chart 04 table E cells do not: "Grav" is
// printed once although the list holds a Grav knowledge under each of
// Driver, Flyer, and Seafarer, and "Spacecraft" covers both Spacecraft ACS
// and Spacecraft BCS (p. 132). Those are resolved by choice, in Master
// Skill List order (ERRATA.md I-10, I-11).
//
// exclude names entries the caller has already spent — the Citizen Hobby
// must differ from the Job (interpretation I-3), and the chart prints the
// label, not the resolved name, so the exclusion can only be applied here.
// A label always covers at least two entries, so at least one survives.
func (r *careerRun) resolveSkillName(name string, exclude ...string) (string, error) {
	options := skill.Options(name)
	if options == nil {
		return name, nil
	}

	options = slices.DeleteFunc(options, func(option string) bool {
		return slices.Contains(exclude, option)
	})

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSkill,
		Prompt:  "Select the specific " + name + " skill",
		Options: options,
		Cite:    "Book 1 p. 132 chart MS (Master Skill List); " + r.def.Cite,
	})
	if err != nil {
		return "", err
	}

	return options[chosen], nil
}

// termSkills rolls the per-term career skills table eligibility (for
// Citizen, "Per Term: 4 on Table C", chart 04 table B); "For each skill,
// roll on the Career Skills Table. The character selects a column and
// rolls 1D for the specific skill" (p. 65).
func (r *careerRun) termSkills(rolls int, only []string) error {
	if rolls == 0 {
		rolls = r.def.SkillsPerTerm
	}

	columns := r.skillColumnOptions()
	if len(only) > 0 {
		columns = only
	}

	for range rolls {
		chosen, choiceSeq, err := choose(r.log, r.decider, Choice{
			ID:      ChooseSkillColumn,
			Prompt:  "Select " + article(r.def.Name) + " " + r.def.Name + " Skills column",
			Options: columns,
			Cite:    "Book 1 p. 65 (the character selects a column and rolls 1D)",
		})
		if err != nil {
			return err
		}

		index := r.columnIndex(columns[chosen])
		if index < 0 {
			// "A Scholar may always take a skill in his Major or Minor
			// instead of from this table" (chart 02 table C): no 1D roll.
			if err := r.awardMajorOrMinor(columns[chosen], choiceSeq); err != nil {
				return err
			}

			continue
		}

		roll := r.roller.Roll(1)
		seq := r.log.Roll(roll, r.def.Cite+" table C, column "+columns[chosen])

		entry := r.def.SkillColumns[index].Entries[roll.Total-1]
		if err := r.awardTableC(entry, seq); err != nil {
			return err
		}
	}

	return nil
}

// rollUnder rolls 1D and rerolls until the face is within limit, which is
// how the charts read a column narrower than a die: chart 04 table E's
// "Roll A (reroll if >3)" (p. 78) and chart 09's A and C columns (p. 83).
// Every consumed face is logged, so the event log accounts for the
// rerolls.
func (r *careerRun) rollUnder(limit int, cite string) int {
	for {
		roll := r.roller.Roll(1)
		r.log.Roll(roll, cite)

		if roll.Total <= limit {
			return roll.Total
		}
	}
}

// columnIndex returns the table C column with the given name, or -1 for
// the Major and Minor options appended alongside them.
func (r *careerRun) columnIndex(name string) int {
	for i, column := range r.def.SkillColumns {
		if column.Name == name {
			return i
		}
	}

	return -1
}

// skillColumnOptions lists the table C columns, plus the character's Major
// and Minor where the chart allows taking one instead (chart 02 table C).
func (r *careerRun) skillColumnOptions() []string {
	columns := r.def.SkillColumnNames()
	if !r.def.MajorOrMinorColumn {
		return columns
	}

	options := slices.Clone(columns)

	for _, name := range []string{r.major(), r.minor()} {
		if name != "" {
			options = append(options, name)
		}
	}

	return options
}

// awardMajorOrMinor awards a level in the named Major or Minor, taken
// instead of a table C roll. The selecting choice is the cause
// (docs/PRD.md FR10).
func (r *careerRun) awardMajorOrMinor(name string, cause int) error {
	if name == "" {
		return fmt.Errorf("%w: no Major or Minor to take", errNotImplemented)
	}

	r.awardAndLog(name, 1, cause)

	return nil
}

// awardTableC applies one career skills table cell.
//
//nolint:exhaustive // Deliberately partitioned: the open-selection kinds are handled by awardOpenCell.
func (r *careerRun) awardTableC(entry career.Entry, cause int) error {
	switch entry.Kind {
	case career.EntrySkill:
		name, err := r.resolveSkillName(entry.Name)
		if err != nil {
			return err
		}

		r.awardAndLog(name, 1, cause)
	case career.EntryCharacteristic:
		return r.awardCharacteristic(entry.Name, cause)
	case career.EntryMajor, career.EntryMinor:
		// "If the character does not have a Major/Minor this benefit is
		// lost." (p. 78) The current Major/Minor are the most recent ones
		// selected (p. 59).
		name := r.major()
		if entry.Kind == career.EntryMinor {
			name = r.minor()
		}

		if name == "" {
			r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceBenefitLost})

			return nil
		}

		r.awardAndLog(name, 1, cause)
	case career.EntryNone:
		r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceNoAward})
	default:
		// The remaining kinds select from a Master Skill List group, or
		// wait on a later milestone; an unknown kind is rejected there
		// rather than silently resolving to nothing.
		return r.awardOpenCell(entry.Kind)
	}

	return nil
}

// characteristicRaiser is the optional careerMechanics seam for a career
// whose rules react to a table C characteristic award. Chart 11's "Each
// increase in Soc during CharGen awards a Land Grant" is the only one so
// far (interpretation I-30, ERRATA.md); the notification carries the
// awarding cause so the consequence chains to it (docs/PRD.md FR10).
type characteristicRaiser interface {
	characteristicRaised(r *careerRun, name string, cause int)
}

// awardCharacteristic applies a Personal-column +1, subject to the p. 68
// maximum (awardCharacteristicAndLog), and notifies the career's mechanics
// when the increase actually lands.
func (r *careerRun) awardCharacteristic(name string, cause int) error {
	before, ok := characteristicValue(&r.character.Characteristics, name)
	if !ok {
		return fmt.Errorf("%w: %q", errUnknownCharacteristic, name)
	}

	awardCharacteristicAndLog(r.character, r.log, name, 1, cause)

	after, _ := characteristicValue(&r.character.Characteristics, name)
	if after == before {
		// The p. 68 maximum refused the increase; nothing was raised.
		return nil
	}

	if raiser, ok := r.mechanics.(characteristicRaiser); ok {
		raiser.characteristicRaised(r, name, cause)
	}

	return nil
}

// continueRoll rolls Continue (for Citizen, "Continue 10-", chart 04):
// "the Character must successfully roll (2D) to Continue (or less) in the
// career. Failure ends Career Resolution. ... If the Continue roll is 2
// exactly, the character is required to Continue" (p. 66). Each term
// elapses 4 years ("the 4-year Term", p. 66).
func (r *careerRun) continueRoll() (bool, error) {
	target := r.def.ContinueTarget
	label := "Continue " + strconv.Itoa(target) + "-"

	switch {
	case r.def.ContinueCharacteristic != "":
		// A characteristic Continue target (chart 05: "Continue Int").
		target, _ = characteristicValue(&r.character.Characteristics, r.def.ContinueCharacteristic)
		label = "Continue " + r.def.ContinueCharacteristic
	case r.def.ContinueFame:
		// The career's own tracked value (chart 03: "Continue Fame").
		target = r.record.Fame
		label = "Continue Fame"
	case r.def.ContinueCC:
		// The career-long controlling characteristic (chart 10:
		// "Continue CC*").
		target, _ = characteristicValue(&r.character.Characteristics, r.record.ControllingCharacteristic)
		label = "Continue " + r.record.ControllingCharacteristic
	}

	bonus, mods, suffix := r.continueMod()
	target += bonus
	label += suffix

	throw := r.roller.Check(2, target)
	seq := r.log.Throw(throw, mods, r.def.Cite+" ("+label+"; p. 66)")

	if err := r.character.advanceYears(TermYears, r.roller, r.log, seq); err != nil {
		return false, err
	}

	if throw.Total == 2 {
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceMandatoryContinue})

		return true, nil
	}

	if throw.Success {
		return true, nil
	}

	if r.def.ContinueWaiver {
		// "An adverse die roll or decision (in ... Continue) may be
		// waived" (chart 02 Waivers, p. 76): the waiver negates the
		// failure, so the career continues into the next term.
		waived, err := r.waive(careerWaiver("Continue: the career would end", r.def.Cite, true), seq)
		if err != nil {
			return false, err
		}

		if waived {
			return true, nil
		}
	}

	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceCareerEnded, Career: r.def.Name})

	return false, nil
}

// continueMod applies the career-tracked value a chart adds to its
// Continue target: chart 02's "*Mod +Pubs" and chart 09's "*Mod +Terms".
func (r *careerRun) continueMod() (int, []Mod, string) {
	switch r.def.ContinueMod {
	case career.ContinueModPublications:
		if r.record.Publications == 0 {
			return 0, nil, ""
		}

		return r.record.Publications,
			[]Mod{{Name: "Publications", Value: r.record.Publications}}, " +Pubs"
	case career.ContinueModTerms:
		// Completed terms, as the p. 66 worked example counts them
		// (interpretation I-12).
		terms := len(r.record.Terms)
		if terms == 0 {
			return 0, nil, ""
		}

		return terms, []Mod{{Name: "Terms", Value: terms}}, " +Terms"
	default:
		return 0, nil, ""
	}
}

// chooseCheckCharacteristic presents a check's stated characteristics
// (score-guided, like the education checks) and returns the chosen name
// and roll-low target. An empty list is a data error, not an "Auto" berth:
// the loader permits a begin track with no checks (chart 06's "To Begin
// Temp Auto"), so the caller must handle that case before calling here.
func chooseCheckCharacteristic(r *careerRun, names []string) (string, int, error) {
	if len(names) == 0 {
		return "", 0, fmt.Errorf("%w: no characteristic stated for the check", errUnknownCharacteristic)
	}

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

// riskModOptions are the mod alternatives every Risk & Reward chart
// offers: "Select Caution,
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

// chooseRiskMod resolves the Caution/Bravery/No Mod selection (p. 65);
// chartCite names the career chart printing the box.
func chooseRiskMod(r *careerRun, chartCite string) (int, error) {
	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseRiskMod,
		Prompt:  "Select Caution, Bravery, or No Mod",
		Options: riskModOptions,
		Cite:    "Book 1 p. 65 (Caution, Bravery, or No Mod); " + chartCite,
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

// injury resolves a Risk failure, shared by every career whose chart
// prints the same text: "Risk Failure: Reduce CC by negative Mods and Flux
// (CC may not be increased). If CC is reduced by 4 or more, then he is
// disabled. Muster Out at Term end with Double Benefits." (charts 05
// p. 79, 06 p. 80, 09 p. 83; wound badge, disabled, and dead per p. 65.)
// The caller supplies its chart's citation. Reports died, then disabled.
func (r *careerRun) injury(cc string, mod, cause int, cite string) (bool, bool) {
	flux := r.roller.Flux()
	r.log.Flux(flux, cite)

	delta := min(mod, 0) + flux.Value
	if delta >= 0 {
		// "CC may not be increased"; an unreduced CC leaves the
		// character unharmed (p. 65).
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

// recordSanityMod charges the career's terms against Sanity where its
// chart says they cost some: "Because of the long-term isolation that a
// Scout must endure, reduce San= -1 for each TWO Terms served" (chart 05
// p. 79). Chart 05 is the only career that prints such a rule, so the
// count of terms per point is a chart fact in the career's data rather
// than a Scout branch here.
//
// The reduction is recorded, not applied. Sanity has no value to reduce:
// "Characters do not generate Sanity until it is first called for by a
// situation, encounter, or stimulus" (p. 52), and generating one here
// would both contradict that and consume the seeded stream for a value v1
// never reads (interpretation I-47, ERRATA.md).
//
// Partial terms owe nothing: two terms are what the chart charges for, so
// three terms cost the same as two.
func (r *careerRun) recordSanityMod(cause int) {
	if r.def.SanityPerTerms == 0 {
		return
	}

	mod := -(len(r.record.Terms) / r.def.SanityPerTerms)
	if mod == 0 {
		return
	}

	r.record.SanityMod = mod
	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceSanityMod, Career: r.def.Name, Delta: mod,
	})
}
