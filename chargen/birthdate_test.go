package chargen_test

// The birthdate, Book 1 p. 58 and pp. 262-263.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/calendar"
	"github.com/philoserf/t5chargen/chargen"
)

// TestEveryCharacterIsBornBeforeHeLived pins the arithmetic p. 263 states:
// "Each character computes his or her birthdate by subtracting Age from the
// current year." The birth year is therefore the current year less the age,
// on every seed, whatever the lifepath did.
func TestEveryCharacterIsBornBeforeHeLived(t *testing.T) {
	t.Parallel()

	for seed := uint64(1); seed <= 200; seed++ {
		c := generate(t, chargen.Options{Seed: seed})

		day, year := parseBirthdate(t, c.Birthdate)

		if want := calendar.DefaultYear - c.Age; year != want {
			t.Fatalf("seed %d: age %d in %d gives birth year %d, want %d",
				seed, c.Age, calendar.DefaultYear, year, want)
		}

		if day < 1 || day > calendar.DaysInYear {
			t.Fatalf("seed %d: born on day %d, outside the year", seed, day)
		}

		if got := calendar.Weekday(day); !strings.HasPrefix(c.Birthdate, got+" ") {
			t.Fatalf("seed %d: %q, but day %d is a %s", seed, c.Birthdate, day, got)
		}
	}
}

// TestTheRefereesYearMovesTheBirthYearAndNothingElse pins p. 58's "The
// Referee provides the current date of the beginning of adventuring."
// Changing it must move the birth year and leave the character alone: the
// year is arithmetic applied after the dice, not an input to them.
func TestTheRefereesYearMovesTheBirthYearAndNothingElse(t *testing.T) {
	t.Parallel()

	const other = 1248 // "The New Era 001-1248" (p. 263)

	for seed := uint64(1); seed <= 50; seed++ {
		golden := generate(t, chargen.Options{Seed: seed})
		shifted := generate(t, chargen.Options{Seed: seed, CurrentYear: other})

		goldenDay, goldenYear := parseBirthdate(t, golden.Birthdate)
		shiftedDay, shiftedYear := parseBirthdate(t, shifted.Birthdate)

		if goldenDay != shiftedDay {
			t.Fatalf("seed %d: the year moved the day from %d to %d", seed, goldenDay, shiftedDay)
		}

		if shiftedYear-goldenYear != other-calendar.DefaultYear {
			t.Fatalf("seed %d: born %d then %d; the years differ by %d, want %d",
				seed, goldenYear, shiftedYear, shiftedYear-goldenYear, other-calendar.DefaultYear)
		}

		if golden.UPP != shifted.UPP || golden.Age != shifted.Age || len(golden.Skills) != len(shifted.Skills) {
			t.Fatalf("seed %d: the Referee's year changed the character, not just his birth year", seed)
		}
	}
}

// TestEvenTheDeadAreBorn pins "Every character has a birthdate" (p. 263).
// Muster out is gated on survival (interpretation I-77) and this is not:
// dying does not unmake the day he was born (interpretation I-86).
func TestEvenTheDeadAreBorn(t *testing.T) {
	t.Parallel()

	dead := 0

	for seed := uint64(1); seed <= 400; seed++ {
		c := generate(t, chargen.Options{Seed: seed})
		if !c.Dead {
			continue
		}

		dead++

		if c.Birthdate == "" {
			t.Fatalf("seed %d: a dead character with no birthdate", seed)
		}
	}

	if dead == 0 {
		t.Fatal("no character in 400 seeds died; the test proves nothing")
	}
}

// parseBirthdate splits "Wonday 044-1075" into its day and year.
func parseBirthdate(t *testing.T, s string) (int, int) {
	t.Helper()

	_, date, ok := strings.Cut(s, " ")
	if !ok {
		t.Fatalf("birthdate %q has no weekday", s)
	}

	dayText, yearText, ok := strings.Cut(date, "-")
	if !ok {
		t.Fatalf("birthdate %q has no year", s)
	}

	day, err := strconv.Atoi(dayText)
	if err != nil {
		t.Fatalf("birthdate %q: %v", s, err)
	}

	year, err := strconv.Atoi(yearText)
	if err != nil {
		t.Fatalf("birthdate %q: %v", s, err)
	}

	return day, year
}
