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
