package chargen

// Fame (Book 1 chart F, p. 91).
//
// Fame is calculated, not accumulated: "Current Fame for an individual is
// based on a variety of accomplishments. For example, Rogue with one
// Failed Scheme (and no other applicable factors) has Fame = 1 x 3 = 3."
// The careers count occurrences as they happen; chart F prices them once,
// over the finished record.
//
// "xN = N Fame points per occurrence."

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/fame"
)

// famePoints is one priced accomplishment, kept itemized so the transcript
// can show the arithmetic rather than a bare total (docs/PRD.md FR10).
type famePoints struct {
	source string
	points int

	// unit is one occurrence's worth, which is what "the highest Fame"
	// compares (interpretation I-63). For a "xN" line the chart states it
	// outright — "xN = N Fame points per occurrence" — so a Scout's six
	// Discoveries are six units of 4, not one of 24. For a line that
	// names a value outright ("=Rank", "Soc x1.5") the whole is the unit.
	unit int
}

// units expands a priced line into the individual Fame points it is made
// of, which is what chart F's stacking rule operates on.
func (f famePoints) units() []int {
	// points as well as unit: make's capacity argument is f.points/f.unit,
	// which panics on a negative. No call site passes one — the two
	// eligibility builders append only on points != 0 — and this is the
	// floor that says so rather than leaving it to them.
	if f.unit <= 0 || f.points <= 0 {
		return []int{f.points}
	}

	out := make([]int, 0, f.points/f.unit)
	for total := 0; total+f.unit <= f.points; total += f.unit {
		out = append(out, f.unit)
	}

	return out
}

// computeFame prices every accomplishment on the record and stacks them.
func computeFame(c *Character, roller *dice.Roller, log *Log, decider Decider) error {
	table := fame.Load()

	earned := make([]famePoints, 0, len(c.Careers))

	for _, record := range c.Careers {
		if !record.Began {
			continue
		}

		def, lookupErr := career.ByName(record.Career)
		if lookupErr != nil {
			return fmt.Errorf("fame: %w", lookupErr)
		}

		lines, err := careerFame(table, record, def, c)
		if err != nil {
			return err
		}

		earned = append(earned, lines...)
	}

	// "If NO other eligibility, 1D." A character who has done nothing the
	// chart prices is still known to someone.
	if len(earned) == 0 {
		roll := roller.Roll(table.NoEligibilityDice)
		log.Roll(roll, table.Cite+" (If NO other eligibility, 1D)")

		earned = append(earned, famePoints{source: "no other eligibility", points: roll.Total})
	}

	points := make([]int, 0, len(earned))
	mods := make([]Mod, 0, len(earned)+1)

	for _, e := range earned {
		// The transcript shows one line per eligibility; the stacking
		// rule works on the individual Fame points those lines are made
		// of (interpretation I-63).
		points = append(points, e.units()...)
		mods = append(mods, Mod{Name: e.source, Value: e.points})
	}

	// Stack the accomplishments before the Flux Event. The limit governs
	// "the sum of all Fame points received", and Flux is not one of them:
	// the chart says "add Flux to Fame", so it applies to the Fame those
	// points stack to. Stacking it alongside them would let the "only the
	// highest Fame applies" clause swallow a loss whenever one eligibility
	// dominates the total, and a symmetric gamble would only ever pay.
	base := table.Stack(points)

	// "The Fame Flux Event. Any character may choose (once during
	// Character Generation or after adventuring begins) to add Flux to
	// Fame." Offered here because it is the last thing that can change
	// the total, and because a calculation over a finished record has no
	// throw of its own to name as the cause of what it computes
	// (docs/PRD.md FR10).
	flux, cause, err := offerFameFlux(table, base, roller, log, decider)
	if err != nil {
		return err
	}

	if flux != 0 {
		mods = append(mods, Mod{Name: "Fame Flux Event", Value: flux})
	}

	c.Fame = max(base+flux, 0)
	log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceFameComputed,
		Value: c.Fame, Skill: table.Descriptor(c.Fame), Mods: mods,
	})

	return nil
}

