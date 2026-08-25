package chargen_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/dice"
)

// TestLogSequenceNumbers verifies monotonic sequence numbers from 1
// (docs/PRD.md FR10) across all four event kinds.
func TestLogSequenceNumbers(t *testing.T) {
	roller := dice.New(1)

	var log chargen.Log

	choice := chargen.ChoiceEvent{
		Decider: chargen.DeciderPolicy,
		Prompt:  "p",
		Options: []string{"a", "b"},
		Chosen:  0,
		Cite:    "c",
	}
	consequence := chargen.ConsequenceEvent{
		Cause:          2,
		Kind:           chargen.ConsequenceCharacteristicSet,
		Characteristic: "Str",
		Value:          7,
	}

	seqs := []int{
		log.Step("Characteristics", "Book 1 p. 72 chart E1"),
		log.Roll(roller.Roll(2), "Book 1 p. 56 chart A"),
		log.Throw(roller.Throw(2, 8), []chargen.Mod{{Name: "test", Value: 1}}, "test cite"),
		log.Choice(choice),
		log.Consequence(consequence),
	}

	for i, seq := range seqs {
		if seq != i+1 {
			t.Errorf("event %d assigned seq %d, want %d", i, seq, i+1)
		}
	}

	events := log.Events()
	if len(events) != len(seqs) {
		t.Fatalf("Events() returned %d events, want %d", len(events), len(seqs))
	}

	wantKinds := []chargen.EventKind{
		chargen.EventStep,
		chargen.EventThrow,
		chargen.EventThrow,
		chargen.EventChoice,
		chargen.EventConsequence,
	}

	for i, event := range events {
		if event.Seq != i+1 {
			t.Errorf("stored event %d has seq %d, want %d", i, event.Seq, i+1)
		}

		if event.Kind != wantKinds[i] {
			t.Errorf("stored event %d has kind %q, want %q", i, event.Kind, wantKinds[i])
		}
	}
}

// TestLogPlainRollPayload verifies untargeted rolls carry the FR10 throw
// fields with nil target and success.
func TestLogPlainRollPayload(t *testing.T) {
	roller := dice.New(5)

	var log chargen.Log

	roll := roller.Roll(2)
	log.Roll(roll, "roll cite")

	event := log.Events()[0]

	plain := event.Throw
	if plain == nil || event.Step != nil || event.Choice != nil || event.Consequence != nil {
		t.Fatal("plain roll event: wrong payload set")
	}

	if plain.Expr != "2D" || plain.Total != roll.Total || plain.Cite != "roll cite" {
		t.Errorf("plain roll event = %+v", plain)
	}

	if plain.Target != nil || plain.Success != nil {
		t.Error("plain roll event carries target/success; want nil for untargeted rolls")
	}
}

// TestLogThrowPayload verifies target-number throws carry the
// FR10-required fields: dice expression, individual dice, target,
// modifiers, and rule citation.
func TestLogThrowPayload(t *testing.T) {
	roller := dice.New(5)

	var log chargen.Log

	throw := roller.Throw(3, 9)
	log.Throw(throw, []chargen.Mod{{Name: "homeworld", Value: 2}}, "throw cite")

	targeted := log.Events()[0].Throw
	if targeted == nil {
		t.Fatal("throw event missing payload")
	}

	target := 9
	want := chargen.ThrowEvent{
		Expr:    "3D",
		Dice:    throw.Faces,
		Total:   throw.Total,
		Target:  &target,
		Success: &throw.Success,
		Mods:    []chargen.Mod{{Name: "homeworld", Value: 2}},
		Cite:    "throw cite",
	}

	if !reflect.DeepEqual(*targeted, want) {
		t.Errorf("throw event = %+v, want %+v", *targeted, want)
	}
}

// TestLogIsolation verifies the log shares no memory with its callers:
// mutating inputs after emitting, or the events returned by Events(),
// cannot corrupt the stored record (which replay depends on).
func TestLogIsolation(t *testing.T) {
	roller := dice.New(9)

	var log chargen.Log

	roll := roller.Roll(2)
	log.Roll(roll, "cite")
	log.Throw(roller.Throw(2, 8), []chargen.Mod{{Name: "m", Value: 1}}, "cite")
	log.Choice(chargen.ChoiceEvent{Decider: chargen.DeciderPolicy, Options: []string{"a", "b"}, Chosen: 1})

	// A consequence carrying Mods, which chart F's computed Fame does
	// (chargen/fame.go). Without one the Consequence payload is copied by
	// value and its slice aliasing goes unexercised.
	consequenceMods := []chargen.Mod{{Name: "Rank", Value: 3}}
	log.Consequence(chargen.ConsequenceEvent{
		Cause: 1, Kind: chargen.ConsequenceFameComputed, Value: 3, Mods: consequenceMods,
	})

	before, err := json.Marshal(log.Events())
	if err != nil {
		t.Fatal(err)
	}

	// Mutate the caller-side input slice and everything reachable from a
	// returned snapshot.
	roll.Faces[0] = 99

	snapshot := log.Events()
	snapshot[0].Throw.Dice[0] = 99
	*snapshot[1].Throw.Target = 99
	*snapshot[1].Throw.Success = !*snapshot[1].Throw.Success
	snapshot[1].Throw.Mods[0].Value = 99
	snapshot[2].Choice.Options[0] = "mutated"
	snapshot[2].Consequence = &chargen.ConsequenceEvent{}
	consequenceMods[0].Value = 99
	snapshot[3].Consequence.Mods[0].Value = 99

	after, err := json.Marshal(log.Events())
	if err != nil {
		t.Fatal(err)
	}

	if string(before) != string(after) {
		t.Errorf("log changed after external mutation:\nbefore %s\nafter  %s", before, after)
	}
}

