package chargen

// The character's birthdate (Book 1 p. 58; p. 263), which is the last
// thing generation produces.

import (
	"errors"
	"fmt"

	"github.com/philoserf/t5chargen/calendar"
	"github.com/philoserf/t5chargen/dice"
)

// ErrCurrentYear reports a current year that cannot yield a birthdate:
// negative, or earlier than the character's own age.
var ErrCurrentYear = errors.New("current year cannot yield a birth year")

// birthdateDice is the throw p. 263 calls for: "Roll four consecutive dice
// to determine the specific day/date of the year." They are read as four
// separate dice — the half of the year, then the row across two of them,
// then the column — so only the faces are used and the total is
// meaningless.
const birthdateDice = 4

// birthdate settles the character's birthdate: "At the end of character
// generation, subtract the character's age at muster out from 1105 to
// determine his birth year, and randomly determine the specific birth day
// from the Imperial Calendar" (p. 58).
//
// Every character gets one, the dead included — "Every character has a
// birthdate" (p. 263) — unlike muster out, which a dead character does not
// reach (interpretation I-77). The step runs last because p. 58 puts it
// last: "Until Character Generation is complete, Birthdate calculation may
// be deferred".
func (c *Character) birthdate(roller *dice.Roller, log *Log, currentYear int) error {
	table, err := calendar.Load()
	if err != nil {
		return fmt.Errorf("birthdate: %w", err)
	}

	// "The default date (if this information is not otherwise provided)
	// is 001-1105" (p. 58). Zero is "not provided" here, the same way the
	// all-zero Homeworld takes the tool-owned default (docs/PRD.md FR2);
	// the CLI rejects an explicitly typed 0 rather than reading it as the
	// default, since a referee who types a year means it.
	year := currentYear
	if year == 0 {
		year = calendar.DefaultYear
	}

	if year < 0 {
		return fmt.Errorf("%w: current year %d", ErrCurrentYear, year)
	}

	// "subtract the character's age at muster out from 1105 to determine
	// his birth year" (p. 58). A current year the character has outlived
	// is not a date the calendar can express, and repairing it would mint
	// a birthdate that is not one.
	if year-c.Age < 1 {
		return fmt.Errorf("%w: age %d in year %d leaves birth year %d",
			ErrCurrentYear, c.Age, year, year-c.Age)
	}

	log.Step("Record the Character's Birthdate", "Book 1 p. 58; p. 263")

	day, cause := 0, 0

	// "Die 1= 4, Die2= 3, Die3=1, Die4=6. Reroll!" (p. 263). An RR cell
	// sends the whole throw back, and the table has 67 of them.
	for day == calendar.Reroll {
		roll := roller.Roll(birthdateDice)
		cause = log.Roll(roll, "Book 1 p. 263 (Birth Date Generation; 4 consecutive dice)")

		day, err = table.Day(roll.Faces[0], roll.Faces[1], roll.Faces[2], roll.Faces[3])
		if err != nil {
			return fmt.Errorf("birthdate: %w", err)
		}
	}

	// "Each character computes his or her birthdate by subtracting Age
	// from the current year" (p. 263).
	born, err := calendar.On(day, year-c.Age)
	if err != nil {
		return fmt.Errorf("birthdate: %w", err)
	}

	c.Birthdate = born.String()

	log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceBirthdate, Detail: c.Birthdate, Value: born.Year,
	})

	return nil
}
