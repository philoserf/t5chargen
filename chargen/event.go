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

// ConsequenceCharacteristicSet records a characteristic receiving its
// initial rolled value (chart A, p. 56).
const ConsequenceCharacteristicSet ConsequenceKind = "characteristic_set"

// ConsequenceEvent records one effect of an earlier throw or choice. Cause
// is the sequence number of that event. Payload fields are structured, not
// prose, so replay can compare stored and recomputed events exactly; the
// transcript renderer derives readable lines from them.
type ConsequenceEvent struct {
	Cause          int             `json:"cause"`
	Kind           ConsequenceKind `json:"kind"`
	Characteristic string          `json:"characteristic,omitempty"` // Str, Dex, End, Int, Edu, Soc
	Value          int             `json:"value"`
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
