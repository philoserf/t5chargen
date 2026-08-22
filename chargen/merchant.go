package chargen

// Merchant career mechanics (Book 1 chart 06, p. 80; prose pp. 64-66).
// "To Begin 4th Officer Int / To Begin Spacehand Dex / To Begin Temp Auto;
// Risk & Reward C1 C2 C3 C4; Officer Promotion Terms x2*; Officer
// Commission Int; Rating Promotion Dex*; Continue Str" with "*Mod +3 if
// Int 8+" (chart 06 box A).
//
// The Merchant is the first career with rank: entry lands on one of three
// tracks, and each term offers the commission and promotion attempts its
// current rank class allows — "Temp may attempt Officer Commission and
// Rating Promotion within the same Term. Rating may attempt Officer
// Commission and Rating promotion within the same Term. Officer may
// attempt Officer Promotion." (chart 06) Each rank gained earns an extra
// skill ("Promotion 1", table B) and any Auto Skill its rank row names.
//
// Risk & Reward is the shared shape (see careerRun.injury): "Risk Failure:
// Reduce CC by negative Mods and Flux ... If CC is reduced by 4 or more,
// then he is disabled." Reward success awards Ship Shares on the
// escalating scale ("Ship Share Rewards are awarded equal to the receipt
// number", chart 06).
//
// Deferred: ship-share economics and the muster-out table D (docs/PRD.md
// milestone 4); Double Benefits on disability (milestone 4 muster out).

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/philoserf/t5chargen/career"
)

// merchantMechanics is the Merchant careerMechanics implementation.
type merchantMechanics struct {
	// rank is the current rank id; empty until Begin succeeds.
	rank string

	// shipShareReceipts counts Reward successes, which set the size of
	// each award (chart 06 Escalating Ship Shares).
	shipShareReceipts int
}

// newMerchant is the Merchant careerRegistry entry.
//
//nolint:ireturn // The registry's function type returns the interface.
func newMerchant() (*career.Definition, careerMechanics, error) {
	def, err := career.Merchant()
	if err != nil {
		return nil, nil, fmt.Errorf("merchant career: %w", err)
	}

	return def, &merchantMechanics{}, nil
}

// begin selects an entry track and rolls its To Begin check. A failed
// attempt costs one year ("Each failed attempt (both Begin or Retry) takes
// one year", p. 65); chart 06 lists no Begin retry.
func (m *merchantMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Merchant: To Begin", r.def.Cite)

	track, trackSeq, err := m.chooseTrack(r)
	if err != nil {
		return false, err
	}

	// "To Begin Temp Auto" (chart 06): the untrained berth needs no throw,
	// so the selecting choice is what causes the entry (docs/PRD.md FR10).
	if len(track.Checks) == 0 {
		return true, m.enterRank(r, track.Rank, trackSeq)
	}

	check, value, err := chooseCheckCharacteristic(r, track.Checks)
	if err != nil {
		return false, err
	}

	throw := r.roller.Check(2, value)
	seq := r.log.Throw(throw, nil, r.def.Cite+" (To Begin "+track.Name+" vs "+check+")")

	if !throw.Success {
		// "Each failed attempt (both Begin or Retry) takes one year" (p. 65).
		r.character.Age++
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceYearsElapsed, Value: 1})
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceCareerNotBegun, Career: r.def.Name})

		return false, nil
	}

	return true, m.enterRank(r, track.Rank, seq)
}

// chooseTrack selects among the chart's entry paths, reporting the
// selecting choice's event sequence so the automatic berth's consequences
// can name their cause.
func (*merchantMechanics) chooseTrack(r *careerRun) (career.BeginTrack, int, error) {
	options := make([]string, len(r.def.BeginTracks))
	for i, track := range r.def.BeginTracks {
		options[i] = track.Name
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseBeginTrack,
		Prompt:  "Select the Merchant berth to begin in",
		Options: options,
		Cite:    r.def.Cite + " (To Begin 4th Officer / Spacehand / Temp)",
	})
	if err != nil {
		return career.BeginTrack{}, 0, err
	}

	return r.def.BeginTracks[chosen], seq, nil
}

