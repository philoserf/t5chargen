// Package calendar is the Imperial Calendar and the Birth Date Generation
// table (Book 1 pp. 262-263).
//
// A character's birthdate is the last thing character generation produces:
// "At the end of character generation, subtract the character's age at
// muster out from 1105 to determine his birth year, and randomly determine
// the specific birth day from the Imperial Calendar" (p. 58).
package calendar

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// DefaultYear is the current date the Referee supplies when nothing else
// is given: "The default date (if this information is not otherwise
// provided) is 001-1105 - the first day of the 1105th year of the Imperial
// calendar" (p. 58). p. 263 names it The Golden Age.
const DefaultYear = 1105

// DaysInYear is the length of the Imperial year: "The Imperial Calendar
// numbers the days of each year from 1 to 365" (p. 262).
const DaysInYear = 365

// Reroll is the table's RR cell, which sends the whole four-dice throw
// back: "Die 1= 4, Die2= 3, Die3=1, Die4=6. Reroll!" (p. 263).
const Reroll = 0

// weekdays are the seven day names of p. 262, starting from day 2. Day 1
// stands outside them: "To make 7-day weeks fit evenly into 365-day years,
// the calendar has established the first day of the year as Holiday, a
// specially-named day of celebration".
var weekdays = [7]string{"Wonday", "Tuday", "Thirday", "Forday", "Fiday", "Sixday", "Senday"}

// holiday is day 1's name (p. 262).
const holiday = "Holiday"

// Row is one row of the Birth Date Generation table, reached by the second
// and third dice. Low is read when the first die is 1-2-3 and High when it
// is 4-5-6; the fourth die picks the column.
type Row struct {
	Dice [2]int `json:"dice"`
	Low  [6]int `json:"low"`
	High [6]int `json:"high"`
}

// Table is p. 263's Birth Date Generation.
type Table struct {
	Cite        string `json:"cite"`
	Instruction string `json:"instruction"`
	Reroll      int    `json:"reroll"`
	Rows        []Row  `json:"rows"`
}

//go:embed data/birthdate.json
var birthdateJSON []byte

var errBadTable = errors.New("invalid birthdate table")

// ErrDice reports a throw outside 1-6.
var ErrDice = errors.New("calendar: die outside 1-6")

// Load returns the embedded Birth Date Generation table.
func Load() (*Table, error) { return table() }

var table = sync.OnceValues(func() (*Table, error) {
	var t Table
	if err := json.Unmarshal(birthdateJSON, &t); err != nil {
		return nil, fmt.Errorf("birthdate table: %w", err)
	}

	if err := t.validate(); err != nil {
		return nil, err
	}

	return &t, nil
})

// Day reads the table with four consecutive dice and returns the day of
// the year, or Reroll for an RR cell: "Roll four consecutive dice to
// determine the specific day/date of the year" (p. 263).
//
// The first die selects the half of the year, the second and third the
// row, and the fourth the column.
func (t *Table) Day(d1, d2, d3, d4 int) (int, error) {
	for _, d := range [4]int{d1, d2, d3, d4} {
		if d < 1 || d > 6 {
			return 0, fmt.Errorf("%w: %d", ErrDice, d)
		}
	}

	row := t.Rows[(d2-1)*6+(d3-1)]

	if d1 <= 3 {
		return row.Low[d4-1], nil
	}

	return row.High[d4-1], nil
}

// Birthdate is a day of the Imperial Calendar, as p. 263 writes one —
// "Sigg's birthdate is Wonday 044-1075".
type Birthdate struct {
	// Day is the day of the year, 1 to 365.
	Day int `json:"day"`

	// Year is the Imperial year: "Each character computes his or her
	// birthdate by subtracting Age from the current year" (p. 263).
	Year int `json:"year"`

	// Weekday names the day, or is "Holiday" for day 1.
	Weekday string `json:"weekday"`
}

// String renders the birthdate the way p. 263 writes it.
func (b Birthdate) String() string {
	return fmt.Sprintf("%s %03d-%d", b.Weekday, b.Day, b.Year)
}

// ErrDay reports a day of the year outside 1 through DaysInYear.
var ErrDay = errors.New("calendar: day outside the year")

// unknownWeekday is what Weekday answers for a day outside the year.
const unknownWeekday = "Unknown"

// On builds the birthdate for a day of the year and an Imperial year.
func On(day, year int) (Birthdate, error) {
	// ErrDay, not errBadTable: the caller passed a day the year does not
	// have, which is nothing to do with the transcription being corrupt.
	if day < 1 || day > DaysInYear {
		return Birthdate{}, fmt.Errorf("%w: day %d", ErrDay, day)
	}

	return Birthdate{Day: day, Year: year, Weekday: Weekday(day)}, nil
}

// Weekday names a day of the year. Day 1 is Holiday and stands outside the
// week; day 2 begins it on Wonday (p. 262).
func Weekday(day int) string {
	if day == 1 {
		return holiday
	}

	// Below day 1 the modulo goes negative and indexes out of range; above
	// DaysInYear it wrapped silently into the next year's week. On
	// range-checks before calling, so neither is reachable through the
	// package's own door — but Weekday is exported and takes the caller's
	// number, and an out-of-range day is the caller's error to see rather
	// than a panic or a plausible wrong answer.
	if day < 1 || day > DaysInYear {
		return unknownWeekday
	}

	return weekdays[(day-2)%len(weekdays)]
}

// validate rejects a transcription the reads assume is well-formed: the 36
// rows in dice order, and every day of the year printed exactly once.
func (t *Table) validate() error {
	if t.Reroll != Reroll {
		return fmt.Errorf("%w: reroll marker is %d, want %d", errBadTable, t.Reroll, Reroll)
	}

	if len(t.Rows) != 36 {
		return fmt.Errorf("%w: %d rows, want 36", errBadTable, len(t.Rows))
	}

	for i, row := range t.Rows {
		if want := [2]int{i/6 + 1, i%6 + 1}; row.Dice != want {
			return fmt.Errorf("%w: row %d is dice %v, want %v", errBadTable, i+1, row.Dice, want)
		}
	}

	return t.validateDays()
}

// validateDays checks the printed cells cover the year exactly once. This
// is the check that earns its keep: the table is 432 hand-read cells, and
// a misread digit either duplicates a day or loses one.
func (t *Table) validateDays() error {
	seen := make(map[int]bool, DaysInYear)
	rerolls := 0

	for _, row := range t.Rows {
		for _, half := range [2][6]int{row.Low, row.High} {
			for _, day := range half {
				if day == Reroll {
					rerolls++

					continue
				}

				if day < 1 || day > DaysInYear {
					return fmt.Errorf("%w: day %d is not in the year", errBadTable, day)
				}

				if seen[day] {
					return fmt.Errorf("%w: day %d printed twice", errBadTable, day)
				}

				seen[day] = true
			}
		}
	}

	if len(seen) != DaysInYear {
		return fmt.Errorf("%w: %d days of %d printed", errBadTable, len(seen), DaysInYear)
	}

	// 36 rows of 12 cells, less the 365 days, is what must be left.
	if want := 36*12 - DaysInYear; rerolls != want {
		return fmt.Errorf("%w: %d reroll cells, want %d", errBadTable, rerolls, want)
	}

	return nil
}
