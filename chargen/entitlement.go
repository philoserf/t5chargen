package chargen

// The Automatics and the Entitlements of chart M1 (Book 1 p. 70; prose
// pp. 67, 69), which are the two of muster out's "three types of awards"
// that are not rolled for.

import (
	"fmt"
	"strconv"

	"github.com/philoserf/t5chargen/benefit"
	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/lifestage"
)

// The eligibilities chart M1 prints beside each Automatic, in the terms
// p. 67 gives them: "Award-Winning Scholar, Scout with 3 Discoveries,
// Craftsman with three perfect Masterpieces, and recipient of the
// Starburst for Extreme Heroism".
const (
	tasDiscoveries         = 3
	tasPerfectMasterpieces = 3
)

// armedForcesRetirement is the bar both Armed Forces entitlements set:
// "A Soldier, Spacer, or Marine who has served at least 4 terms" (p. 69).
const armedForcesRetirement = 4

// automatics records what a character already owns: "When a character ends
// character generation he may find that he already own some specific
// awards or items" (p. 67).
func (r *musterOutRun) automatics() {
	for _, a := range r.table.Automatics {
		if r.eligibleFor(a.ID) {
			r.character.Automatics = append(r.character.Automatics, a.Name)
		}
	}
}

// eligibleFor reads chart M1's eligibility column against the record.
func (r *musterOutRun) eligibleFor(id string) bool {
	c := r.character

	switch id {
	case "personal_weapons":
		// "A character who has received Fighter-1 or greater owns one
		// personal weapon ... This benefit does not apply for skills
		// other than Fighter" (p. 67).
		return c.skillLevel("Fighter") >= 1
	case "medal":
		// Chart M1 prints "Spacer or Soldier" and not Marine, though
		// chart 12 awards medals (interpretation I-78).
		return r.hasMedalFrom("Spacer") || r.hasMedalFrom("Soldier")
	case "commendations":
		return r.careerRecord("Agent") != nil && r.careerRecord("Agent").Commendations > 0
	case "fame":
		return c.Fame > 0
	case "tas_life":
		return r.tasEligible()
	default:
		return r.holdsSomething(id)
	}
}

// holdsSomething reads the Automatics whose eligibility is a count on the
// record: "Land Grants — Nobles, Scouts" and "Masterpieces — Craftsman"
// (chart M1 p. 70). A Scout's grants come one per Discovery (chart 05).
func (r *musterOutRun) holdsSomething(id string) bool {
	switch id {
	case "land_grants":
		return r.totalOf(func(rec CareerRecord) int { return rec.LandGrants }) > 0 ||
			r.totalOf(func(rec CareerRecord) int { return rec.Discoveries }) > 0
	case "masterpieces":
		return r.totalOf(func(rec CareerRecord) int { return rec.Masterpieces }) > 0
	default:
		return false
	}
}

// tasEligible reads p. 67's four examples: "Award-Winning Scholar, Scout
// with 3 Discoveries, Craftsman with three perfect Masterpieces, and
// recipient of the Starburst for Extreme Heroism".
func (r *musterOutRun) tasEligible() bool {
	if scholar := r.careerRecord("Scholar"); scholar != nil && scholar.Publications > 0 {
		return true
	}

	if r.totalOf(func(rec CareerRecord) int { return rec.Discoveries }) >= tasDiscoveries {
		return true
	}

	if r.totalOf(func(rec CareerRecord) int { return rec.PerfectMasterpieces }) >= tasPerfectMasterpieces {
		return true
	}

	for _, record := range r.character.Careers {
		for _, award := range record.Medals {
			if award.Code == "SEH" || award.Code == "*SEH*" {
				return true
			}
		}
	}

	return false
}

// hasMedalFrom reports whether a named career decorated the character.
func (r *musterOutRun) hasMedalFrom(name string) bool {
	record := r.careerRecord(name)

	return record != nil && len(record.Medals) > 0
}

