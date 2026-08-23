// Package chargen is the character-generation engine. This file defines the
// generation record: the ordered event log of every checklist step, throw,
// choice, and consequence in a lifepath (docs/PRD.md FR10).
//
// Events carry monotonic sequence numbers starting at 1 and increasing by 1.
// Consequence events reference the sequence number of the throw or choice
// that caused them. The stored log is verification data for replay: the
// engine re-runs from the seed and choice events and recomputes every throw
// (docs/PRD.md, Replay and provenance contract).
package chargen

import (
	"slices"

	"github.com/philoserf/t5chargen/dice"
)

// EventKind discriminates the four generation-record event kinds
// (docs/PRD.md FR10).
type EventKind string

// The four event kinds: a checklist step entered, a throw, a choice, and a
// consequence.
const (
	EventStep        EventKind = "step"
	EventThrow       EventKind = "throw"
	EventChoice      EventKind = "choice"
	EventConsequence EventKind = "consequence"
)

// Event is one entry in the generation record. Exactly one payload field is
// non-nil, matching Kind.
type Event struct {
	Seq  int       `json:"seq"`
	Kind EventKind `json:"kind"`

	Step        *StepEvent        `json:"step,omitempty"`
	Throw       *ThrowEvent       `json:"throw,omitempty"`
	Choice      *ChoiceEvent      `json:"choice,omitempty"`
	Consequence *ConsequenceEvent `json:"consequence,omitempty"`
}

// StepEvent records entering a step of the Master Chargen Checklist
// (Book 1 chart E1, p. 72).
type StepEvent struct {
	Name string `json:"name"`
	Cite string `json:"cite"`
}

// Mod is one named modifier applied to a throw's target number ("A Mod
// Changes The Target Number", Book 1 p. 18).
type Mod struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// ThrowEvent records one throw: "throw events record the dice expression,
// individual dice, target, modifiers, and rule citation" (docs/PRD.md FR10).
// Target and Success are nil for plain rolls that resolve against no target
// (for example the 2D characteristic rolls of chart A, p. 56).
type ThrowEvent struct {
	Expr    string `json:"expr"`
	Dice    []int  `json:"dice"`
	Mod     int    `json:"mod,omitempty"` // fixed adjustment in the expression (the -7 in 2D-7)
	Total   int    `json:"total"`
	Target  *int   `json:"target,omitempty"`
	Success *bool  `json:"success,omitempty"`
	Mods    []Mod  `json:"mods,omitempty"` // named target-number modifiers, itemized
	Cite    string `json:"cite"`
}

// DeciderKind identifies who resolved a choice (docs/PRD.md FR10: "who
// decided — player or policy").
type DeciderKind string

// The two deciders of docs/PRD.md goal 2: the interactive player and the
// auto-mode policy.
const (
	DeciderPlayer DeciderKind = "player"
	DeciderPolicy DeciderKind = "policy"
)

// ChoiceEvent records one resolved choice point: who decided, what the
// alternatives were, and which was chosen (docs/PRD.md FR10). Choice events
// are replay input: re-running the engine reapplies them.
type ChoiceEvent struct {
	Decider DeciderKind `json:"decider"`
	Prompt  string      `json:"prompt"`
	Options []string    `json:"options"`
	Chosen  int         `json:"chosen"` // index into Options
	Cite    string      `json:"cite"`
}

// ConsequenceKind discriminates consequence payloads. Kinds are added as
// mechanics grow; docs/PRD.md FR10 names skill awards, rank gains,
// characteristic changes, and elapsed years.
type ConsequenceKind string

