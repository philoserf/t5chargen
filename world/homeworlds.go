package world

// Chart B's world list: "Select a Homeworld (Spinward Marches)" (p. 56),
// the thirty-six cells "Homeworld. Select or determine a Homeworld" draws
// on. The grant table beside it — which Trade Classification awards which
// skill — is in world.go; this file is the list of worlds themselves.

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
)

//go:embed data/homeworlds.json
var homeworldsJSON []byte

// ChartBWorld is one cell of the chart B world list. Hex, Sector and UWP
// are empty for the one cell that names no world, which marks itself
// (interpretation I-97).
type ChartBWorld struct {
	D1 int `json:"d1"`
	D2 int `json:"d2"`

	// Code is the chart's own one-character row code, "A" through "0".
	// Transcription: the page prints a Code column, and no rule reads it.
	// docs/deaddata_test.go cannot excuse it — the gate matches by field
	// name and Award.Code is read in production, so this field passes the
	// gate for the wrong reason and is documented here instead.
	Code string `json:"code"`

	Name string `json:"name"`

	Hex    string `json:"hex,omitempty"`
	Sector string `json:"sector,omitempty"`
	UWP    string `json:"uwp,omitempty"`

	// DeepSpace marks the one cell that names no world. The data says so
	// outright rather than leaving the code to infer it from the prose
	// beside it, which is a fragile thing to key a rule on.
	DeepSpace bool `json:"deep_space,omitempty"`

	// Note carries the chart's own words for that cell, "Born In Deep
	// Space". Transcription: the page prints it, and no rule reads it.
	Note string `json:"note,omitempty"`

	TradeClassifications []string `json:"trade_classifications"`
}

// chartBTable is the parsed world list.
type chartBTable struct {
	Cite   string        `json:"cite"`
	Note   string        `json:"note"`
	Worlds []ChartBWorld `json:"worlds"`
}

// chartB parses and validates the embedded world list once.
var chartB = sync.OnceValues(func() (*chartBTable, error) {
	var t chartBTable
	if err := json.Unmarshal(homeworldsJSON, &t); err != nil {
		return nil, fmt.Errorf("world: parsing homeworlds.json: %w", err)
	}

	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("world: homeworlds.json: %w", err)
	}

	return &t, nil
})

// errBadWorldList reports invalid chart B world-list data.
var errBadWorldList = errors.New("invalid homeworld list")

// chartBCells is the size of a two-dice table read as D1 then D2.
const chartBCells = 36

// validate rejects a world list the chart would not recognise: every D1/D2
// cell present exactly once, every world named, every Trade Classification
// one the grant table knows, and a UWP wherever a world is named.
func (t *chartBTable) validate() error {
	if len(t.Worlds) != chartBCells {
		return fmt.Errorf("%w: %d cells, want %d", errBadWorldList, len(t.Worlds), chartBCells)
	}

	seen := map[[2]int]bool{}

	for _, w := range t.Worlds {
		if w.D1 < 1 || w.D1 > diceFaces || w.D2 < 1 || w.D2 > diceFaces {
			return fmt.Errorf("%w: %q is at cell %d %d", errBadWorldList, w.Name, w.D1, w.D2)
		}

		if seen[[2]int{w.D1, w.D2}] {
			return fmt.Errorf("%w: cell %d %d appears twice", errBadWorldList, w.D1, w.D2)
		}

		seen[[2]int{w.D1, w.D2}] = true

		if err := w.validate(); err != nil {
			return err
		}
	}

	return nil
}

// diceFaces is the range of one die, which is what indexes the chart.
const diceFaces = 6

// ChartB returns the world list in chart order. The returned slice is
// fresh per call, trade classifications included: the parsed table is a
// process-wide singleton, so handing out its own slices would let one
// caller's mutation reach every later lookup.
func ChartB() ([]ChartBWorld, error) {
	t, err := chartB()
	if err != nil {
		return nil, err
	}

	worlds := make([]ChartBWorld, len(t.Worlds))

	for i, w := range t.Worlds {
		w.TradeClassifications = slices.Clone(w.TradeClassifications)
		worlds[i] = w
	}

	return worlds, nil
}

// At returns the world in the cell the two dice name, D1 then D2 as the
// chart prints them.
func At(d1, d2 int) (Homeworld, error) {
	t, err := chartB()
	if err != nil {
		return Homeworld{}, err
	}

	for _, w := range t.Worlds {
		if w.D1 == d1 && w.D2 == d2 {
			return w.Homeworld(), nil
		}
	}

	return Homeworld{}, fmt.Errorf("%w: no cell %d %d", errBadWorldList, d1, d2)
}

// Homeworld renders a chart B cell as the homeworld the engine carries.
func (w ChartBWorld) Homeworld() Homeworld {
	tcs := make([]string, len(w.TradeClassifications))
	copy(tcs, w.TradeClassifications)

	return Homeworld{Name: w.Name, UWP: w.UWP, DeepSpace: w.DeepSpace, TradeClassifications: tcs}
}

// validate checks one cell.
func (w ChartBWorld) validate() error {
	if w.Name == "" || len(w.TradeClassifications) == 0 {
		return fmt.Errorf("%w: cell %d %d names no world or no trade classifications", errBadWorldList, w.D1, w.D2)
	}

	// A cell that names a world must carry its UWP; the one that names
	// none must not pretend to (I-97).
	if (w.UWP == "") != w.DeepSpace {
		return fmt.Errorf("%w: %q has UWP %q and deep_space %v, which disagree",
			errBadWorldList, w.Name, w.UWP, w.DeepSpace)
	}

	if w.UWP != "" {
		if err := ValidateUWP(w.UWP); err != nil {
			return fmt.Errorf("%w: %q: %w", errBadWorldList, w.Name, err)
		}
	}

	for _, tc := range w.TradeClassifications {
		if _, err := GrantFor(tc); err != nil {
			return fmt.Errorf("%w: %q: %w", errBadWorldList, w.Name, err)
		}
	}

	return nil
}
