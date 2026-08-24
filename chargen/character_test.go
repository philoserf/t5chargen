package chargen_test

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// update rewrites the golden fixtures instead of comparing against them:
// `task goldens`, or `go test ./chargen -update`. The fixtures are the
// engine's own output, compared byte for byte, so regenerating them is a
// deliberate act that belongs in the same commit as the change that moved
// them.
var update = flag.Bool("update", false, "rewrite testdata golden files instead of comparing")

// goldenJSON compares the record's canonical serialization against the
// named fixture, or rewrites the fixture under -update. Marshalling lives
// here and nowhere else so the writer and the comparison cannot disagree
// about the bytes a fixture holds.
func goldenJSON(t *testing.T, c chargen.Character, file string) {
	t.Helper()

	got, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	got = append(got, '\n')

	if *update {
		requireVersionBump(t, file, got)

		if err := os.WriteFile(file, got, 0o600); err != nil {
			t.Fatal(err)
		}

		return
	}

	want, err := os.ReadFile(file) //nolint:gosec // G304: fixed test-owned paths under testdata/.
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != string(want) {
		t.Errorf("record differs from %s:\n%s", file, got)
	}
}

// requireVersionBump enforces the rule the replay contract depends on: a
// regeneration that changes anything replay recomputes must carry a new
// engine_version.
//
// It exists because the rule was stated and then missed twice in one
// milestone. Replay excludes policy_version from its provenance check by
// design — recorded choices are reapplied, so the policy is never consulted
// — which leaves engine_version as the only gate. Miss the bump and an old
// record passes checkProvenance and then dies somewhere in compareEvents:
// one such record reported `"change_career": recorded the answer 3, outside
// the 2 options` at event 60, blaming a career choice for a build mismatch.
//
// Changing which option the policy picks is a policy change and needs no
// engine bump, so a differing `chosen` is normalised away. Changing what
// the engine presents or rolls is an engine change, and is not.
func requireVersionBump(t *testing.T, file string, got []byte) {
	t.Helper()

	want, err := os.ReadFile(file) //nolint:gosec // G304: fixed test-owned paths under testdata/.
	if err != nil {
		return // a new fixture has nothing to bump against
	}

	if string(got) == string(want) {
		return
	}

	before, after := replayShape(t, want), replayShape(t, got)
	if before == after {
		return // only the policy's answers moved
	}

	if oldVersion := versionOf(t, want); oldVersion == chargen.EngineVersion {
		t.Fatalf("%s changes what replay recomputes while engine_version stays %q.\n"+
			"Bump chargen.EngineVersion: without it a record from the previous build passes the "+
			"provenance check and then diverges at some arbitrary event, which describes nothing.",
			file, oldVersion)
	}
}

// replayShape reduces a record to what replay recomputes: the provenance
// versions and the policy's own answers are removed, so what remains is
// the engine's behaviour.
func replayShape(t *testing.T, data []byte) string {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}

	// The three provenance versions are all excluded. engine_version is
	// what this check is about; policy_version is not verified on replay
	// at all; and schema_version is itself a gate checkProvenance
	// compares, so a schema bump on its own already rejects an old record
	// and needs no engine bump behind it.
	delete(record, "schema_version")
	delete(record, "engine_version")
	delete(record, "policy_version")

	events, _ := record["events"].([]any)
	for _, event := range events {
		if choice, ok := event.(map[string]any)["choice"].(map[string]any); ok {
			choice["chosen"] = 0
		}
	}

	shape, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}

	return string(shape)
}

// versionOf reads a stored fixture's engine_version.
func versionOf(t *testing.T, data []byte) string {
	t.Helper()

	var record struct {
		EngineVersion string `json:"engine_version"`
	}

	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}

	return record.EngineVersion
}

// TestGenerateGoldenJSON pins the full seed-1 character record against
// testdata/seed1.json. This is the schema lock and part of the replay
// contract: a diff here means the record format or the seeded generation
// changed, and one of schema_version or engine_version must bump.
func TestGenerateGoldenJSON(t *testing.T) {
	goldenJSON(t, generate(t, chargen.Options{Seed: 1}), "testdata/seed1.json")
}

// TestCareerGoldens pins one full character record per implemented career
// (docs/PRD.md, Testing: golden character fixtures per career). Each
// fixture covers a complete multi-term run of that career; a diff means
// the career's mechanics or the record format changed.
func TestCareerGoldens(t *testing.T) {
	tests := []struct {
		career string
		seed   uint64
		file   string
	}{
		{career: "Citizen", seed: 9, file: "testdata/career_citizen.json"},
		{career: "Scout", seed: 26, file: "testdata/career_scout.json"},
		{career: "Merchant", seed: 17, file: "testdata/career_merchant.json"},
		{career: "Entertainer", seed: 572, file: "testdata/career_entertainer.json"},
		{career: "Scholar", seed: 23, file: "testdata/career_scholar.json"},
		{career: "Noble", seed: 2978, file: "testdata/career_noble.json"},
		{career: "Soldier", seed: 305, file: "testdata/career_soldier.json"},
		{career: "Spacer", seed: 659, file: "testdata/career_spacer.json"},
		{career: "Marine", seed: 529, file: "testdata/career_marine.json"},
		{career: "Agent", seed: 1717, file: "testdata/career_agent.json"},
		{career: "Rogue", seed: 39, file: "testdata/career_rogue.json"},
	}

	for _, tt := range tests {
		t.Run(tt.career, func(t *testing.T) {
			goldenJSON(t, generate(t, chargen.Options{Seed: tt.seed, Career: tt.career}), tt.file)
		})
	}
}