// The consequence kinds the engine currently emits.
const (
	// ConsequenceCharacteristicSet records a characteristic receiving
	// its initial rolled value (chart A, p. 56). Value is the new value.
	ConsequenceCharacteristicSet ConsequenceKind = "characteristic_set"

	// ConsequenceCharacteristicChange records a characteristic changing
	// (for example table C column 1 Personal, p. 78). Delta is the
	// change, Value the new value.
	ConsequenceCharacteristicChange ConsequenceKind = "characteristic_change"

	// ConsequenceSkillAwarded records a skill receipt. Delta is the
	// levels received, Value the resulting level.
	ConsequenceSkillAwarded ConsequenceKind = "skill_awarded"

	// ConsequenceJobSet and ConsequenceHobbySet record the Citizen Job
	// and Hobby determinations ("Once determined, Job and Hobby cannot
	// be changed", chart 04 p. 78). Skill names the determined skill.
	ConsequenceJobSet   ConsequenceKind = "job_set"
	ConsequenceHobbySet ConsequenceKind = "hobby_set"

	// ConsequenceNoAward records a receipt that awards nothing: a failed
	// Citizen Life throw ("no Job or Hobby skills", p. 78), a table cell
	// with no award, or a skill receipt fully absorbed by the Skill-15
	// cap (p. 134) — the last carries Skill naming the capped skill.
	ConsequenceNoAward ConsequenceKind = "no_award"

	// ConsequenceJobUndetermined records a Job determination roll landing
	// on the "No Skill" cell of table E (p. 78): the Job stays
	// undetermined and the next Citizen Life success retries
	// (interpretation I-1, ERRATA.md).
	ConsequenceJobUndetermined ConsequenceKind = "job_undetermined"

	// ConsequenceBenefitLost records a benefit that could not apply: a
	// Major/Minor cell without the education to use it ("If the character
	// does not have a Major/Minor this benefit is lost", p. 78 table C
	// note), or — with Characteristic set — a characteristic increase
	// blocked by the human maximum ("Characteristics for Humans cannot
	// exceed 15. If a benefit elevates a characteristic above 15, that
	// benefit is lost.", p. 68).
	ConsequenceBenefitLost ConsequenceKind = "benefit_lost"

	// ConsequenceMandatoryContinue records a Continue roll of exactly 2:
	// "If the Continue roll is 2 exactly, the character is required to
	// Continue in the current Career" (p. 66).
	ConsequenceMandatoryContinue ConsequenceKind = "mandatory_continue"

	// ConsequenceYearsElapsed records career time passing. Value is the
	// years ("the 4-year Term", p. 66).
	ConsequenceYearsElapsed ConsequenceKind = "years_elapsed"

	// ConsequenceCareerEnded records the end of career resolution
	// ("Failure ends Career Resolution", p. 66). Career names it.
	ConsequenceCareerEnded ConsequenceKind = "career_ended"

	// ConsequenceCareerNotBegun records a failed To Begin: "If both Begin
	// and Retry fail, this career may not be used. Each failed attempt
	// ... takes one year." (p. 65) Career names it.
	ConsequenceCareerNotBegun ConsequenceKind = "career_not_begun"

	// ConsequenceWoundBadge records a Risk-failure wound: "If the
	// Controlling Characteristic is reduced, the Character has been
	// wounded and receives a Wound Badge." (p. 65) Value is the badge
	// count.
	ConsequenceWoundBadge ConsequenceKind = "wound_badge"

	// ConsequenceDisabled records a 4+ reduction: "If CC is reduced by 4
	// or more, then he is disabled. Muster Out at Term end with Double
	// Benefits" (chart 05 p. 79; p. 65).
	ConsequenceDisabled ConsequenceKind = "disabled"

	// ConsequenceDead records a death. Injury names the characteristic:
	// "If the Controlling Characteristic is reduced to zero or less, the
	// Character is dead" (p. 65). Aging names none: "The second time
	// three characteristics are reduced to 0, the character dies"
	// (p. 89 chart A).
	ConsequenceDead ConsequenceKind = "dead"

	// ConsequenceDiscovery records a Scout Reward success: "The Scout
	// discovers a valuable new world or a valuable feature on a known
	// world (a Discovery)" (chart 05 p. 79). Value is the running count.
	ConsequenceDiscovery ConsequenceKind = "discovery"

	// ConsequenceEntitlement records an annual payment (chart M1 p. 70).
	// Delta is what it pays a year and Value the age it begins.
	ConsequenceEntitlement ConsequenceKind = "entitlement"

	// ConsequenceBenefit records a muster-out award (chart M1 p. 70).
	// Skill is the benefit's name, Value how many, Delta the credits it
	// paid, and Characteristic the one a "Str +1" raised.
	ConsequenceBenefit ConsequenceKind = "benefit"

	// ConsequenceMasterpiece records a Craftsman creation success (chart
	// 01 p. 75). Value is the Master Points it scored, Delta the credits
	// it would sell for; a Perfect Masterpiece is one of 55 or more.
	ConsequenceMasterpiece ConsequenceKind = "masterpiece"

	// ConsequenceAssociated records the prior career a Functionary
	// position belongs to (chart 13 p. 87); Skill carries its name.
	ConsequenceAssociated ConsequenceKind = "associated"

	// ConsequenceReserve records enrolment on leaving the Armed Forces
	// (p. 67). Skill carries the Reserve Rank, which is the last rank
	// held: "There is no process for promotion or advancement while in
	// the Reserves".
	ConsequenceReserve ConsequenceKind = "reserve"

	// ConsequenceCareerChanged records a voluntary career change: "A
	// character may avoid the Continue roll ... by voluntarily ending his
	// service in the current career and selecting a different career for
	// which he is eligible" (p. 66); Career is the career left.
	ConsequenceCareerChanged ConsequenceKind = "career_changed"

	// ConsequenceAgingEffect records an Aging Check that landed: "If the
	// Aging Check imposes an effect, the characteristic is reduced -1"
	// (p. 89 chart A). Delta is -1, Value the resulting value.
	ConsequenceAgingEffect ConsequenceKind = "aging_effect"

	// ConsequenceCharacteristicReset records the floor aging cannot pass:
	// "If one Characteristic is reduced to 0, it is reset to 1" (p. 89
	// chart A).
	ConsequenceCharacteristicReset ConsequenceKind = "characteristic_reset"

	// ConsequenceMajorIllness records two characteristics reduced to zero
	// in one Aging Check: "the character suffers a major illness and must
	// spend four weeks in rest and recuperation" (p. 89 chart A). Value
	// is how many reached zero.
	ConsequenceMajorIllness ConsequenceKind = "major_illness"

	// ConsequenceExtremelyMajorIllness records three or more: "an
	// extremely major illness and must spend four months in rest and
	// recuperation" (p. 89 chart A; interpretation I-49 reads the
	// chart's three as three or more). Value is how many reached zero.
	ConsequenceExtremelyMajorIllness ConsequenceKind = "extremely_major_illness"

	// ConsequenceSanityMod records a pending reduction against a Sanity
	// value that has not been generated: "reduce San= -1 for each TWO
	// Terms served" (chart 05 p. 79; p. 52). Delta is the reduction the
	// career owes; there is no value to total it against.
	ConsequenceSanityMod ConsequenceKind = "sanity_mod"

	// ConsequenceFameComputed records the Fame calculated over a finished
	// character (chart F p. 91). Value is the level, Skill its printed
	// descriptor, and Mods the priced accomplishments it stacks — the
	// arithmetic the chart's own example shows ("Fame = 1 x 3 = 3").
	ConsequenceFameComputed ConsequenceKind = "fame_computed"

	// ConsequenceFameChange records a Fame change (chart 05: "Fame +1";
	// the full Fame system, chart F p. 91, lands with milestone 4).
	// Delta is the change, Value the new Fame.
	ConsequenceFameChange ConsequenceKind = "fame_change"

	// ConsequenceRankSet records entry to a rank, on beginning a career
	// or on advancement (chart 06 Table Of Merchant Ranks, p. 80). Skill
	// carries the rank title.
	ConsequenceRankSet ConsequenceKind = "rank_set"

	// ConsequenceScheme records the Rogue's plan for the term,
	// ConsequenceSentenced a prison sentence owed, ConsequenceImprisoned
	// a term spent serving one, and ConsequencePayoff a scheme's takings
	// (chart 10, p. 84).
	ConsequenceScheme     ConsequenceKind = "scheme"
	ConsequenceSentenced  ConsequenceKind = "sentenced"
	ConsequenceImprisoned ConsequenceKind = "imprisoned"
	ConsequencePayoff     ConsequenceKind = "payoff"

	// ConsequenceUndercover records an Agent's cover identity for the
	// term, and ConsequenceCommendation a Reward success (chart 09,
	// p. 83).
	ConsequenceUndercover   ConsequenceKind = "undercover_assignment"
	ConsequenceCommendation ConsequenceKind = "commendation"

	// ConsequenceBranchSet records the Armed Forces branch served in, and
	// ConsequenceOperation one of the term's assignments (chart 08,
	// p. 82).
	ConsequenceBranchSet ConsequenceKind = "branch_set"
	ConsequenceOperation ConsequenceKind = "operation"

	// ConsequenceMedal records a decoration from the p. 70 table, and
	// ConsequenceServiceBadge the Risk-success Exemplary Service Badge.
	ConsequenceMedal        ConsequenceKind = "medal"
	ConsequenceServiceBadge ConsequenceKind = "service_badge"

	// ConsequenceExiled and ConsequenceReturned record the Noble's exile
	// state (chart 11, p. 85).
	ConsequenceExiled   ConsequenceKind = "exiled"
	ConsequenceReturned ConsequenceKind = "returned_from_exile"

	// ConsequenceIntrigue records a successful Intrigue, which earns an
	// Elevation attempt (chart 11).
	ConsequenceIntrigue ConsequenceKind = "intrigue"

	// ConsequenceElevated records an Elevation to the next Noble rank
	// (chart 11); ConsequenceLandGrant records the Land Grant a Soc
	// increase or a Scout Discovery awards; ConsequenceBirthdate records
	// the day the character was born (p. 58; p. 263), which is the last
	// thing generation settles.
	ConsequenceElevated  ConsequenceKind = "elevated"
	ConsequenceLandGrant ConsequenceKind = "land_grant"
	ConsequenceBirthdate ConsequenceKind = "birthdate"

	// ConsequencePublication records a Scholar Publication success
	// (chart 02, p. 76); an Award-Winning publication has Delta 2.
	ConsequencePublication ConsequenceKind = "publication"

	// ConsequenceTenure records the chart 02 Tenure grant.
	ConsequenceTenure ConsequenceKind = "tenure"

	// ConsequenceWaived records an adverse outcome negated by a waiver
	// (chart 02, p. 76; p. 59).
	ConsequenceWaived ConsequenceKind = "waived"

	// ConsequenceMajorSet and ConsequenceMinorSet record the Scholar's
	// areas where the career selects them (chart 02).
	ConsequenceMajorSet ConsequenceKind = "major_set"
	ConsequenceMinorSet ConsequenceKind = "minor_set"

	// ConsequenceTalentSet records the Entertainer's Talent, set with
	// Fame before Begin and raised whenever Fame increases (chart 03,
	// p. 77).
	ConsequenceTalentSet ConsequenceKind = "talent_set"

	// ConsequenceSpecialtySet records the Entertainer's chosen art
	// (chart 03 "Select A Specialty").
	ConsequenceSpecialtySet ConsequenceKind = "specialty_set"

	// ConsequenceComeback records an Entertainer Comeback: "Reset Fame to
	// 2D; Talent is unchanged" (chart 03).
	ConsequenceComeback ConsequenceKind = "comeback"

	// ConsequenceShipShares records a Merchant Reward award: "Every
	// Reward gives the character Ship Shares" on the escalating scale
	// (chart 06, p. 80).
	ConsequenceShipShares ConsequenceKind = "ship_shares"
)

