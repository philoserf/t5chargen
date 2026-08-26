// Package medal holds the Imperial Medals table (Book 1 p. 70 chart M1),
// which the Armed Forces careers consult on a Reward success: "use the
// unmodified Reward roll to determine the Medal; Mod +1 if an Officer"
// (p. 66).
//
// Per the data/logic boundary (docs/PRD.md, Architecture notes) the table
// is embedded data; this file is lookup logic only.
package medal

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Medal is one row of the table.
type Medal struct {
	Roll int    `json:"roll"`
	Code string `json:"code"`
	Name string `json:"name"`

	// Mod is the promotion modifier the medal carries: "Medals (but not
	// Wound Badges) are Mods for Soldier / Spacer / Marine Promotion"
	// (p. 70).
	Mod int `json:"mod"`
}

// ErrOffTable reports a lookup outside the printed rows.
var ErrOffTable = errors.New("medal: roll outside the table")

// errBadTable reports invalid embedded data.
var errBadTable = errors.New("invalid medals table")

// For returns the medal for a successful unmodified Reward roll, with the
// officer modifier already applied by the caller through officer.
func For(reward int, officer bool) (Medal, error) {
	t, err := table()
	if err != nil {
		return Medal{}, err
	}

	line := reward
	if officer {
		line += t.OfficerDM
	}

	for _, row := range t.Rows {
		if row.Roll == line {
			return row, nil
		}
	}

	return Medal{}, fmt.Errorf("%w: %d", ErrOffTable, line)
}

// Cite returns the table's citation.
func Cite() string {
	t, err := table()
	if err != nil {
		return ""
	}

	return t.Cite
}

// Err reports a failure to load the embedded table.
func Err() error {
	_, err := table()

	return err
}

//go:embed data/medals.json
var medalsJSON []byte

type tableData struct {
	Cite      string  `json:"cite"`
	Note      string  `json:"note"`
	OfficerDM int     `json:"officer_dm"`
	Rows      []Medal `json:"rows"`
}

// table parses and validates the embedded table once.
var table = sync.OnceValues(func() (*tableData, error) {
	var t tableData
	if err := json.Unmarshal(medalsJSON, &t); err != nil {
		return nil, fmt.Errorf("medal: parsing medals.json: %w", err)
	}

	if t.Cite == "" || len(t.Rows) == 0 {
		return nil, fmt.Errorf("%w: missing cite or rows", errBadTable)
	}

	// The officer DM widens the reachable range, so it has to be sane
	// before the span check below can use it.
	if t.OfficerDM < 1 {
		return nil, fmt.Errorf("%w: officer DM is %d", errBadTable, t.OfficerDM)
	}

	for i, row := range t.Rows {
		if row.Code == "" || row.Name == "" {
			return nil, fmt.Errorf("%w: row %d is incomplete", errBadTable, i)
		}

		if i > 0 && row.Roll != t.Rows[i-1].Roll+1 {
			return nil, fmt.Errorf("%w: rows are not consecutive at %d", errBadTable, row.Roll)
		}

		// "Medals (but not Wound Badges) are Mods for Soldier / Spacer /
		// Marine Promotion." A medal with no mod is a medal that does not
		// do the one thing the note says medals do.
		if row.Mod < 1 {
			return nil, fmt.Errorf("%w: %q carries no promotion mod", errBadTable, row.Code)
		}
	}

	// The rows must span every roll that can reach them. The reward throw
	// is 2D, and an officer's result is increased by the officer DM, so
	// the reachable range is 2 through 12+DM. Asserting against that
	// rather than against literals keeps the check honest if the DM moves.
	const minReward, maxReward = 2, 12

	if first, last := t.Rows[0].Roll, t.Rows[len(t.Rows)-1].Roll; first != minReward || last != maxReward+t.OfficerDM {
		return nil, fmt.Errorf("%w: rows cover %d-%d, but a reward roll reaches %d-%d",
			errBadTable, first, last, minReward, maxReward+t.OfficerDM)
	}

	return &t, nil
})
