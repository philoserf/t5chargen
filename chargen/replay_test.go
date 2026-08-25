package chargen_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// readFixture loads a golden record as the replay verifier's caller would.
func readFixture(t *testing.T, path string) chargen.Character {
	t.Helper()

	data, err := os.ReadFile(path) //nolint:gosec // the path is a testdata fixture chosen by the test
	if err != nil {
		t.Fatal(err)
	}

	var stored chargen.Character
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}

	return stored
}

// TestReplayRoundTrip re-runs every golden record and requires it to
// reproduce itself: "re-running the engine from the recorded seed and
// choices reproduces the identical character" (docs/PRD.md goal 3).
//
// This sweeps the whole engine, not just the verifier. The fixtures cover
// all thirteen careers, including the two no policy can reach, so a change
// anywhere in the lifepath that makes a record unreproducible fails here
// even when the fixture itself still matches — the two are different
// claims. It also proves the record carries enough to rebuild its own
// inputs: eleven of the fourteen were generated under a --career force,
// which holds the first career's option list to one entry.
func TestReplayRoundTrip(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("no fixtures found; the round-trip is asserting nothing")
	}

	for _, file := range files {
		t.Run(strings.TrimSuffix(filepath.Base(file), ".json"), func(t *testing.T) {
			if _, err := chargen.Replay(readFixture(t, file)); err != nil {
				t.Errorf("replay: %v", err)
			}
		})
	}
}

// TestReplayRejectsForeignProvenance verifies replay stops before rolling
// anything when the record was not produced by this build. The point is
// the diagnosis: a foreign record would otherwise diverge at some
// arbitrary sequence number, and that number would describe nothing.
func TestReplayRejectsForeignProvenance(t *testing.T) {
	base := readFixture(t, filepath.Join("testdata", "seed1.json"))

	for _, tc := range []struct {
		name  string
		alter func(*chargen.Character)
	}{
		{"schema_version", func(c *chargen.Character) { c.SchemaVersion = "0.1.0" }},
		{"engine_version", func(c *chargen.Character) { c.EngineVersion = "0.1.0" }},
		{"ruleset", func(c *chargen.Character) { c.Ruleset = "some other book" }},
		{"rng.algorithm", func(c *chargen.Character) { c.RNG.Algorithm = "dice-by-hand" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stored := base
			tc.alter(&stored)

			_, err := chargen.Replay(stored)
			if !errors.Is(err, chargen.ErrReplayProvenance) {
				t.Fatalf("replay of a foreign %s = %v, want ErrReplayProvenance", tc.name, err)
			}

			if !strings.Contains(err.Error(), tc.name) {
				t.Errorf("error %q does not name the field that mismatched", err)
			}

			// Both sides, so a reader can see how far apart the builds
			// are rather than being told one field at a time.
			if !strings.Contains(err.Error(), "this build") {
				t.Errorf("error %q does not say what this build stamps", err)
			}
		})
	}
}