// ConsequenceEvent records one effect of an earlier throw or choice. Cause
// is the sequence number of that event. Payload fields are structured, not
// prose, so replay can compare stored and recomputed events exactly; the
// transcript renderer derives readable lines from them.
type ConsequenceEvent struct {
	Cause          int             `json:"cause"`
	Kind           ConsequenceKind `json:"kind"`
	Characteristic string          `json:"characteristic,omitempty"` // Str, Dex, End, Int, Edu, Soc
	Skill          string          `json:"skill,omitempty"`
	Career         string          `json:"career,omitempty"`

	// Detail carries a consequence whose value is neither a skill nor a
	// number, as BenefitRecord.Detail does for an award.
	Detail string `json:"detail,omitempty"`

	Delta int `json:"delta,omitempty"`
	Value int `json:"value,omitempty"`

	// Mods itemizes a computed consequence's sources, as a throw's Mods
	// itemize its target: chart F's Fame is a sum the record should show
	// the working for (docs/PRD.md FR10).
	Mods []Mod `json:"mods,omitempty"`
}

// Log accumulates the generation record in event order.
//
// The zero value is ready to use. A Log is not safe for concurrent use; the
// engine emits events sequentially so that replay is deterministic.
type Log struct {
	events []Event
}

// Events returns a deep copy of the log in sequence order: mutating the
// returned events, their payloads, or their slices cannot corrupt the
// stored record, which replay depends on.
func (l *Log) Events() []Event {
	events := make([]Event, len(l.events))
	for i, event := range l.events {
		events[i] = event.clone()
	}

	return events
}