// offerFameFlux puts the once-per-character gamble to the decider and
// reports the Flux it won or lost, and the choice event to hang the
// computed Fame from.
//
// The stacked Fame so far is offered as a score so a policy can weigh the
// gamble without recomputing it: Flux is symmetric, so the only thing it
// can buy is crossing a threshold that matters — and p. 68's "one
// additional roll if Fame 19+" is the one this milestone knows about.
func offerFameFlux(
	table *fame.Table, base int, roller *dice.Roller, log *Log, decider Decider,
) (int, int, error) {
	chosen, seq, err := choose(log, decider, Choice{
		ID:      ChooseFameFlux,
		Prompt:  "Invoke the Fame Flux Event? (Fame " + strconv.Itoa(base) + " so far)",
		Options: []string{"Keep this Fame", "Add Flux to Fame"},
		Scores:  []int{base, base},
		Cite:    table.Cite + " (The Fame Flux Event)",
	})
	if err != nil {
		return 0, 0, err
	}

	if chosen == 0 {
		return 0, seq, nil
	}

	flux := roller.Flux()
	log.Flux(flux, table.Cite+" (The Fame Flux Event)")

	return flux.Value, seq, nil
}

// fameScope is one career record being priced: what the eligibility rules
// read, and the lines they have earned so far.
type fameScope struct {
	table     *fame.Table
	record    CareerRecord
	def       *career.Definition
	character *Character
	earned    []famePoints
}

// add records a line worth points in total, made of units of `unit` each;
// unit 0 means the line is a single amount.
func (s *fameScope) add(source string, points, unit int) {
	if points != 0 {
		s.earned = append(s.earned, famePoints{
			source: s.record.Career + " " + source, points: points, unit: unit,
		})
	}
}

// perOccurrence records a "xN" line: "xN = N Fame points per occurrence".
func (s *fameScope) perOccurrence(line string, count, n int) {
	s.add(line+" x"+strconv.Itoa(n), count*n, n)
}

// careerFameRules prices each career's eligibilities on chart F. Its key
// set must match career.Available (tested), so a career the chart prices
// cannot fall through to pricing nothing by a misspelt name.
var careerFameRules = map[string]func(*fameScope){
	// "Citizen no intrinsic Fame."
	"Citizen": func(*fameScope) {},
	// "Scholar =Rank / Scholar =Publications".
	"Scholar": func(s *fameScope) {
		s.add("Rank", rankNumber(s.record.Rank), 0)
		s.add("Publications", s.record.Publications, 0)
	},
	// "Imperial Noble Soc x1.5 / Imperial Noble Per Exile +1".
	"Noble": func(s *fameScope) {
		s.add("Soc x1.5", nobleFame(s.table, s.character.Characteristics.Soc), 0)
		perExile := s.table.Careers.Noble.PerExile
		s.add("per Exile", s.record.TimesExiled*perExile, perExile)
	},
	// Chart F does not list the Functionary at all, and chart 13 is not an
	// Armed Force, so it earns nothing.
	"Functionary": func(*fameScope) {},
	// "Craftsman Masterpieces x3 / Craftsman Perfect Masterpieces x5".
	// A Perfect Masterpiece is counted in both, being a Masterpiece.
	"Craftsman": func(s *fameScope) {
		craftsman := s.table.Careers.Craftsman
		s.perOccurrence("Masterpieces", s.record.Masterpieces, craftsman.Masterpieces)
		s.perOccurrence("Perfect Masterpieces", s.record.PerfectMasterpieces, craftsman.PerfectMasterpieces)
	},
	"Soldier": armedForcesFame,
	"Spacer":  armedForcesFame,
	"Marine":  armedForcesFame,
	// "Agent =Number of Commendations".
	"Agent": func(s *fameScope) {
		s.add("Commendations", s.record.Commendations, 0)
	},
	"Rogue": rogueFame,
	// "Entertainer detailed under Career" (chart 03 tracks its own).
	// Chart 03's Fame is a Flux-driven running level that can fall below
	// zero; a career the character ended unknown for contributes nothing
	// rather than subtracting from the others (interpretation I-68).
	"Entertainer": func(s *fameScope) {
		s.add("Fame", max(s.record.Fame, 0), 0)
	},
	// "Scout Discoveries x4".
	"Scout": func(s *fameScope) {
		s.perOccurrence("Discoveries", s.record.Discoveries, s.table.Careers.Scout.Discoveries)
	},
	// "Merchant =Rank". "Merchant Ship Owner = 1D" is deferred: ownership
	// is settled at muster out (interpretation I-64).
	"Merchant": func(s *fameScope) {
		s.add("Rank", rankNumber(s.record.Rank), 0)
	},
}

