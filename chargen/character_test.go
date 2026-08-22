package chargen_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// TestGenerateGoldenJSON pins the full seed-1 character record against
// testdata/seed1.json. This is the schema lock and part of the replay
// contract: a diff here means the record format or the seeded generation
// changed, and one of schema_version or engine_version must bump.
func TestGenerateGoldenJSON(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 1})

	got, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	got = append(got, '\n')

	want, err := os.ReadFile("testdata/seed1.json")
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != string(want) {
		t.Errorf("seed 1 record differs from testdata/seed1.json:\n%s", got)
	}
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
	}

	for _, tt := range tests {
		t.Run(tt.career, func(t *testing.T) {
			c := generate(t, chargen.Options{Seed: tt.seed, Career: tt.career})

			got, err := json.MarshalIndent(c, "", "  ")
			if err != nil {
				t.Fatal(err)
			}

			got = append(got, '\n')

			want, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatal(err)
			}

			if string(got) != string(want) {
				t.Errorf("%s record differs from %s:\n%s", tt.career, tt.file, got)
			}
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
