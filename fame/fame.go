// Package fame holds chart F (Book 1 p. 91), which is a calculation over a
// finished character rather than a running total: "Current Fame for an
// individual is based on a variety of accomplishments. For example, Rogue
// with one Failed Scheme (and no other applicable factors) has Fame =
// 1 x 3 = 3."
//
// Per the data/logic boundary (docs/PRD.md, Architecture notes) the tables
// are embedded data; this file is lookup and arithmetic only. Which
// accomplishments a character has is the engine's business, in
// chargen/fame.go.
package fame

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Table is chart F's tables and constants.
type Table struct {
	Cite string `json:"cite"`

	// StackLimit: "A character's Fame is the sum of all Fame points
	// received to 20; beyond 20, only the highest Fame applies."
	StackLimit int `json:"stack_limit"`

	// NoEligibilityDice: "If NO other eligibility, 1D."
	NoEligibilityDice int `json:"no_eligibility_dice"`

	// NobleSocMultiplier: "Imperial Noble Soc x1.5".
	NobleSocMultiplier float64 `json:"noble_soc_multiplier"`

	// Medals are the Fame points each decoration is worth per occurrence
	// ("xN = N Fame points per occurrence"), keyed by the chart M1 code.
	// Exemplary Service is listed at x0: it is famous for nothing.
	Medals map[string]int `json:"medals"`

	// Descriptors name each Fame level, indexed by it: "F= Descriptor",
	// 0 Unknown through 36 All Reality. "A world famous Entertainer has
	// Fame-10: name recognition anywhere on the world on which he
	// performs."
	Descriptors []string `json:"descriptors"`
}

//go:embed data/fame.json
var fameJSON []byte

var errBadTable = errors.New("invalid fame table")

// Load returns the embedded chart F.
func Load() (*Table, error) { return table() }

var table = sync.OnceValues(func() (*Table, error) {
	var t Table
	if err := json.Unmarshal(fameJSON, &t); err != nil {
		return nil, fmt.Errorf("fame table: %w", err)
	}

	if err := t.validate(); err != nil {
		return nil, err
	}

	return &t, nil
})

// Stack combines the Fame points a character has earned: "A character's
// Fame is the sum of all Fame points received to 20; beyond 20, only the
// highest Fame applies."
//
// Read as a cap rather than a cliff (interpretation I-63, ERRATA.md): the
// sum applies up to 20, and a single accomplishment worth more than 20
// carries past it. The alternative — that a total above 20 collapses to
// the largest single source — would let a character's Fame fall because he
// achieved something more, which no reading of "stacks" supports.
func (t *Table) Stack(points []int) int {
	sum, highest := 0, 0

	for _, p := range points {
		sum += p
		highest = max(highest, p)
	}

	return max(min(sum, t.StackLimit), highest)
}

// Descriptor names a Fame level: "Fame is noted as Fame-<level>". Levels
// past the printed table take the last descriptor, there being nothing
// beyond All Reality.
func (t *Table) Descriptor(level int) string {
	if level < 0 {
		return ""
	}

	return t.Descriptors[min(level, len(t.Descriptors)-1)]
}

// MedalPoints returns the Fame a decoration is worth per occurrence, and
// whether the code is one chart F prices.
func (t *Table) MedalPoints(code string) (int, bool) {
	points, ok := t.Medals[code]

	return points, ok
}

// validate rejects embedded data the lookups assume is well-formed.
func (t *Table) validate() error {
	if t.StackLimit < 1 {
		return fmt.Errorf("%w: non-positive stack limit", errBadTable)
	}

	if t.NoEligibilityDice < 1 {
		return fmt.Errorf("%w: non-positive no-eligibility dice", errBadTable)
	}

	if t.NobleSocMultiplier <= 0 {
		return fmt.Errorf("%w: non-positive noble multiplier", errBadTable)
	}

	if len(t.Medals) == 0 {
		return fmt.Errorf("%w: no medals priced", errBadTable)
	}

	// The descriptor list is indexed by Fame level and must at least
	// reach the stacking limit.
	if len(t.Descriptors) <= t.StackLimit {
		return fmt.Errorf("%w: %d descriptors for a limit of %d",
			errBadTable, len(t.Descriptors), t.StackLimit)
	}

	for i, name := range t.Descriptors {
		if name == "" {
			return fmt.Errorf("%w: descriptor %d is empty", errBadTable, i)
		}
	}

	return nil
}
