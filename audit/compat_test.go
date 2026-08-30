package audit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/render"
)

// The compatibility corpus: one record per released version, written by
// that version's own binary and never regenerated.
//
// docs/COMPATIBILITY.md makes two promises, and this is the gate under
// them. Prose saying older records still render is worth nothing on its
// own — the fixtures here were produced by `go install ...@v0.1.0-alpha.1`
// and are the actual output of a version that no longer exists in the
// tree.
//
// `task goldens` must never touch these. It rewrites ./chargen and
// ./render, deliberately not this directory: a corpus a later engine can
// rewrite proves nothing about what an earlier engine wrote.
const corpusDir = "testdata/corpus"

// TestTheCorpusHoldsEveryReleasedVersion guards the corpus against
// quietly emptying. A compatibility gate with nothing in it passes.
func TestTheCorpusHoldsEveryReleasedVersion(t *testing.T) {
	records := corpusRecords(t)

	if len(records) == 0 {
		t.Fatal("the compatibility corpus is empty; it would pass while proving nothing")
	}

	versions := map[string]bool{}

	for _, path := range records {
		// v0.1.0-alpha.1_auto.json -> v0.1.0-alpha.1
		name := filepath.Base(path)
		versions[name[:strings.LastIndex(name, "_")]] = true
	}

	// Every released version must appear. The list is written out rather
	// than derived from git tags: a tag can be deleted, and the promise
	// this gate holds is about what was published, not about what the
	// repository still remembers publishing.
	for _, released := range []string{"v0.1.0-alpha.1"} {
		if !versions[released] {
			t.Errorf("no corpus record from %s; docs/COMPATIBILITY.md promises it still renders", released)
		}
	}
}

// TestEveryCorpusRecordStillRenders is the render-forward promise:
// docs/COMPATIBILITY.md says a record written by a released version
// renders under every later released version. This engine is the later
// one.
func TestEveryCorpusRecordStillRenders(t *testing.T) {
	for _, path := range corpusRecords(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			c := corpusRecord(t, path)

			sheet := render.Sheet(c)
			if !strings.Contains(sheet, "UPP") {
				t.Errorf("the sheet carries no UPP line:\n%s", sheet)
			}

			if history := render.History(c); history == "" {
				t.Error("the history transcript is empty")
			}

			// A record whose events did not survive the round trip would
			// still render, emptily. The transcript is the record's own
			// account of itself, so it has to have something to say.
			if len(c.Events) == 0 {
				t.Error("the record decoded with no events")
			}
		})
	}
}

// TestCorpusReplayIsPinnedToItsEngine is the other half of
// docs/COMPATIBILITY.md: replay stays pinned to the engine that wrote the
// record, and refuses rather than guessing.
//
// Both outcomes are correct, and which one is right depends on whether
// the engine has moved since the record was written — so the gate is that
// the outcome matches the versions, not that it is one particular
// outcome. Today engine 0.45.0 wrote the corpus and still reads it, so
// these replay clean; the first engine bump turns that into a refusal,
// and the refusal has to name the version it wants.
func TestCorpusReplayIsPinnedToItsEngine(t *testing.T) {
	for _, path := range corpusRecords(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			c := corpusRecord(t, path)

			switch _, err := chargen.Replay(c); {
			case err == nil:
				// A clean replay is only legitimate while this engine is
				// the one that wrote the record. Replaying a record from
				// a different engine would mean the pin is not holding.
				if c.EngineVersion != chargen.EngineVersion {
					t.Errorf("engine %s replayed a record written by %s; the provenance pin is not holding",
						chargen.EngineVersion, c.EngineVersion)
				}
			case !strings.Contains(err.Error(), c.EngineVersion):
				t.Errorf("replay refused without naming the engine that wrote the record (%s): %v",
					c.EngineVersion, err)
			}
		})
	}
}

// corpusRecords lists the corpus fixtures.
func corpusRecords(t *testing.T) []string {
	t.Helper()

	records, err := filepath.Glob(filepath.Join(corpusDir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	return records
}

// corpusRecord decodes one corpus fixture.
func corpusRecord(t *testing.T, path string) chargen.Character {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // G304: fixed test-owned paths under testdata/.
	if err != nil {
		t.Fatal(err)
	}

	var c chargen.Character
	if err := json.Unmarshal(body, &c); err != nil {
		t.Fatalf("%s no longer decodes into the current Character: %v", path, err)
	}

	return c
}