// TestProvenanceNamesEveryMismatch verifies a record several versions old
// is told so at once. Reporting the first difference and stopping says
// less about how far apart the two builds are, and a reader who fixed one
// would only meet the next.
func TestProvenanceNamesEveryMismatch(t *testing.T) {
	stored := readFixture(t, filepath.Join("testdata", "seed1.json"))
	stored.SchemaVersion = "0.1.0"
	stored.EngineVersion = "0.2.0"

	_, err := chargen.Replay(stored)
	if !errors.Is(err, chargen.ErrReplayProvenance) {
		t.Fatalf("replay = %v, want ErrReplayProvenance", err)
	}

	for _, want := range []string{"schema_version", "engine_version"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}

// TestReplayDetectsTampering verifies the verifier verifies. Each case
// alters one thing about the record and requires a divergence naming the
// sequence number of the event that disagreed.
//
// The options case is the one that justifies the whole design: a recorded
// answer is an index, so the same number against a reordered option list
// silently means a different option. POLICY.md's 0.10.0 entry records
// exactly that hazard. Without the options check this record would replay
// "clean" into a different character.
// tamperingCase is one alteration to a record and the name it fails under.
type tamperingCase struct {
	name  string
	alter func(*testing.T, *chargen.Character)
}

// tamperingCases enumerates the alterations replay must catch.
func tamperingCases() []tamperingCase {
	return []tamperingCase{
		{"a reordered option list", func(t *testing.T, c *chargen.Character) {
			t.Helper()

			event := firstRealChoice(t, c.Events)
			event.Options[0], event.Options[1] = event.Options[1], event.Options[0]
		}},
		{"a different answer", func(t *testing.T, c *chargen.Character) {
			t.Helper()

			firstRealChoice(t, c.Events).Chosen = 1
		}},
		{"a reworded prompt", func(t *testing.T, c *chargen.Character) {
			t.Helper()

			firstRealChoice(t, c.Events).Prompt += " (reworded)"
		}},
		{"a tampered throw", func(t *testing.T, c *chargen.Character) {
			t.Helper()

			for _, event := range c.Events {
				if event.Kind == chargen.EventThrow {
					event.Throw.Total++

					return
				}
			}

			t.Fatal("seed 1 recorded no throw")
		}},
		{"a dropped choice", func(t *testing.T, c *chargen.Character) {
			t.Helper()

			for i, event := range c.Events {
				if event.Kind == chargen.EventChoice {
					c.Events = append(c.Events[:i:i], c.Events[i+1:]...)

					return
				}
			}

			t.Fatal("seed 1 recorded no choice")
		}},
		// Out of range is not merely "diverged": without the range
		// check it surfaces as errBadChoice, which blames the decider
		// for answering wrongly when the record and engine disagree.
		{"an answer past the options", func(t *testing.T, c *chargen.Character) {
			t.Helper()

			firstRealChoice(t, c.Events).Chosen = 99
		}},
		{"a changed seed", func(_ *testing.T, c *chargen.Character) { c.RNG.Seed = 2 }},
		{"a changed current year", func(_ *testing.T, c *chargen.Character) { c.Inputs.CurrentYear = 1120 }},
	}
}

func TestReplayDetectsTampering(t *testing.T) {
	for _, tc := range tamperingCases() {
		t.Run(tc.name, func(t *testing.T) {
			// Re-read per case: the alterations reach into the event
			// slice, so cases must not share one record.
			stored := readFixture(t, filepath.Join("testdata", "seed1.json"))
			tc.alter(t, &stored)

			if _, err := chargen.Replay(stored); err == nil {
				t.Fatalf("replay of a record with %s reported no divergence", tc.name)
			} else if !errors.Is(err, chargen.ErrReplayDiverged) {
				t.Fatalf("replay of a record with %s = %v, want ErrReplayDiverged", tc.name, err)
			}
		})
	}
}

// TestReplayComparesDerivedValues isolates the whole-record comparison.
// UPP, credits and the skill list are computed from the run rather than
// carried by any event, so a record whose derived state was altered has an
// event log that agrees completely — "Derived values are stored and
// recomputed on replay" (docs/PRD.md, JSON conventions) is a claim only
// this comparison can check.
func TestReplayComparesDerivedValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		alter func(*chargen.Character)
	}{
		{"upp", "upp", func(c *chargen.Character) { c.UPP = "AAAAAA" }},
		{"credits", "credits", func(c *chargen.Character) { c.Credits += 1000 }},
		{"a skill level", "skills", func(c *chargen.Character) { c.Skills[0].Level++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stored := readFixture(t, filepath.Join("testdata", "seed1.json"))
			before := len(stored.Events)
			tc.alter(&stored)

			if len(stored.Events) != before {
				t.Fatalf("the alteration touched the event log, so it does not isolate the record comparison")
			}

			_, err := chargen.Replay(stored)
			if !errors.Is(err, chargen.ErrReplayDiverged) {
				t.Fatalf("replay of a record with an altered %s = %v, want ErrReplayDiverged", tc.name, err)
			}

			// Naming the field is the difference between a report and two
			// two-kilobyte JSON dumps for the reader to diff by eye.
			if !strings.Contains(err.Error(), tc.field+" differs") {
				t.Errorf("replay reported %q, want it to name %q", err, tc.field)
			}
		})
	}
}

// TestReplayReportsWhereItRanOut verifies the exhaustion message locates
// itself in the record. A truncated log leaves the engine asking for a
// choice the record does not hold, and "past the choices the record holds"
// on its own tells a reader nothing about where — the contract is to report
// a sequence number ("reporting the diverging event's sequence number",
// docs/PRD.md).
func TestReplayReportsWhereItRanOut(t *testing.T) {
	stored := readFixture(t, filepath.Join("testdata", "seed1.json"))

	// Keep the first two choices, so the run gets under way and then runs
	// out with a recorded sequence number behind it.
	kept, seen := []chargen.Event{}, 0

	for _, event := range stored.Events {
		if event.Kind == chargen.EventChoice {
			if seen == 2 {
				break
			}

			seen++
		}

		kept = append(kept, event)
	}

	last := 0

	for _, event := range kept {
		if event.Kind == chargen.EventChoice {
			last = event.Seq
		}
	}

	stored.Events = kept

	_, err := chargen.Replay(stored)
	if !errors.Is(err, chargen.ErrReplayDiverged) {
		t.Fatalf("replay of a truncated record = %v, want ErrReplayDiverged", err)
	}

	if want := fmt.Sprintf("after event %d", last); !strings.Contains(err.Error(), want) {
		t.Errorf("replay reported %q, want it to say %q", err, want)
	}
}