// totalOf sums a counter across every career the character began.
func (r *musterOutRun) totalOf(count func(CareerRecord) int) int {
	total := 0

	for _, record := range r.character.Careers {
		if record.Began {
			total += count(record)
		}
	}

	return total
}

// entitlements settles the annual payments. "Pensions are available for
// Citizens, Functionaries, Reservists, and Professors. A pension begins
// when a character reaches Life Stage 9 Retirement (= age 66 for Humans)."
// The Armed Forces retirements begin at muster out (p. 69).
//
// "A character may receive duplicate Entitlements (for example, a Reserve
// and a Functionary pension, or both Military and Professor's retirement
// pay)", so they accumulate rather than replace — with the one exception
// the page names, the Functionary's pension replacing the Citizen's.
func (r *musterOutRun) entitlements() error {
	stages, err := lifestage.Load()
	if err != nil {
		return fmt.Errorf("entitlements: %w", err)
	}

	retirementAge := stages.FirstYearOf(stages.MentalStage)

	for _, id := range r.entitled() {
		e, ok := r.table.Entitlement(id)
		if !ok {
			return fmt.Errorf("%w: entitlement %q", errNotImplemented, id)
		}

		annual := r.annualFor(e)
		if annual == 0 {
			continue
		}

		from := retirementAge
		if e.FromLifeStage == 0 {
			from = r.character.Age
		}

		got := EntitlementRecord{
			ID: e.ID, Name: e.Name, AnnualCredits: annual,
			FromAge: from, CashOut: annual * r.table.CashOutYears,
		}

		// "Cashing Out. Any Entitlement can be cashed out for a lump sum
		// equal to five years of payments" (p. 69). The decision is also
		// what the record hangs from: an entitlement is read off the
		// finished character, and has no throw of its own to name.
		cashed, seq, err := r.offerCashOut(got)
		if err != nil {
			return err
		}

		if cashed {
			r.character.Credits += got.CashOut
		} else {
			r.character.Entitlements = append(r.character.Entitlements, got)
		}

		r.log.Consequence(ConsequenceEvent{
			Cause: seq, Kind: ConsequenceEntitlement, Skill: got.Name,
			Delta: got.AnnualCredits, Value: got.FromAge,
			Characteristic: cashOutNote(cashed, got.CashOut),
		})
	}

	return nil
}

// entitled lists the entitlement ids a character has earned.
func (r *musterOutRun) entitled() []string {
	var ids []string

	// "Any character who has been a Citizen ... receives a Citizen's
	// Pension", and the Functionary's "replaces a Citizen's pension, if
	// any" (p. 69).
	functionary := r.careerRecord("Functionary")
	if functionary != nil {
		ids = append(ids, "functionary_pension")
	} else if r.careerRecord("Citizen") != nil {
		ids = append(ids, "citizen_pension")
	}

	if r.reserveYears() > 0 {
		ids = append(ids, "reserve_pension")
	}

	if scholar := r.careerRecord("Scholar"); scholar != nil && scholar.Tenured {
		ids = append(ids, "professor_pension")
	}

	// "Retirement Pay based on total combined terms served" across the
	// three services (p. 69).
	if terms := r.serviceTerms(); terms >= armedForcesRetirement {
		if r.mustersOutAnOfficer() {
			ids = append(ids, "officer_retirement")
		} else {
			ids = append(ids, "enlisted_retirement")
		}
	}

	return ids
}

// annualFor prices one entitlement, applying the doublings muster out
// awarded: "Each doubling is of the original Pension: the first x2 doubles
// the Pension, the second x2 triples the pension" (p. 68).
func (r *musterOutRun) annualFor(e benefit.Entitlement) int {
	annual := e.AnnualCredits

	switch {
	case e.CreditsPerYear > 0:
		annual = e.CreditsPerYear * r.reserveYears()
	case e.CreditsPerTerm > 0:
		annual = e.CreditsPerTerm * r.serviceTerms()
	}

	kind := benefit.PensionDoubling
	if e.CreditsPerTerm > 0 {
		kind = benefit.RetirementDoubling
	}

	// n doublings give (n+1) times the original, not two to the n.
	doublings := 0

	for _, got := range r.character.Benefits {
		if got.Kind == kind {
			doublings += max(got.Count, 1)
		}
	}

	return annual * (doublings + 1)
}

