// Package world holds the homeworld data the engine consumes: UWP
// validation, the Homeworld type, and the chart B homeworld-skills table
// (Book 1 p. 56;
// prose p. 58). Per the data/logic boundary (docs/PRD.md, Architecture
// notes) this package is data and validation only.
//
// Homeworld skills key off Trade Classifications, not the UWP:
// "World descriptions include Trade Classifications and Remarks (TC&R). A
// character receives one specified skill for each Trade Classification or
// Remark from the homeworld." (p. 58) A bare UWP therefore grants no
// homeworld skills; TC derivation from a UWP is Book 2/3 material and out
// of scope while the ruleset is pinned to Book 1 (docs/PRD.md FR2 note).
package world

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/philoserf/t5chargen/ehex"
	"github.com/philoserf/t5chargen/skill"
)

// Homeworld identifies where a character was raised (chart B, p. 56).
type Homeworld struct {
	Name string `json:"name,omitempty"` // display name; optional for supplied worlds
	UWP  string `json:"uwp"`

	// DeepSpace marks the one chart B cell that names no world: "Born In
	// Deep Space" (p. 56, interpretation I-97). It is the only homeworld
	// permitted to carry no UWP, and it is set by the chart rather than
	// by a caller — trade classifications without a UWP remain a partial
	// world and are still rejected (docs/PRD.md FR2).
	DeepSpace bool `json:"deep_space,omitempty"`

	// TradeClassifications drive the homeworld skill grants (p. 58), in
	// chart order. They are supplied with the world (chart B's own list
	// carries them per world); they are not derived from the UWP.
	TradeClassifications []string `json:"trade_classifications,omitempty"`
}

// Default is the tool-owned default homeworld, fixed in the tool's data
// files (docs/PRD.md FR2): Regina, the book's own worked example ("from:
// Regina (1910 Spinward Marches)", p. 58; chart B row R, p. 56). The value
// lives in data/homeworld_skills.json per the data/logic boundary.
func Default() (Homeworld, error) {
	t, err := table()
	if err != nil {
		return Homeworld{}, err
	}

	d := t.Default
	d.TradeClassifications = slices.Clone(d.TradeClassifications)

	return d, nil
}

// Label renders the homeworld for display and choice events: the name (when
// there is one), the UWP, and the trade classifications.
// Assembled from the parts that are present rather than by appending to
// whatever came before: chart B's deep space cell carries no UWP (I-97),
// and a homeworld with no name either would otherwise open its label with
// the separator space — which lands in the character sheet and in the
// permanent choice event.
func (h Homeworld) Label() string {
	var parts []string

	if h.Name != "" {
		parts = append(parts, h.Name)
	}

	if h.UWP != "" {
		parts = append(parts, h.UWP)
	}

	if len(h.TradeClassifications) > 0 {
		parts = append(parts, "("+strings.Join(h.TradeClassifications, " ")+")")
	}

	return strings.Join(parts, " ")
}

// Validate checks a homeworld is well-formed: a valid UWP and known,
// unrepeated trade classifications. "A character receives one specified
// skill for each Trade Classification or Remark from the homeworld."
// (p. 58) — a world carries each TC at most once, so a repeated TC is
// invalid input, rejected rather than silently deduplicated or
// double-granted (docs/PRD.md FR2).
func (h Homeworld) Validate() error {
	// Chart B's last cell is "Born In Deep Space" and names no world, so
	// it carries no UWP (p. 56, interpretation I-97). Only that cell may
	// omit one, and it says so outright: trade classifications without a
	// UWP are still the partial world docs/PRD.md FR2 refuses.
	if !h.DeepSpace {
		if err := ValidateUWP(h.UWP); err != nil {
			return err
		}
	} else if err := h.validateDeepSpace(); err != nil {
		return err
	}

	seen := make(map[string]bool, len(h.TradeClassifications))

	for _, tc := range h.TradeClassifications {
		if _, err := GrantFor(tc); err != nil {
			return err
		}

		if seen[tc] {
			return fmt.Errorf("%w: %q", ErrDuplicateTC, tc)
		}

		seen[tc] = true
	}

	return nil
}

// ErrInvalidUWP reports a malformed UWP: "Invalid or partial UWPs are
// rejected with an error, never silently repaired" (docs/PRD.md FR2).
var ErrInvalidUWP = errors.New("invalid UWP")