// careerFame prices what one career contributed (chart F's eligibility
// column).
func careerFame(table *fame.Table, record CareerRecord, def *career.Definition, c *Character) ([]famePoints, error) {
	rule, ok := careerFameRules[record.Career]
	if !ok {
		return nil, fmt.Errorf("fame: %w: %q", errUnregisteredCareer, record.Career)
	}

	s := &fameScope{table: table, record: record, def: def, character: c}
	rule(s)

	return s.earned, nil
}

// rogueFame prices chart F's two Rogue lines and chart 10's infamy.
func rogueFame(s *fameScope) {
	// "Rogue Successful Schemes x2 / Rogue Failed Schemes x3" (chart F).
	rogue := s.table.Careers.Rogue
	s.perOccurrence("Successful Schemes", s.record.SuccessfulSchemes, rogue.SuccessfulSchemes)
	s.perOccurrence("Failed Schemes", s.record.FailedSchemes, rogue.FailedSchemes)

	// "Fame +1 (actually Infamy)" per imprisonment (chart 10 p. 84),
	// which chart F does not enumerate (interpretation I-67).
	s.add("Infamy", s.record.Fame, 1)
}

// armedForcesFame prices "Army / Marine / Navy: Officer Rank *" and the
// decorations beneath it, subject to the footnote "*Armed Forces Enlisted
// = no Fame" (interpretation I-65).
func armedForcesFame(s *fameScope) {
	if !s.def.Reserves {
		return
	}

	rank, ok := s.def.RankByID(s.record.Rank)
	if !ok || rank.Class != "officer" {
		// "*Armed Forces Enlisted = no Fame."
		return
	}

	s.add("Officer Rank", rankNumber(s.record.Rank), 0)

	// "Wound Badge WB x1." A Wound Badge is not a Medal — the Risk
	// failure awards it, not the Reward success ("If the Soldier, Spacer,
	// or Marines Risk Roll fails, the character is wounded and receives a
	// Wound Badge (WB)", p. 91) — so it is counted on the record rather
	// than listed in Medals, and priced here from the same Armed Forces
	// block of chart F.
	if points, priced := s.table.MedalPoints(woundBadgeCode); priced {
		s.perOccurrence(woundBadgeCode, s.record.WoundBadges, points)
	}

	for _, award := range s.record.Medals {
		points, priced := s.table.MedalPoints(award.Code)
		if !priced {
			// A code chart F does not price is a transcription fault,
			// not a medal worth nothing. audit's cross-check holds
			// fame.json's keys to medals.json's codes plus WB, so this
			// is unreachable; skipping it silently is what made the two
			// cases indistinguishable before that gate existed.
			continue
		}

		if points == 0 {
			// Exemplary Service is listed at x0.
			continue
		}

		s.perOccurrence(award.Code, award.Count, points)
	}
}

// woundBadgeCode is chart F's code for the Wound Badge ("Wound Badge WB
// x1"), which the Medals table prices alongside the decorations.
const woundBadgeCode = "WB"

// nobleFame prices "Imperial Noble Soc x1.5". The chart prints no rounding
// rule for the half point; it rounds down (interpretation I-66).
func nobleFame(table *fame.Table, soc int) int {
	return int(math.Floor(float64(soc) * table.NobleSocMultiplier))
}

// rankNumber reads the numeric part of a rank id ("O4" is 4, "F7" is 7),
// which is what chart F's "=Rank" eligibilities count.
func rankNumber(id string) int {
	n, err := strconv.Atoi(strings.TrimLeft(id, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	if err != nil {
		return 0
	}

	return n
}
