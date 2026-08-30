package chargen_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// FuzzReplayMalformedRecord holds Replay to the promise that matters most
// for a beta tester's saved characters: a record it will not accept comes
// back as an error, never as a panic and never as a character.
//
// Replay is the provenance contract's enforcement point (docs/PRD.md
// FR10), so it is the function most likely to be handed a record from an
// engine that no longer exists, or one a user edited by hand to see what
// happened. Both are refusals, and a refusal is a return value.
//
// The seed corpus is small records, not the goldens: at ~220KB the
// fixtures reduce the fuzzing engine to a few dozen executions in twenty
// seconds, because mutating an input that size costs far more than the
// replay it is testing.
func FuzzReplayMalformedRecord(f *testing.F) {
	minimal, err := os.ReadFile(filepath.Join("..", "docs", "character.minimal.json"))
	if err != nil {
		f.Fatal(err)
	}

	f.Add(minimal)

	for _, seed := range []string{
		`{}`,
		`{"rng":{"algorithm":"pcg","seed":1}}`,
		`{"rng":{"algorithm":"unknown","seed":1}}`,
		`{"ruleset":{"engine_version":"0.0.0"}}`,
		`{"rng":{"algorithm":"pcg","seed":1},"inputs":{"career":"Scout"}}`,
		`{"rng":{"algorithm":"pcg","seed":18446744073709551615}}`,
		`{"inputs":{"career":"\ud800","current_year":0}}`,
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		var stored chargen.Character

		if json.Unmarshal(body, &stored) != nil {
			return
		}

		// A refusal and a divergence are both correct answers, so the
		// error is not the assertion. What must hold is that a replay
		// which succeeds regenerated the record it was given: it reads
		// the seed out of the record, so the character it returns cannot
		// carry a different one.
		for _, replay := range []func(chargen.Character) (chargen.Character, error){
			chargen.Replay, chargen.ReplayIgnoringProvenance,
		} {
			got, err := replay(stored)
			if err != nil {
				continue
			}

			if got.RNG.Seed != stored.RNG.Seed {
				t.Fatalf("replay of seed %d returned a character with seed %d",
					stored.RNG.Seed, got.RNG.Seed)
			}
		}
	})
}