// clone deep-copies the event's payload pointer and any inner slices.
func (e Event) clone() Event {
	if e.Step != nil {
		step := *e.Step
		e.Step = &step
	}

	if e.Throw != nil {
		throw := *e.Throw
		throw.Dice = slices.Clone(throw.Dice)
		throw.Mods = slices.Clone(throw.Mods)

		if throw.Target != nil {
			target := *throw.Target
			throw.Target = &target
		}

		if throw.Success != nil {
			success := *throw.Success
			throw.Success = &success
		}

		e.Throw = &throw
	}

	if e.Choice != nil {
		choice := *e.Choice
		choice.Options = slices.Clone(choice.Options)
		e.Choice = &choice
	}

	if e.Consequence != nil {
		consequence := *e.Consequence
		e.Consequence = &consequence
	}

	return e
}

// Step records entering a checklist step and returns its sequence number.
func (l *Log) Step(name, cite string) int {
	return l.append(Event{Kind: EventStep, Step: &StepEvent{Name: name, Cite: cite}})
}

// Roll records a plain roll — one that resolves against no target number —
// and returns its sequence number. The faces are copied: the caller's
// dice.Roll cannot alias the stored record.
func (l *Log) Roll(roll dice.Roll, cite string) int {
	return l.append(Event{Kind: EventThrow, Throw: &ThrowEvent{
		Expr:  roll.Expr(),
		Dice:  slices.Clone(roll.Faces),
		Mod:   roll.Mod,
		Total: roll.Total,
		Cite:  cite,
	}})
}