// ErrDuplicateTC reports a trade classification supplied more than once
// for a single world (each grants "one specified skill", p. 58).
var ErrDuplicateTC = errors.New("duplicate trade classification")

// starports are the Book 1 starport classes (chart B worlds use A-E and X).
const starports = "ABCDEX"

// ValidateUWP validates a UWP string structurally: a starport class, six
// eHex digits (p. 22), a hyphen, and an eHex tech level (as in chart B's
// worlds, for example A788899-C). Book 1 gives the format by example, so
// the validation is structural against the eHex definition rather than
// field-range rules. The string is never normalized or repaired
// (docs/PRD.md FR2).
func ValidateUWP(uwp string) error {
	if len(uwp) != 9 {
		return fmt.Errorf("%w: %q: want 9 characters like A788899-C", ErrInvalidUWP, uwp)
	}

	if !strings.ContainsRune(starports, rune(uwp[0])) {
		return fmt.Errorf("%w: %q: starport must be one of A B C D E X", ErrInvalidUWP, uwp)
	}

	if uwp[7] != '-' {
		return fmt.Errorf("%w: %q: want '-' before the tech level", ErrInvalidUWP, uwp)
	}

	for _, i := range []int{1, 2, 3, 4, 5, 6, 8} {
		if _, err := ehex.Decode(uwp[i]); err != nil {
			return fmt.Errorf("%w: %q position %d: %w", ErrInvalidUWP, uwp, i, err)
		}
	}

	return nil
}

// GrantKind discriminates what a trade classification grants (chart B,
// p. 56).
type GrantKind string

// The grant kinds of chart B.
const (
	// GrantSkill awards the listed skills (usually one; Ds Deep Space
	// awards both Vacc Suit and Zero-G).
	GrantSkill GrantKind = "skill"

	// GrantNone is a "(no skill)" row.
	GrantNone GrantKind = "none"

	// GrantArt is Ri Rich: "One Art (Choose One): Actor, Artist, Author,
	// Chef, Dancer, Musician" (p. 56).
	GrantArt GrantKind = "art"

	// GrantTrade is In Industrial: "The Trades (Choose One): Biologics,
	// Craftsman, Electronics, Fluidics, Gravitics, Magnetics, Mechanic,
	// Photonics, Polymers, Programmer" (p. 56).
	GrantTrade GrantKind = "trade"
)

// Grant is one trade classification's homeworld skill row.
type Grant struct {
	TC     string    `json:"tc"`
	Label  string    `json:"label"`
	Kind   GrantKind `json:"grant"`
	Skills []string  `json:"skills,omitempty"`
}

// skillsTable is the parsed chart B data.
type skillsTable struct {
	Cite     string    `json:"cite"`
	Default  Homeworld `json:"default"`
	OneArt   []string  `json:"one_art"`
	OneTrade []string  `json:"one_trade"`
	Skills   []Grant   `json:"skills"`
}

//go:embed data/homeworld_skills.json
var skillsJSON []byte

// table parses and validates the embedded chart B data once.
var table = sync.OnceValues(func() (*skillsTable, error) {
	var t skillsTable
	if err := json.Unmarshal(skillsJSON, &t); err != nil {
		return nil, fmt.Errorf("world: parsing homeworld_skills.json: %w", err)
	}

	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("world: homeworld_skills.json: %w", err)
	}

	return &t, nil
})

// errBadTable reports invalid chart B data.
var errBadTable = errors.New("invalid homeworld skills table")

// validate rejects malformed-but-parseable chart B data at load time.
func (t *skillsTable) validate() error {
	if len(t.OneArt) == 0 || len(t.OneTrade) == 0 || len(t.Skills) == 0 {
		return fmt.Errorf("%w: empty selection or skills list", errBadTable)
	}

	seen := map[string]bool{}

	for _, grant := range t.Skills {
		if grant.TC == "" || seen[grant.TC] {
			return fmt.Errorf("%w: missing or duplicate TC %q", errBadTable, grant.TC)
		}

		seen[grant.TC] = true

		if err := grant.validate(); err != nil {
			return err
		}
	}

	return t.validateDefault(seen)
}