// reserveYears counts the years to Life Stage 9 a character spends in the
// Reserves: "Cr 100 per Reserve year" (chart M1), from when the service
// ended (p. 67).
func (r *musterOutRun) reserveYears() int {
	stages, err := lifestage.Load()
	if err != nil {
		return 0
	}

	retirementAge := stages.FirstYearOf(stages.MentalStage)

	years := 0

	for _, record := range r.character.Careers {
		if record.Reserve {
			years += max(retirementAge-record.EndAge, 0)
		}
	}

	return years
}

// serviceTerms counts the Armed Forces terms a character served, combined
// across the three services (p. 69).
func (r *musterOutRun) serviceTerms() int {
	terms := 0

	for _, record := range r.character.Careers {
		if record.Reserve {
			terms += len(record.Terms)
		}
	}

	return terms
}

// mustersOutAnOfficer reports whether the character left the Armed Forces
// commissioned: Officer Retirement is for one "who musters out as an
// Officer" (p. 69).
func (r *musterOutRun) mustersOutAnOfficer() bool {
	for _, record := range r.character.Careers {
		if record.Reserve && r.isOfficer(record) {
			return true
		}
	}

	return false
}

// careerRecord finds the character's record for a career he entered.
func (r *musterOutRun) careerRecord(name string) *CareerRecord {
	for i := range r.character.Careers {
		if r.character.Careers[i].Career == name && r.character.Careers[i].Began {
			return &r.character.Careers[i]
		}
	}

	return nil
}

// isOfficer reports whether a career record ended on the officer ladder,
// read from the rank's class rather than from its id, which is the idiom
// the rest of the engine uses.
func (r *musterOutRun) isOfficer(record CareerRecord) bool {
	def, err := career.ByName(record.Career)
	if err != nil {
		return false
	}

	rank, ok := def.RankByID(record.Rank)

	return ok && rank.Class == officerRankClass
}

// offerCashOut puts p. 69's choice and reports whether it was taken, and
// the choice event to hang the entitlement from.
func (r *musterOutRun) offerCashOut(got EntitlementRecord) (bool, int, error) {
	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:     ChooseCashOut,
		Prompt: "Cash out the " + got.Name + "?",
		Options: []string{
			"Keep it (Cr" + itoa(got.AnnualCredits) + " a year from age " + itoa(got.FromAge) + ")",
			"Cash out (Cr" + itoa(got.CashOut) + ")",
		},
		Scores: []int{got.AnnualCredits, got.CashOut},
		Cite:   "Book 1 p. 69 (Cashing Out)",
	})
	if err != nil {
		return false, 0, err
	}

	return chosen == 1, seq, nil
}

// cashOutNote marks a cashed-out entitlement in its consequence.
func cashOutNote(cashed bool, lump int) string {
	if !cashed {
		return ""
	}

	return "cashed out for Cr" + itoa(lump)
}

// itoa is strconv.Itoa, named short because these prompts read better
// without the package noise.
func itoa(n int) string { return strconv.Itoa(n) }

// careerAutomatics awards what a career's own table D grants outright,
// beyond chart M1's list: chart 13 prints two, "Automatic: Gold Watch
// (Value= 100 x Terms as Functionary)" and "Automatic: Directorship if
// Rank F6+" (p. 87).
func (r *musterOutRun) careerAutomatics(record *CareerRecord, def *career.Definition) {
	for _, a := range def.MusterOut.Automatics {
		if a.MinimumRank != "" && rankNumber(record.Rank) < rankNumber(a.MinimumRank) {
			continue
		}

		name := a.Name

		if a.CreditsPerTerm > 0 {
			credits := a.CreditsPerTerm * len(record.Terms)
			r.character.Credits += credits
			name += " (Cr" + itoa(credits) + ")"
		}

		r.character.Automatics = append(r.character.Automatics, name)
	}
}
