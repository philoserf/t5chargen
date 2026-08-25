package chargen

// Mustering Out (Book 1 pp. 67-69; chart M1 p. 70; each career's table D).
//
// "Mustering Out counts up the character's belongings (at least the major
// ones), the money, and the abilities that a character has accumulated
// through several years of career and notes them as assets for the
// adventuring situations to come." (p. 67)
//
// It runs once, over the finished record, rather than at each career's
// end. p. 68 forecloses the alternative twice: "If the character later
// served as a Functionary associated with that Career, add those
// Functionary Terms to the DM" makes a career's DM depend on careers
// served after it, and the worked example runs three careers and musters
// out once.
//
// Deferred to the next chunk: the Automatics list and the Entitlements,
// both of chart M1.

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/philoserf/t5chargen/benefit"
	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/dice"
)

// benefitDice is the throw: "Roll 1D + DMs" (p. 68).
const benefitDice = 1

// officerRankClass is the rank class the Armed Forces ladders and chart
// 06's give their commissioned ranks.
const officerRankClass = "officer"

// fameBonusRoll is the Fame that earns one: "He is allowed one additional
// roll if Fame 19+" (p. 68).
const fameBonusRoll = 19

// bonusRollMedals are the decorations that each earn a roll: "He is
// allowed one additional roll per Commendation, MCG, or SEH" (p. 68).
// *SEH* is an SEH with diamonds and counts as one.
var bonusRollMedals = map[string]bool{"MCG": true, "SEH": true, "*SEH*": true}

// rerollable are the benefits a duplicate of which is worth nothing:
// "A result that duplicates a previous (unwanted or unusable) benefit may
// be rerolled until a different benefit is received, for example: Wafer
// Jack, TAS Member, Knighthood" (p. 69).
var rerollable = map[benefit.Kind]bool{
	benefit.WaferJack:     true,
	benefit.TASLife:       true,
	benefit.TASFellowship: true,
	benefit.Knighthood:    true,
	benefit.LifeInsurance: true,
}

// musterOutRun carries what one muster out needs.
type musterOutRun struct {
	character *Character
	roller    *dice.Roller
	log       *Log
	decider   Decider
	table     *benefit.Table
	held      map[benefit.Kind]bool
}

// musterOut resolves the whole of a character's muster out.
func musterOut(c *Character, roller *dice.Roller, log *Log, decider Decider) error {
	table, err := benefit.Load()
	if err != nil {
		return fmt.Errorf("muster out: %w", err)
	}

	run := &musterOutRun{
		character: c, roller: roller, log: log, decider: decider,
		table: table, held: map[benefit.Kind]bool{},
	}

	// The Fame-19+ roll is the character's, not a career's: "A character
	// with a roll allowed by Fame-19+ may select which career-dictated
	// table to use" (p. 68). It goes to the first career served, which
	// the policy would select anyway (interpretation I-73).
	bonus := 0
	if c.Fame >= fameBonusRoll {
		bonus = 1
	}

	for i := range c.Careers {
		record := &c.Careers[i]
		if !record.Began {
			continue
		}

		def, err := career.ByName(record.Career)
		if err != nil {
			return fmt.Errorf("muster out: %w", err)
		}

		if def.MusterOut == nil {
			return fmt.Errorf("%w: %q has no muster-out table", errNotImplemented, record.Career)
		}

		if err := run.career(record, def, rollsFor(record)+bonus); err != nil {
			return err
		}

		run.careerAutomatics(record, def)

		bonus = 0
	}

	// The Rogue's payoff is money he already has: "Payoff= V x (1+CC-R+
	// Mods)" (chart 10 p. 84), counted up here with the rest.
	for _, record := range c.Careers {
		c.Credits += record.SchemePayoff
	}

	// The Automatics are read off the finished record rather than rolled
	// for, so they are recorded like the UPP and the Life Stage and emit
	// no events: there is no throw or choice to name as their cause
	// (docs/PRD.md FR10).
	if err := run.automatics(); err != nil {
		return err
	}

	return run.entitlements()
}

// rollsFor counts a career's muster-out rolls: "A character is allowed one
// Mustering Out roll for each term served in Career Resolution. He is
// allowed one additional roll per Commendation, MCG, or SEH" (p. 68).
//
// A Disability Muster Out doubles the count: "Double Benefits is twice the
// count of Benefits (rather than two of each Benefit)" (p. 69).
func rollsFor(record *CareerRecord) int {
	rolls := len(record.Terms) + record.Commendations

	for _, award := range record.Medals {
		if bonusRollMedals[award.Code] {
			rolls += award.Count
		}
	}

	if record.Disabled {
		rolls *= 2
	}

	return rolls
}

