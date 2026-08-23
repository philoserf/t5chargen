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
}

// computeFame prices every accomplishment on the record and stacks them.
func computeFame(c *Character, roller *dice.Roller, log *Log, decider Decider) error {
	table, err := fame.Load()
	if err != nil {
		return fmt.Errorf("fame: %w", err)
	}

	earned := make([]famePoints, 0, len(c.Careers))

	for _, record := range c.Careers {
		if !record.Began {
			continue
		}

		def, err := career.ByName(record.Career)
		if err != nil {
			return fmt.Errorf("fame: %w", err)
		}

		earned = append(earned, careerFame(table, record, def, c)...)
	}

	// "If NO other eligibility, 1D." A character who has done nothing the
	// chart prices is still known to someone.
	if len(earned) == 0 {
		roll := roller.Roll(table.NoEligibilityDice)
		log.Roll(roll, table.Cite+" (If NO other eligibility, 1D)")

		earned = append(earned, famePoints{source: "no other eligibility", points: roll.Total})
	}

	// "The Fame Flux Event. Any character may choose (once during
	// Character Generation or after adventuring begins) to add Flux to
	// Fame." Offered here because it is the last thing that can change
	// the total, and because a calculation over a finished record has no
	// throw of its own to name as the cause of what it computes
	// (docs/PRD.md FR10).
	flux, cause, err := offerFameFlux(table, earned, roller, log, decider)
	if err != nil {
		return err
	}

	if flux != 0 {
		earned = append(earned, famePoints{source: "Fame Flux Event", points: flux})
	}

	points := make([]int, 0, len(earned))
	mods := make([]Mod, 0, len(earned))

	for _, e := range earned {
		points = append(points, e.points)
		mods = append(mods, Mod{Name: e.source, Value: e.points})
	}

	c.Fame = max(table.Stack(points), 0)
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
// The base total is offered as a score so a policy can weigh the gamble
// without recomputing it: Flux is symmetric, so the only thing it can buy
// is crossing a threshold that matters — and p. 68's "one additional roll
// if Fame 19+" is the one this milestone knows about.
func offerFameFlux(
	table *fame.Table, earned []famePoints, roller *dice.Roller, log *Log, decider Decider,
) (int, int, error) {
	base := 0
	for _, e := range earned {
		base += e.points
	}

	chosen, seq, err := choose(log, decider, Choice{
		ID:      ChooseFameFlux,
		Prompt:  "Invoke the Fame Flux Event? (Fame " + strconv.Itoa(min(base, table.StackLimit)) + " so far)",
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

// careerFame prices what one career contributed (chart F's eligibility
// column).
func careerFame(table *fame.Table, record CareerRecord, def *career.Definition, c *Character) []famePoints {
	var earned []famePoints

	add := func(source string, points int) {
		if points != 0 {
			earned = append(earned, famePoints{source: record.Career + " " + source, points: points})
		}
	}

	switch record.Career {
	case "Craftsman":
		// "Craftsman Masterpieces x3 / Craftsman Perfect Masterpieces x5".
		// A Perfect Masterpiece is counted in both, being a Masterpiece.
		add("Masterpieces x3", record.Masterpieces*3)
		add("Perfect Masterpieces x5", record.PerfectMasterpieces*5)
	case "Scholar":
		// "Scholar =Rank / Scholar =Publications".
		add("Rank", rankNumber(record.Rank))
		add("Publications", record.Publications)
	case "Rogue":
		rogueFame(add, record)
	case "Noble":
		// "Imperial Noble Soc x1.5 / Imperial Noble Per Exile +1".
		add("Soc x1.5", nobleFame(table, c.Characteristics.Soc))
		add("per Exile", record.TimesExiled)
	default:
		singleLineFame(add, record)
		// Spacer, Soldier, Marine; the Functionary earns none.
		earned = append(earned, armedForcesFame(table, record, def)...)
	}

	return earned
}

// singleLineFame prices the careers chart F gives one eligibility each,
// and the two it gives none: "Citizen no intrinsic Fame", and the
// Functionary, which the chart does not list at all.
func singleLineFame(add func(string, int), record CareerRecord) {
	switch record.Career {
	case "Entertainer":
		// "Entertainer detailed under Career" (chart 03 tracks its own).
		add("Fame", record.Fame)
	case "Scout":
		// "Scout Discoveries x4".
		add("Discoveries x4", record.Discoveries*4)
	case "Merchant":
		// "Merchant =Rank". "Merchant Ship Owner = 1D" is deferred:
		// ownership is settled at muster out (interpretation I-64).
		add("Rank", rankNumber(record.Rank))
	case "Agent":
		// "Agent =Number of Commendations".
		add("Commendations", record.Commendations)
	}
}

// rogueFame prices chart F's two Rogue lines and chart 10's infamy.
func rogueFame(add func(string, int), record CareerRecord) {
	// "Rogue Successful Schemes x2 / Rogue Failed Schemes x3" (chart F).
	add("Successful Schemes x2", record.SuccessfulSchemes*2)
	add("Failed Schemes x3", record.FailedSchemes*3)

	// "Fame +1 (actually Infamy)" per imprisonment (chart 10 p. 84),
	// which chart F does not enumerate (interpretation I-67).
	add("Infamy", record.Fame)
}

// armedForcesFame prices "Army / Marine / Navy: Officer Rank *" and the
// decorations beneath it, subject to the footnote "*Armed Forces Enlisted
// = no Fame" (interpretation I-65).
//
// The Functionary reaches here too and earns nothing: chart F prices no
// Functionary accomplishment, and chart 13 is not an Armed Force.
func armedForcesFame(table *fame.Table, record CareerRecord, def *career.Definition) []famePoints {
	if !def.Reserves {
		return nil
	}

	rank, ok := def.RankByID(record.Rank)
	if !ok || rank.Class != "officer" {
		// "*Armed Forces Enlisted = no Fame."
		return nil
	}

	earned := []famePoints{{source: record.Career + " Officer Rank", points: rankNumber(record.Rank)}}

	for _, award := range record.Medals {
		points, priced := table.MedalPoints(award.Code)
		if !priced || points == 0 {
			// Exemplary Service is listed at x0.
			continue
		}

		earned = append(earned, famePoints{
			source: record.Career + " " + award.Code + " x" + strconv.Itoa(points),
			points: award.Count * points,
		})
	}

	return earned
}

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