// Flux records a Flux roll ("Flux is Light Die minus Dark Die", p. 261)
// as a throw event with both faces, and returns its sequence number.
func (l *Log) Flux(flux dice.FluxRoll, cite string) int {
	return l.append(Event{Kind: EventThrow, Throw: &ThrowEvent{
		Expr:  flux.Expr(),
		Dice:  []int{flux.First, flux.Second},
		Total: flux.Value,
		Cite:  cite,
	}})
}

// Throw records a target-number throw with its itemized target modifiers
// and returns its sequence number. The faces and mods are copied: the
// caller's slices cannot alias the stored record.
func (l *Log) Throw(throw dice.Throw, mods []Mod, cite string) int {
	return l.append(Event{Kind: EventThrow, Throw: &ThrowEvent{
		Expr:    throw.Expr(),
		Dice:    slices.Clone(throw.Faces),
		Mod:     throw.Mod,
		Total:   throw.Total,
		Target:  &throw.Target,
		Success: &throw.Success,
		Mods:    slices.Clone(mods),
		Cite:    cite,
	}})
}

// Choice records a resolved choice point and returns its sequence number.
// The options are copied: the caller's slice cannot alias the stored record.
func (l *Log) Choice(choice ChoiceEvent) int {
	choice.Options = slices.Clone(choice.Options)

	return l.append(Event{Kind: EventChoice, Choice: &choice})
}

// Consequence records an effect of the throw or choice at sequence number
// cause, and returns its own sequence number.
func (l *Log) Consequence(consequence ConsequenceEvent) int {
	return l.append(Event{Kind: EventConsequence, Consequence: &consequence})
}

// append assigns the next sequence number (monotonic from 1) and stores the
// event.
func (l *Log) append(event Event) int {
	event.Seq = len(l.events) + 1
	l.events = append(l.events, event)

	return event.Seq
}