// career resolves one career's rolls against its own table D.
func (r *musterOutRun) career(record *CareerRecord, def *career.Definition, rolls int) error {
	if rolls == 0 {
		return nil
	}

	r.log.Step("Muster Out: "+record.Career, def.MusterOut.Cite)

	for nth := range rolls {
		// A career with several muster-out rolls asks the same two
		// questions for each of them; the count is what tells a player
		// which he is answering (Choice.Nth).
		if err := r.roll(record, def, nth+1, rolls); err != nil {
			return err
		}
	}

	return nil
}

// roll resolves one muster-out roll: the column, the DM, the throw, and
// what the cell awards.
func (r *musterOutRun) roll(record *CareerRecord, def *career.Definition, nth, of int) error {
	table := def.MusterOut

	column, _, err := r.chooseColumn(def, nth, of)
	if err != nil {
		return err
	}

	dm, err := r.chooseDM(record, def, column, nth, of)
	if err != nil {
		return err
	}

	cell, roll := r.throw(table, column, dm)

	// "A result that duplicates a previous (unwanted or unusable) benefit
	// may be rerolled until a different benefit is received" (p. 69).
	//
	// Until, not once. The loop is bounded by what the dice can actually
	// reach: a throw of 1D+DM clamped to the table can only land on a
	// handful of rows, and if every one of them holds the same duplicate
	// there is no different benefit to receive and the rule asks for
	// something the table cannot give (interpretation I-74).
	for r.isDuplicate(cell.Kind) && r.canDiffer(table, column, dm) {
		r.log.Consequence(ConsequenceEvent{
			Cause: roll, Kind: ConsequenceBenefitLost, Career: record.Career,
			Skill: string(cell.Kind),
		})

		cell, roll = r.throw(table, column, dm)
	}

	return r.award(record, def, cell, roll)
}

// throw rolls 1D + DM against a column and returns the cell it lands on.
// "If the roll is greater than the maximum value on the table, use the
// maximum value instead" (p. 68).
func (r *musterOutRun) throw(
	table *career.MusterOut, column string, dm int,
) (career.MusterOutCell, int) {
	roll := r.roller.RollMod(benefitDice, dm)
	seq := r.log.Roll(roll, table.Cite+" ("+column+" column, 1D+"+strconv.Itoa(dm)+")")

	index := min(max(roll.Total, 1), len(table.Rows)) - 1

	return columnCell(table.Rows[index], column), seq
}

// columnCell picks a row's cell for a named column.
func columnCell(row career.MusterOutRow, column string) career.MusterOutCell {
	switch column {
	case "Money":
		return row.Money
	case "Power":
		return *row.Power
	default:
		return row.Benefit
	}
}

// chooseColumn puts p. 68's per-roll decision: "Which Column? Character
// may select either the Money column or Benefits column for each roll."
// Chart 11 adds a third, Power, which only the Noble has — read off the
// table rather than off the career's name, so the third column is offered
// exactly where the data prints one (validateMusterOutRow ties a row's
// Power cell to the column's DM line, so an offered Power column always
// has a cell to land on).
func (r *musterOutRun) chooseColumn(def *career.Definition, nth, of int) (string, int, error) {
	options := []string{"Money", "Benefits"}
	if def.MusterOut.PowerDM != nil {
		options = append(options, "Power")
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseBenefitColumn,
		Prompt:  "Which column for this " + def.Name + " muster-out roll?",
		Options: options,
		Nth:     nth,
		Of:      of,
		Cite:    "Book 1 p. 68 (Which Column?)",
	})
	if err != nil {
		return "", 0, err
	}

	return options[chosen], seq, nil
}

// chooseDM puts the other per-roll decision. "The DMs on the Benefits
// Tables are optional (for Terms, Fame, or other values). They may be
// partially applied (any value may be selected from 0 to the total
// allowed value)" (p. 68).
func (r *musterOutRun) chooseDM(record *CareerRecord, def *career.Definition, column string, nth, of int) (int, error) {
	dm := def.MusterOut.BenefitDM
	if column == "Money" {
		dm = def.MusterOut.MoneyDM
	}

	if column == "Power" && def.MusterOut.PowerDM != nil {
		dm = *def.MusterOut.PowerDM
	}

	maximum := r.dmValue(record, dm)
	if maximum <= 0 {
		return 0, nil
	}

	options := make([]string, 0, maximum+1)
	scores := make([]int, 0, maximum+1)

	for value := range maximum + 1 {
		options = append(options, "+"+strconv.Itoa(value))
		scores = append(scores, value)
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseBenefitDM,
		Prompt:  "How much of the " + dm.Kind + " DM to apply?",
		Options: options,
		Scores:  scores,
		Nth:     nth,
		Of:      of,
		Cite:    "Book 1 p. 68 (the DMs ... may be partially applied)",
	})
	if err != nil {
		return 0, err
	}

	return chosen, nil
}