// enterRank records a rank and awards its Auto Skill (chart 06 table B:
// "Automatic Skills by Rank"). Auto Skills are ordinary receipts —
// "a one-level increase if the skill is already held. If not, the
// character receives the skill at level-1" (p. 66).
func (m *merchantMechanics) enterRank(r *careerRun, id string, cause int) error {
	rank, ok := r.def.RankByID(id)
	if !ok {
		return fmt.Errorf("%w: %q", errUnknownRank, id)
	}

	m.rank = rank.ID
	r.record.Rank = rank.ID
	r.record.RankTitle = rank.Title

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceRankSet, Career: r.def.Name, Skill: rank.Title, Value: 0,
	})

	if rank.AutoSkill == "" {
		return nil
	}

	name, err := r.resolveSkillName(rank.AutoSkill)
	if err != nil {
		return err
	}

	r.awardAndLog(name, 1, cause)

	return nil
}

// resolveTerm runs Risk & Reward, then the advancement attempts the
// current rank class allows.
func (m *merchantMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	outcome, err := m.riskAndReward(r, cc)
	if err != nil || outcome.died {
		return outcome, err
	}

	gained, err := m.advance(r)
	if err != nil {
		return termOutcome{}, err
	}

	outcome.skillRolls = r.def.SkillsPerTerm + gained*r.def.SkillsPerAdvancement

	return outcome, nil
}

// riskAndReward runs the chart 06 Risk & Reward box.
func (m *merchantMechanics) riskAndReward(r *careerRun, cc string) (termOutcome, error) {
	var outcome termOutcome

	mod, err := chooseRiskMod(r, "chart 06 p. 80")
	if err != nil {
		return outcome, err
	}

	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return outcome, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	risk := r.roller.Check(2, value+mod)
	riskSeq := r.log.Throw(risk, riskMods(mod, 1), r.def.Cite+" (Risk vs "+cc+"+Mods)")

	if !risk.Success {
		died, disabled := r.injury(cc, mod, riskSeq,
			"Book 1 p. 80 chart 06 (Risk Failure: reduce CC by negative Mods and Flux)")
		if died {
			outcome.died = true

			return outcome, nil
		}

		outcome.endCareer = disabled
	}

	reward := r.roller.Check(2, value-mod)
	rewardSeq := r.log.Throw(reward, riskMods(mod, -1), r.def.Cite+" (Reward vs "+cc+"+ opposite sign Mods)")

	if !reward.Success {
		r.log.Consequence(ConsequenceEvent{Cause: rewardSeq, Kind: ConsequenceNoAward})

		return outcome, nil
	}

	outcome.success = true

	m.awardShipShares(r, rewardSeq)

	return outcome, nil
}

// awardShipShares applies the escalating scale: "Ship Share Rewards are
// awarded equal to the receipt number. First 1 ... Sixth 6" (chart 06).
// The chart stops at the sixth receipt; later receipts award six
// (interpretation I-14, ERRATA.md).
func (m *merchantMechanics) awardShipShares(r *careerRun, cause int) {
	m.shipShareReceipts++

	shares := min(m.shipShareReceipts, maxShipShareAward)
	r.record.ShipShares += shares

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceShipShares,
		Delta: shares, Value: r.record.ShipShares,
	})
}

// maxShipShareAward is the largest award the chart prints ("Sixth 6").
const maxShipShareAward = 6

