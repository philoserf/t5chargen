package chargen_test

import (
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/dice"
)

// TestRollCharacteristicsStreamOrder verifies the chart A roll order (p. 56:
// "record the results in the order they are rolled: Strength, Dexterity,
// Endurance, Intelligence, Education, and Social Standing") is stream order:
// six 2D rolls, nothing more, in Str..Soc order.
func TestRollCharacteristicsStreamOrder(t *testing.T) {
	const seed = 11

	var log chargen.Log

	got := chargen.RollCharacteristics(dice.New(seed), &log)

	manual := dice.New(seed)
	want := chargen.Characteristics{
		Str: manual.Roll(2).Total,
		Dex: manual.Roll(2).Total,
		End: manual.Roll(2).Total,
		Int: manual.Roll(2).Total,
		Edu: manual.Roll(2).Total,
		Soc: manual.Roll(2).Total,
	}

	if got != want {
		t.Errorf("RollCharacteristics = %+v, want %+v", got, want)
	}
}

// TestRollCharacteristicsEvents verifies each characteristic emits a throw
// event followed by a characteristic_set consequence referencing it
// (docs/PRD.md FR10).
func TestRollCharacteristicsEvents(t *testing.T) {
	var log chargen.Log

	c := chargen.RollCharacteristics(dice.New(3), &log)

	events := log.Events()
	if len(events) != 12 {
		t.Fatalf("got %d events, want 12 (throw + consequence per characteristic)", len(events))
	}

	names := []string{"Str", "Dex", "End", "Int", "Edu", "Soc"}
	values := []int{c.Str, c.Dex, c.End, c.Int, c.Edu, c.Soc}

	for i, name := range names {
		checkCharacteristicPair(t, name, values[i], events[2*i], events[2*i+1])
	}
}

// checkCharacteristicPair asserts one characteristic's throw event and the
// characteristic_set consequence referencing it.
func checkCharacteristicPair(t *testing.T, name string, value int, throw, consequence chargen.Event) {
	t.Helper()

	if throw.Kind != chargen.EventThrow || throw.Throw == nil {
		t.Fatalf("event %d: want throw, got %+v", throw.Seq, throw)
	}

	if throw.Throw.Expr != "2D" || throw.Throw.Total != value {
		t.Errorf("%s throw = %+v, want 2D totaling %d", name, throw.Throw, value)
	}

	if consequence.Kind != chargen.EventConsequence || consequence.Consequence == nil {
		t.Fatalf("event %d: want consequence, got %+v", consequence.Seq, consequence)
	}

	want := chargen.ConsequenceEvent{
		Cause:          throw.Seq,
		Kind:           chargen.ConsequenceCharacteristicSet,
		Characteristic: name,
		Value:          value,
	}
	if *consequence.Consequence != want {
		t.Errorf("%s consequence = %+v, want %+v", name, *consequence.Consequence, want)
	}
}

// TestGoldenUPP pins the derived UPP for seed 1 against the golden face
// stream (6,1 6,6 2,6 5,1 4,2 3,6): part of the replay contract.
func TestGoldenUPP(t *testing.T) {
	var log chargen.Log

	c := chargen.RollCharacteristics(dice.New(1), &log)

	if got, want := c.UPP(), "7C8669"; got != want {
		t.Errorf("seed 1 UPP = %q, want %q", got, want)
	}
}

// TestUPP checks eHex digit rendering, including the p. 22 "?" convention
// for out-of-range values.
func TestUPP(t *testing.T) {
	tests := []struct {
		name string
		c    chargen.Characteristics
		want string
	}{
		{"typical", chargen.Characteristics{Str: 7, Dex: 7, End: 7, Int: 7, Edu: 7, Soc: 7}, "777777"},
		{"hex digits", chargen.Characteristics{Str: 10, Dex: 11, End: 12, Int: 13, Edu: 14, Soc: 15}, "ABCDEF"},
		{"mixed", chargen.Characteristics{Str: 2, Dex: 12, End: 8, Int: 6, Edu: 6, Soc: 9}, "2C8669"},
		{"out of range renders ?", chargen.Characteristics{Str: 7, Dex: 7, End: 34, Int: -1, Edu: 6, Soc: 7}, "77??67"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.UPP(); got != tt.want {
				t.Errorf("UPP() = %q, want %q", got, tt.want)
			}
		})
	}
}
