package chargen

import "github.com/philoserf/t5chargen/dice"

// Provenance constants for the replay and provenance contract (docs/PRD.md):
// every character record carries them so old characters stay auditable after
// embedded tables change. Changing the RNG algorithm, the seeded stream's
// consumption order, or the default policy is a version bump; EngineVersion
// is hand-bumped in v1 (no build-info plumbing).
const (
	// SchemaVersion identifies the character JSON schema.
	SchemaVersion = "0.1.0"

	// Ruleset is pinned: all rule citations resolve against this artifact.
	Ruleset = "Traveller5 Core Rules Book 1, Print Edition 5.1"

	// EngineVersion identifies this implementation of the generation
	// procedure, including the seeded stream's consumption order.
	EngineVersion = "0.1.0"

	// PolicyVersion is "none" until the auto-mode default policy lands;
	// it then becomes the POLICY.md version (docs/PRD.md, CLI sketch).
	PolicyVersion = "none"

	// RNGAlgorithm names the recorded random stream: Go math/rand/v2 PCG,
	// seeded as documented at dice.New. The exact string is compared on
	// replay; changing it is a version bump.
	RNGAlgorithm = "math/rand/v2-pcg"
)

// RNG records the random stream a character was generated from
// (docs/PRD.md, Replay and provenance contract).
type RNG struct {
	Algorithm string `json:"algorithm"`
	Seed      uint64 `json:"seed"`
}

// Character is the character record (docs/PRD.md FR8). The JSON record is
// the source of truth; rendered sheets are derived from it. UPP is a stored
// derived value (docs/PRD.md, JSON conventions): replay recomputes and
// compares it.
type Character struct {
	SchemaVersion string   `json:"schema_version"`
	Ruleset       string   `json:"ruleset"`
	EngineVersion string   `json:"engine_version"`
	PolicyVersion string   `json:"policy_version"`
	RNG           RNG      `json:"rng"`
	Errata        []string `json:"errata,omitempty"` // applied ERRATA.md deviations

	Name            string          `json:"name,omitempty"` // blank by default (docs/PRD.md, Decisions)
	Characteristics Characteristics `json:"characteristics"`
	UPP             string          `json:"upp"`

	Events []Event `json:"events"`
}

// Generate runs the generation procedure from a seed and returns the
// character record. It currently covers checklist step A (chart E1, p. 72);
// later steps are added chunk by chunk per docs/PRD.md milestone 1.
func Generate(seed uint64, name string) Character {
	roller := dice.New(seed)

	var log Log

	log.Step("Generate Characteristics", "Book 1 p. 72 chart E1 step A")
	characteristics := RollCharacteristics(roller, &log)

	return Character{
		SchemaVersion:   SchemaVersion,
		Ruleset:         Ruleset,
		EngineVersion:   EngineVersion,
		PolicyVersion:   PolicyVersion,
		RNG:             RNG{Algorithm: RNGAlgorithm, Seed: seed},
		Name:            name,
		Characteristics: characteristics,
		UPP:             characteristics.UPP(),
		Events:          log.Events(),
	}
}