// dmValue reads what a DM line is worth for this character.
//
// "If the DM is Terms, use the number of Terms served in that Career. If
// the character later served as a Functionary associated with that Career,
// add those Functionary Terms to the DM" (p. 68).
func (r *musterOutRun) dmValue(record *CareerRecord, dm career.MusterOutDM) int {
	switch dm.Kind {
	case "terms":
		return len(record.Terms) + r.functionaryTerms(record.Career)
	case "total_terms":
		return totalTerms(r.character)
	case "officer_rank", "scholar_rank":
		// The number printed beside the rank's title, on whichever ladder
		// the character stands: an enlisted S4 counts 4, as I-68 reads
		// the same DM on chart 06 and as the Knighthood's officers-only
		// clause requires (interpretation I-75).
		return rankNumber(record.Rank)
	case "commendations":
		return record.Commendations
	case "fame":
		return r.character.Fame / dm.Divisor
	default:
		return 0
	}
}

// functionaryTerms counts the terms a character later served as a
// Functionary associated with the named career (p. 68).
func (r *musterOutRun) functionaryTerms(careerName string) int {
	terms := 0

	for _, record := range r.character.Careers {
		if record.Career == "Functionary" && record.AssociatedCareer == careerName {
			terms += len(record.Terms)
		}
	}

	return terms
}

// award applies what a cell gives and records it.
//
// Only a Money cell pays credits. p. 68 prices the rest — "StarPass ... has
// a value of Cr250,000" — but that is what one is worth if the character
// sells it, and cashing out is its own step: crediting a passage as money
// would both spend a ticket he still holds and count it twice, once in the
// money and once in the benefits.
func (r *musterOutRun) award(
	record *CareerRecord, def *career.Definition, cell career.MusterOutCell, cause int,
) error {
	entry, err := r.table.For(cell.Kind)
	if err != nil {
		return fmt.Errorf("muster out: %w", err)
	}

	got := BenefitRecord{Kind: cell.Kind, Name: entry.Name, Career: def.Name, Detail: cell.Detail}

	//nolint:exhaustive // Deliberately partitioned: default covers the rest.
	switch cell.Kind {
	case benefit.Money:
		got.Credits = cell.Credits
	case benefit.Characteristic:
		return r.awardCharacteristic(def, entry, cell, cause)
	case benefit.Knighthood:
		return r.awardKnighthood(record, def, entry, cause)
	case benefit.Fame:
		return r.awardFame(def, entry, cell, cause)
	default:
		got.Count = max(cell.Count, 1)

		if cell.Dice > 0 {
			roll := r.roller.Roll(cell.Dice)
			r.log.Roll(roll, entry.Name+" ("+strconv.Itoa(cell.Dice)+"D)")

			got.Count = roll.Total
		}
	}

	r.record(got, cause)

	return nil
}

// awardFame applies a "Fame +1" or "Fame +2" cell (charts 02, 03, 05 and
// 09 box D; interpretation I-72). A Fame cell has to move the counter or
// it awards nothing at all, which is the reading I-72 rejects. It is added
// to the Fame chart F has already computed (p. 91), muster out running
// after that step, so a Fame DM taken later in the same muster out sees
// the raised figure.
func (r *musterOutRun) awardFame(
	def *career.Definition, entry benefit.Benefit, cell career.MusterOutCell, cause int,
) error {
	got := max(cell.Count, 1)

	r.record(BenefitRecord{
		Kind: benefit.Fame, Name: entry.Name, Career: def.Name, Count: got,
	}, cause)

	r.character.Fame += got
	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceFameChange, Delta: got, Value: r.character.Fame,
	})

	return nil
}

// record adds a benefit to the character and logs it.
func (r *musterOutRun) record(got BenefitRecord, cause int) {
	r.held[got.Kind] = true
	r.character.Credits += got.Credits
	r.character.Benefits = append(r.character.Benefits, got)

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceBenefit, Career: got.Career,
		Skill: got.Name, Value: got.Count, Delta: got.Credits, Characteristic: got.Detail,
	})
}