// validateDefault checks the tool-owned default homeworld (docs/PRD.md
// FR2): a named world, a well-formed UWP, and chart B TCs. Checked against
// the seen set directly — Homeworld.Validate would re-enter the table
// load.
func (t *skillsTable) validateDefault(seen map[string]bool) error {
	if t.Default.Name == "" {
		return fmt.Errorf("%w: default homeworld missing a name", errBadTable)
	}

	if err := ValidateUWP(t.Default.UWP); err != nil {
		return fmt.Errorf("%w: default homeworld: %w", errBadTable, err)
	}

	defaultSeen := map[string]bool{}

	for _, tc := range t.Default.TradeClassifications {
		if !seen[tc] || defaultSeen[tc] {
			return fmt.Errorf("%w: default homeworld TC %q unknown or repeated", errBadTable, tc)
		}

		defaultSeen[tc] = true
	}

	return nil
}

// validate checks one grant row.
func (g Grant) validate() error {
	switch g.Kind {
	case GrantSkill:
		if len(g.Skills) == 0 {
			return fmt.Errorf("%w: TC %q grants skill but lists none", errBadTable, g.TC)
		}

		// Chart B names Master Skill List entries (ERRATA.md I-9).
		// Ambiguous names are rejected rather than merely validated: the
		// homeworld grants award directly, with no choice point to resolve
		// a label covering several entries the way the career charts do
		// (ERRATA.md I-10, I-11). No chart B grant is ambiguous.
		for _, name := range g.Skills {
			if _, err := skill.Resolve(name); err != nil {
				return fmt.Errorf("%w: TC %q: %w", errBadTable, g.TC, err)
			}
		}
	case GrantNone, GrantArt, GrantTrade:
		if len(g.Skills) != 0 {
			return fmt.Errorf("%w: TC %q kind %q must not list skills", errBadTable, g.TC, g.Kind)
		}
	default:
		return fmt.Errorf("%w: TC %q has unknown grant kind %q", errBadTable, g.TC, g.Kind)
	}

	return nil
}

// ErrUnknownTC reports a trade classification outside chart B.
var ErrUnknownTC = errors.New("unknown trade classification")

// GrantFor returns the chart B row for a trade classification.
func GrantFor(tc string) (Grant, error) {
	t, err := table()
	if err != nil {
		return Grant{}, err
	}

	for _, grant := range t.Skills {
		if grant.TC == tc {
			return grant, nil
		}
	}

	return Grant{}, fmt.Errorf("%w: %q", ErrUnknownTC, tc)
}

// ArtChoices returns the "One Art (Choose One)" alternatives in chart
// order (p. 56). The returned slice is shared; callers must not mutate it.
func ArtChoices() ([]string, error) {
	t, err := table()
	if err != nil {
		return nil, err
	}

	return t.OneArt, nil
}

// TradeChoices returns "The Trades (Choose One)" alternatives in chart
// order (p. 56). The returned slice is shared; callers must not mutate it.
func TradeChoices() ([]string, error) {
	t, err := table()
	if err != nil {
		return nil, err
	}

	return t.OneTrade, nil
}

// validateDeepSpace holds the deep space mark to what chart B's cell
// actually is. The mark exempts a homeworld from carrying a UWP, so
// without this it is a way to skip validation by asserting it: a
// hand-written record claiming deep space with arbitrary trade
// classifications would pass.
//
// A deep space birth carries Ds and no UWP, which is what the book shows
// twice over — chart B's cell prints exactly that, and p. 58 says such a
// character "naturally learns the skills Zero-G and Vacc Suit", which is
// the Ds grant. Requiring it is reading the cell, not inventing a rule.
func (h Homeworld) validateDeepSpace() error {
	if h.UWP != "" {
		return fmt.Errorf("%w: a deep space birth carries the UWP %q", ErrInvalidUWP, h.UWP)
	}

	if !slices.Contains(h.TradeClassifications, deepSpaceTC) {
		return fmt.Errorf("%w: a deep space birth without the %s trade classification",
			ErrInvalidUWP, deepSpaceTC)
	}

	return nil
}

// deepSpaceTC is chart B's trade classification for a world that is not
// one: "Ds Deep Space — Vacc Suit +Zero-G" (p. 56).
const deepSpaceTC = "Ds"