// advance attempts each advancement row open to the rank class held at the
// start of the term, in chart order, and reports how many ranks were
// gained.
//
// Eligibility is tested against the rank held when the phase begins, not
// the rank as it changes: chart 06 enumerates what each class may attempt
// per term — "Temp may attempt Officer Commission and Rating Promotion
// within the same Term ... Officer may attempt Officer Promotion" — so a
// Temp commissioned this term does not also attempt Officer Promotion on
// the strength of the commission he just received. Each row still advances
// from the character's current rank, so a rating promoted this term is
// promoted from the rank he now holds.
func (m *merchantMechanics) advance(r *careerRun) (int, error) {
	entry, ok := r.def.RankByID(m.rank)
	if !ok {
		return 0, fmt.Errorf("%w: %q", errUnknownRank, m.rank)
	}

	gained := 0

	for _, advancement := range r.def.Advancements {
		current, ok := r.def.RankByID(m.rank)
		if !ok {
			return 0, fmt.Errorf("%w: %q", errUnknownRank, m.rank)
		}

		// The entry class decides which rows the term offers; the current
		// class decides whether a row still applies to who the character
		// now is. A commissioned Temp fails the second test on Rating
		// Promotion, which is moot once he is an officer.
		if !openToClass(advancement, entry) || !eligibleForAdvancement(advancement, current, r.def) {
			continue
		}

		promoted, err := m.attempt(r, advancement, current)
		if err != nil {
			return 0, err
		}

		if promoted {
			gained++
		}
	}

	return gained, nil
}

// openToClass reports whether the row lists the rank's class.
func openToClass(a career.Advancement, rank career.Rank) bool {
	return slices.Contains(a.FromClasses, rank.Class)
}

// eligibleForAdvancement reports whether the rank class may attempt the
// row and whether it has anywhere to go: the top of a ladder bars the
// attempt (interpretation I-13, ERRATA.md).
func eligibleForAdvancement(a career.Advancement, rank career.Rank, def *career.Definition) bool {
	if !openToClass(a, rank) {
		return false
	}

	if a.ToRank != "" {
		return true
	}

	_, hasNext := def.NextRank(rank.ID)

	return hasNext
}

// attempt offers and, if accepted, rolls one advancement.
func (m *merchantMechanics) attempt(r *careerRun, a career.Advancement, rank career.Rank) (bool, error) {
	target, mods, err := m.advancementTarget(r, a)
	if err != nil {
		return false, err
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseAdvancement,
		Prompt:  "Attempt " + a.Name + "?",
		Options: []string{"Attempt " + a.Name, "Decline"},
		Cite:    r.def.Cite + " (" + a.Name + ")",
	})
	if err != nil {
		return false, err
	}

	if chosen != 0 {
		return false, nil
	}

	throw := r.roller.Check(2, target)
	seq := r.log.Throw(throw, mods, r.def.Cite+" ("+a.Name+")")

	if !throw.Success {
		return false, nil
	}

	next := a.ToRank
	if next == "" {
		promoted, ok := r.def.NextRank(rank.ID)
		if !ok {
			return false, nil
		}

		next = promoted.ID
	}

	return true, m.enterRank(r, next, seq)
}

// advancementTarget derives the throw target and its itemized modifiers.
func (m *merchantMechanics) advancementTarget(r *careerRun, a career.Advancement) (int, []Mod, error) {
	var (
		target int
		mods   []Mod
	)

	switch a.Target {
	case career.TargetCharacteristic:
		value, ok := characteristicValue(&r.character.Characteristics, a.Check)
		if !ok {
			return 0, nil, fmt.Errorf("%w: %q", errUnknownCharacteristic, a.Check)
		}

		target = value
	case career.TargetTermsTimesTwo:
		// "Terms x2" counts completed terms, as the p. 66 worked example
		// counts them for Continue (interpretation I-12, ERRATA.md).
		target = len(r.record.Terms) * 2
	default:
		return 0, nil, fmt.Errorf("%w: %q", errUnknownAdvancementTarget, a.Target)
	}

	if a.Mod == nil {
		return target, mods, nil
	}

	value, ok := characteristicValue(&r.character.Characteristics, a.Mod.Characteristic)
	if !ok {
		return 0, nil, fmt.Errorf("%w: %q", errUnknownCharacteristic, a.Mod.Characteristic)
	}

	if value >= a.Mod.Min {
		target += a.Mod.Value
		mods = []Mod{{
			Name:  a.Mod.Characteristic + " " + strconv.Itoa(a.Mod.Min) + "+",
			Value: a.Mod.Value,
		}}
	}

	return target, mods, nil
}