// firstRealChoice returns the record's first choice that had a genuine
// alternative. The record's very first choice is the homeworld, which the
// engine presents with exactly one option as a pure Decider seam — nothing
// about it can be reordered or answered differently.
func firstRealChoice(t *testing.T, events []chargen.Event) *chargen.ChoiceEvent {
	t.Helper()

	for _, event := range events {
		if event.Kind == chargen.EventChoice && len(event.Choice.Options) > 1 {
			return event.Choice
		}
	}

	t.Fatal("record holds no choice with an alternative")

	return nil
}

// TestReplayForcedCareerNeedsTheInput pins why the record carries its
// career force. Blanking the input leaves the engine offering every
// eligible career where the record holds a list of one, so the recorded
// index reads against the wrong list — the failure the inputs block
// exists to prevent.
func TestReplayForcedCareerNeedsTheInput(t *testing.T) {
	stored := readFixture(t, filepath.Join("testdata", "career_agent.json"))

	if stored.Inputs.Career != "Agent" {
		t.Fatalf("fixture's recorded career force = %q, want %q", stored.Inputs.Career, "Agent")
	}

	stored.Inputs.Career = ""

	if _, err := chargen.Replay(stored); !errors.Is(err, chargen.ErrReplayDiverged) {
		t.Fatalf("replay without the recorded force = %v, want ErrReplayDiverged", err)
	}
}

// TestReplayIgnoringProvenanceRunsARecordReplayRefuses is the flag's whole
// point: a record whose versions do not match this build still has a
// generation in it, and the useful answer is where that generation
// disagrees rather than a refusal to look.
//
// The record here is a fixture with its versions falsified, so nothing
// about the run itself changed — which is exactly the case the ordinary
// provenance gate cannot distinguish from a real engine difference, and
// the reason waiving it has to be a deliberate act.
func TestReplayIgnoringProvenanceRunsARecordReplayRefuses(t *testing.T) {
	stored := readFixture(t, filepath.Join("testdata", "seed1.json"))
	stored.SchemaVersion = "0.1.0"
	stored.EngineVersion = "0.2.0"

	if _, err := chargen.Replay(stored); !errors.Is(err, chargen.ErrReplayProvenance) {
		t.Fatalf("Replay = %v, want ErrReplayProvenance (the premise of this test)", err)
	}

	if _, err := chargen.ReplayIgnoringProvenance(stored); err != nil {
		t.Errorf("ReplayIgnoringProvenance = %v, want the run to reproduce", err)
	}
}

// TestReplayIgnoringProvenanceStillVerifies guards the other half: waiving
// the version check must not waive the verification. A record that has
// been tampered with has to be caught whichever entry point re-runs it,
// or the flag would turn replay into a way of not checking.
func TestReplayIgnoringProvenanceStillVerifies(t *testing.T) {
	stored := readFixture(t, filepath.Join("testdata", "seed1.json"))
	stored.EngineVersion = "0.2.0"
	stored.RNG.Seed++

	if _, err := chargen.ReplayIgnoringProvenance(stored); !errors.Is(err, chargen.ErrReplayDiverged) {
		t.Errorf("ReplayIgnoringProvenance = %v, want ErrReplayDiverged", err)
	}
}

// TestReplayIgnoringProvenanceKeepsTheProvenanceOutOfTheDiff checks that
// the falsified versions are not then reported back as a record
// difference. A caller who waived the check and got it returned as the
// answer would have gained nothing, and the real divergence — if there
// were one — would be behind it.
func TestReplayIgnoringProvenanceKeepsTheProvenanceOutOfTheDiff(t *testing.T) {
	stored := readFixture(t, filepath.Join("testdata", "seed1.json"))
	stored.SchemaVersion = "0.1.0"
	stored.EngineVersion = "0.2.0"
	stored.Ruleset = "some other book"
	stored.RNG.Algorithm = "not-pcg"

	if _, err := chargen.ReplayIgnoringProvenance(stored); err != nil {
		t.Errorf("ReplayIgnoringProvenance = %v, want the four provenance fields excluded from the diff", err)
	}
}