// awardCharacteristic applies a "Str +1" cell, subject to p. 68's losses:
// "Characteristics for Humans cannot exceed 15. If a benefit elevates a
// characteristic above 15, that benefit is lost." The Caste clause — "If
// the improvement is C6+1 and for the character C6= Caste, the benefit is
// lost" — cannot arise for a human (docs/PRD.md non-goals).
func (r *musterOutRun) awardCharacteristic(
	def *career.Definition, entry benefit.Benefit, cell career.MusterOutCell, cause int,
) error {
	value, ok := characteristicValue(&r.character.Characteristics, cell.Detail)
	if !ok {
		return fmt.Errorf("%w: %q", errUnknownCharacteristic, cell.Detail)
	}

	// awardCharacteristicAndLog enforces the p. 68 maximum itself and
	// logs the loss. The check here is not the cap — it is what keeps a
	// benefit that was lost out of the record, so the sheet does not
	// credit the character with an improvement he never received.
	if value >= CharacteristicMax {
		awardCharacteristicAndLog(r.character, r.log, cell.Detail, 1, cause)

		return nil
	}

	awardCharacteristicAndLog(r.character, r.log, cell.Detail, 1, cause)
	r.record(BenefitRecord{
		Kind: cell.Kind, Name: entry.Name, Career: def.Name, Detail: cell.Detail, Count: 1,
	}, cause)

	return nil
}

// awardKnighthood applies p. 68's three clauses: "A Knighthood raises any
// value of Soc to B; if the character is already Soc 11+, he receives Soc
// +1 instead" and "In the Spacer, Soldier, and Marine careers, Knighthood
// is only available to Officers. A non-officer receives Soc +1".
func (r *musterOutRun) awardKnighthood(
	record *CareerRecord, def *career.Definition, entry benefit.Benefit, cause int,
) error {
	soc, _ := characteristicValue(&r.character.Characteristics, "Soc")

	knight := true

	if slices.Contains(entry.OfficersOnlyCareers, def.Name) {
		rank, ok := def.RankByID(record.Rank)
		knight = ok && rank.Class == officerRankClass
	}

	// A non-officer, or a character already at the floor, takes the
	// Social Standing instead of the title.
	if !knight || soc >= entry.SocFloor {
		bonus := entry.NonOfficerBonus
		if knight {
			bonus = entry.AboveFloorBonus
		}

		awardCharacteristicAndLog(r.character, r.log, "Soc", bonus, cause)

		// A Soc that cannot rise takes the benefit with it: "If a benefit
		// elevates a characteristic above 15, that benefit is lost"
		// (p. 68). awardCharacteristicAndLog logs the loss; nothing is
		// recorded, so the sheet does not credit a Social Standing the
		// character never received.
		if soc+bonus > CharacteristicMax {
			return nil
		}

		// What he received is the Social Standing, so that is what the
		// record says: "A non-officer receives Soc +1", and a character
		// already at the floor "receives Soc +1 instead" (p. 68).
		characteristic, err := r.table.For(benefit.Characteristic)
		if err != nil {
			return fmt.Errorf("muster out: %w", err)
		}

		r.record(BenefitRecord{
			Kind: benefit.Characteristic, Name: characteristic.Name,
			Career: def.Name, Detail: "Soc", Count: bonus,
		}, cause)

		return nil
	}

	awardCharacteristicAndLog(r.character, r.log, "Soc", entry.SocFloor-soc, cause)
	r.record(BenefitRecord{
		Kind: benefit.Knighthood, Name: entry.Name, Career: def.Name, Count: 1,
	}, cause)

	return nil
}

// isDuplicate reports whether a cell repeats a benefit already held that
// is worth nothing twice (p. 69).
func (r *musterOutRun) isDuplicate(kind benefit.Kind) bool {
	return rerollable[kind] && r.held[kind]
}

// canDiffer reports whether a reroll could land on anything the character
// does not already hold. The throw is 1D plus the DM, clamped to the last
// row, so the rows it can reach are a known span; if every one of them
// repeats something held, no number of rerolls will find "a different
// benefit" and the rule asks for what the table cannot give.
//
// The question is what the reachable rows offer, not merely whether they
// differ from each other: a span of three held duplicates differs plenty
// and still has nothing to give.
func (r *musterOutRun) canDiffer(table *career.MusterOut, column string, dm int) bool {
	for face := 1; face <= 6; face++ {
		row := table.Rows[min(max(face+dm, 1), len(table.Rows))-1]
		if !r.isDuplicate(columnCell(row, column).Kind) {
			return true
		}
	}

	return false
}