// TestEventJSONShape pins the snake_case wire format of the events array
// (docs/PRD.md FR10, JSON conventions).
func TestEventJSONShape(t *testing.T) {
	var log chargen.Log

	roller := dice.New(1)
	seq := log.Roll(roller.Roll(2), "Book 1 p. 56 chart A")
	log.Consequence(chargen.ConsequenceEvent{
		Cause:          seq,
		Kind:           chargen.ConsequenceCharacteristicSet,
		Characteristic: "Str",
		Value:          7,
	})

	data, err := json.Marshal(log.Events())
	if err != nil {
		t.Fatal(err)
	}

	want := `[{"seq":1,"kind":"throw","throw":{"expr":"2D","dice":[6,1],"total":7,` +
		`"cite":"Book 1 p. 56 chart A"}},{"seq":2,"kind":"consequence",` +
		`"consequence":{"cause":1,"kind":"characteristic_set","characteristic":"Str","value":7}}]`
	if string(data) != want {
		t.Errorf("events JSON =\n%s\nwant\n%s", data, want)
	}
}

// namesAStep checks one consequence's cause and reports whether it named a
// step, failing the test if it named anything else.
func namesAStep(t *testing.T, career string, seed uint64, e chargen.Event, kinds map[int]chargen.EventKind) bool {
	t.Helper()

	switch kinds[e.Consequence.Cause] { //nolint:exhaustive // the rest are faults, named below
	case chargen.EventThrow, chargen.EventChoice:
		return false
	case chargen.EventStep:
		return true
	}

	t.Fatalf("%s seed %d: consequence %q names seq %d, which is %q",
		career, seed, e.Consequence.Kind, e.Consequence.Cause, kinds[e.Consequence.Cause])

	return false
}

// causeSweepOptions asks for one career of the sweep.
//
// Two careers cannot open a lifepath and so cannot be forced by name.
// craftsmanPath reaches chart 01, which the auto policy cannot
// (interpretation I-60) — and chart 01 holds two of I-87's three
// step-caused consequences. functionaryPath reaches chart 13, whose
// "Continue Office Politics" is the one career path that continues a term
// with no Continue throw, and so the one that reads termOutcome.endCause on
// a term nothing went wrong in.
func causeSweepOptions(career string, seed uint64) chargen.Options {
	switch career {
	case "Citizen":
		return chargen.Options{Seed: seed, Decider: &craftsmanPath{}}
	case "Functionary":
		return chargen.Options{Seed: seed, Decider: functionaryPath{first: "Scholar"}}
	default:
		return chargen.Options{Seed: seed, Career: career}
	}
}

// TestEveryConsequenceNamesItsCause walks generated records and pins FR10's
// causality invariant, in the form the amendment leaves it: a consequence
// names a throw, a choice, or — where no throw or choice produced it — the
// step that established the state (interpretation I-87).
//
// The half that is not a formality is the last: a cause of zero names no
// event at all. Nothing asserted it before, and a career whose mechanics
// leave a term's governing throw unset would emit one for every term.
func TestEveryConsequenceNamesItsCause(t *testing.T) {
	t.Parallel()

	careers := []string{
		"Citizen", "Scout", "Rogue", "Merchant", "Soldier", "Spacer",
		"Marine", "Scholar", "Noble", "Entertainer", "Agent", "Functionary",
	}
	steps, total := 0, 0

	for _, career := range careers {
		for seed := uint64(1); seed <= 40; seed++ {
			c, open := generateIfOpen(t, causeSweepOptions(career, seed))
			if !open {
				continue // below chart 11's Soc B+ prerequisite (I-28)
			}

			kinds := make(map[int]chargen.EventKind, len(c.Events))
			for _, e := range c.Events {
				kinds[e.Seq] = e.Kind
			}

			for _, e := range c.Events {
				if e.Consequence == nil {
					continue
				}

				total++

				if namesAStep(t, career, seed, e, kinds) {
					steps++
				}
			}
		}
	}

	if total == 0 {
		t.Fatal("no consequences generated; the test proves nothing")
	}

	// The step causes are real and expected — the Rogue's imprisonment is
	// the common one. If they vanish, I-87 has stopped describing the code.
	if steps == 0 {
		t.Errorf("no consequence named a step across %d consequences; interpretation I-87 describes three sites", total)
	}

	t.Logf("%d consequences, %d of them named a step", total, steps)
}
