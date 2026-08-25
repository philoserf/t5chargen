package chargen_test

import (
	"encoding/json"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

type hostile struct{ chargen.DefaultPolicy }

func (hostile) Watch(e chargen.Event) {
	// The step payload is the one a real watcher reads (interactive's
	// session takes its headings from Step.Name), so it is the one the
	// guard most needs to cover.
	if e.Step != nil {
		e.Step.Name = "Tampered"
		e.Step.Cite = "Tampered"
	}

	if e.Consequence != nil {
		e.Consequence.Value = 9999
		e.Consequence.Skill = "Tampered"
		e.Consequence.Mods = nil
	}

	if e.Throw != nil {
		e.Throw.Total = 9999
		e.Throw.Dice = nil
	}

	if e.Choice != nil {
		e.Choice.Options = nil
	}
}

func TestHostileWatcherCannotCorruptTheRecord(t *testing.T) {
	clean := generate(t, chargen.Options{Seed: 1})
	watched := generate(t, chargen.Options{Seed: 1, Decider: hostile{}})

	if len(clean.Events) != len(watched.Events) {
		t.Fatalf("%d events watched, %d unwatched", len(watched.Events), len(clean.Events))
	}

	for i := range clean.Events {
		if jsonOf(t, clean.Events[i]) != jsonOf(t, watched.Events[i]) {
			t.Fatalf("event %d differs when watched", clean.Events[i].Seq)
		}
	}
}

func jsonOf(t *testing.T, e chargen.Event) string {
	t.Helper()

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}