// TestGenerateProvenance verifies the provenance header the replay contract
// requires on every record (docs/PRD.md): schema_version, ruleset,
// engine_version, policy_version, and rng.
func TestGenerateProvenance(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 42, Name: "Eneri Dinsha"})

	if c.SchemaVersion != chargen.SchemaVersion || c.Ruleset != chargen.Ruleset ||
		c.EngineVersion != chargen.EngineVersion || c.PolicyVersion != chargen.PolicyVersion {
		t.Errorf("provenance header = %+v", c)
	}

	if c.RNG != (chargen.RNG{Algorithm: chargen.RNGAlgorithm, Seed: 42}) {
		t.Errorf("rng = %+v", c.RNG)
	}

	if c.Name != "Eneri Dinsha" {
		t.Errorf("name = %q", c.Name)
	}
}

// TestGenerateDerivedUPP verifies the stored UPP is the value derived from
// the stored characteristics (docs/PRD.md, JSON conventions: derived values
// are stored and recomputed on replay).
func TestGenerateDerivedUPP(t *testing.T) {
	for seed := range uint64(20) {
		c := generate(t, chargen.Options{Seed: seed})

		if c.UPP != c.Characteristics.UPP() {
			t.Errorf("seed %d: stored UPP %q != derived %q", seed, c.UPP, c.Characteristics.UPP())
		}
	}
}

// TestEveryTermElapsesFourYears pins the rule that a term costs its four
// years however it ends: "the 4-year Term" (p. 66). A disabled character
// "Musters Out at Term end" (chart 05 p. 79) and a dead one stops at the
// injury (p. 65) — both complete the term, and neither reaches the
// Continue throw that used to be the only thing advancing the clock. The
// three cases below are the fixtures that carry those flags; before the
// age advance was centralized each of them lost four years silently,
// because no fixture can notice an event that was never emitted.
func TestEveryTermElapsesFourYears(t *testing.T) {
	tests := []struct {
		name     string
		career   string
		seed     uint64
		dead     bool
		disabled bool
	}{
		{name: "disabled ends the career", career: "Soldier", seed: 22, disabled: true},
		{name: "disabled in the Armed Forces", career: "Spacer", seed: 9, disabled: true},
		{name: "death at the injury", career: "Scholar", seed: 23, dead: true},
		// Aging kills through a different path: the term reaches its
		// Continue throw, and the years it elapses are what prove fatal.
		{name: "death by aging", career: "Rogue", seed: 39, dead: true},
		{name: "an ordinary career", career: "Citizen", seed: 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := generate(t, chargen.Options{Seed: tt.seed, Career: tt.career})

			// Guard the premise: a fixture that stopped dying would make
			// this test pass while testing nothing.
			if c.Dead != tt.dead || c.Disabled != tt.disabled {
				t.Fatalf("seed %d %s: dead=%v disabled=%v, want dead=%v disabled=%v",
					tt.seed, tt.career, c.Dead, c.Disabled, tt.dead, tt.disabled)
			}

			terms := 0
			for _, career := range c.Careers {
				terms += len(career.Terms)
			}

			got := 0

			for _, e := range c.Events {
				if e.Kind == chargen.EventConsequence &&
					e.Consequence.Kind == chargen.ConsequenceYearsElapsed &&
					e.Consequence.Value == chargen.TermYears {
					got++
				}
			}

			if got != terms {
				t.Errorf("%d terms served but %d elapsed four years", terms, got)
			}
		})
	}
}

// TestYearsElapsedNamesAThrow holds the FR10 contract across the paths
// that end a career without a Continue throw: the added consequence must
// name the Risk throw that ended the term, never the step and never
// sequence zero (docs/PRD.md FR10).
func TestYearsElapsedNamesAThrow(t *testing.T) {
	for _, tt := range []struct {
		career string
		seed   uint64
	}{
		{career: "Scholar", seed: 23},
		{career: "Marine", seed: 529},
		{career: "Soldier", seed: 305},
	} {
		t.Run(tt.career, func(t *testing.T) {
			c := generate(t, chargen.Options{Seed: tt.seed, Career: tt.career})

			kinds := make(map[int]chargen.EventKind, len(c.Events))
			for _, e := range c.Events {
				kinds[e.Seq] = e.Kind
			}

			for _, e := range c.Events {
				if e.Kind != chargen.EventConsequence ||
					e.Consequence.Kind != chargen.ConsequenceYearsElapsed {
					continue
				}

				if kind := kinds[e.Consequence.Cause]; kind != chargen.EventThrow {
					t.Errorf("seq %d: years_elapsed caused by %q, want a throw", e.Seq, kind)
				}
			}
		})
	}
}
