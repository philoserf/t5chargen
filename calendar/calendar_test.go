package calendar_test

// The Imperial Calendar and Birth Date Generation, Book 1 pp. 262-263.

import (
	"testing"

	"github.com/philoserf/t5chargen/calendar"
)

func load(t *testing.T) *calendar.Table {
	t.Helper()

	table, err := calendar.Load()
	if err != nil {
		t.Fatal(err)
	}

	return table
}

// TestTheTableCoversTheYearExactlyOnce is the check that makes a 432-cell
// hand transcription trustworthy: a misread digit either duplicates a day
// or loses one, and both show here. The loader asserts it too, so this
// test also proves the loader's assertion is reachable.
func TestTheTableCoversTheYearExactlyOnce(t *testing.T) {
	t.Parallel()

	seen, rerolls := sweep(t, load(t))

	// The first die only picks a half, so each cell is reached by three
	// of its six faces.
	for day, count := range seen {
		if count != 3 {
			t.Errorf("day %d is reachable %d ways, want 3", day, count)
		}
	}

	if len(seen) != calendar.DaysInYear {
		t.Errorf("%d days of the year are reachable, want %d", len(seen), calendar.DaysInYear)
	}

	// 6 halves x 36 rows x 6 columns, less three ways to each of 365 days.
	if want := 6*36*6 - 3*calendar.DaysInYear; rerolls != want {
		t.Errorf("%d reroll cells, want %d", rerolls, want)
	}
}

// sweep reads every cell the four dice can reach, counting the days and
// the rerolls.
func sweep(t *testing.T, table *calendar.Table) (map[int]int, int) {
	t.Helper()

	seen := make(map[int]int, calendar.DaysInYear)
	rerolls := 0

	for d1 := 1; d1 <= 6; d1++ {
		for d2 := 1; d2 <= 6; d2++ {
			for d3 := 1; d3 <= 6; d3++ {
				for d4 := 1; d4 <= 6; d4++ {
					day, err := table.Day(d1, d2, d3, d4)
					if err != nil {
						t.Fatal(err)
					}

					if day == calendar.Reroll {
						rerolls++
					} else {
						seen[day]++
					}
				}
			}
		}
	}

	return seen, rerolls
}

// TestSiggOdraLandsOnDayFortyFour is p. 263's own worked example, dice and
// all: "Die 1= 4, Die2= 3, Die3=1, Die4=6. Reroll! Die1=2, Die2=2, Die3=2,
// Die4=2. Day=44. Sigg's birthdate is Wonday 044-1075".
func TestSiggOdraLandsOnDayFortyFour(t *testing.T) {
	t.Parallel()

	table := load(t)

	if day, err := table.Day(4, 3, 1, 6); err != nil || day != calendar.Reroll {
		t.Fatalf("the page's first throw gave day %d (err %v), want a reroll", day, err)
	}

	day, err := table.Day(2, 2, 2, 2)
	if err != nil {
		t.Fatal(err)
	}

	if day != 44 {
		t.Fatalf("the page's second throw gave day %d, want 44", day)
	}

	// Sigg "completes character generation and is Age= 30. The current
	// adventuring year is 1105. Sigg was born in 1105 minus 30= 1075."
	born, err := calendar.On(day, calendar.DefaultYear-30)
	if err != nil {
		t.Fatal(err)
	}

	if got := born.String(); got != "Wonday 044-1075" {
		t.Errorf("the page's example renders as %q, want %q", got, "Wonday 044-1075")
	}
}

// TestTheWeekRunsFromDayTwo pins p. 262: day 1 stands outside the week as
// Holiday, and Wonday begins it on day 2. The page's other two worked
// examples check the cycle at both ends of the year.
func TestTheWeekRunsFromDayTwo(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		day  int
		want string
	}{
		{1, "Holiday"},
		{2, "Wonday"},
		{3, "Tuday"},
		{4, "Thirday"},
		{5, "Forday"},
		{6, "Fiday"},
		{7, "Sixday"},
		{8, "Senday"},
		{9, "Wonday"},
		{44, "Wonday"},  // Sigg Odra, p. 263
		{65, "Wonday"},  // Dagin, March 6, p. 263
		{241, "Tuday"},  // Ciencia, August 29, p. 263
		{365, "Senday"}, // the last day of the year
	} {
		if got := calendar.Weekday(tc.day); got != tc.want {
			t.Errorf("day %d is %s, want %s", tc.day, got, tc.want)
		}
	}
}

// TestADayOutsideTheYearIsRefused keeps On from minting a date the
// calendar has no room for.
func TestADayOutsideTheYearIsRefused(t *testing.T) {
	t.Parallel()

	for _, day := range []int{0, -1, 366} {
		if _, err := calendar.On(day, calendar.DefaultYear); err == nil {
			t.Errorf("day %d was accepted; the year runs 1 to 365", day)
		}
	}
}

// TestADieOutsideTheRangeIsRefused guards the four-dice read.
func TestADieOutsideTheRangeIsRefused(t *testing.T) {
	t.Parallel()

	table := load(t)

	for _, dice := range [][4]int{{0, 1, 1, 1}, {1, 7, 1, 1}, {1, 1, 0, 1}, {1, 1, 1, 9}} {
		if _, err := table.Day(dice[0], dice[1], dice[2], dice[3]); err == nil {
			t.Errorf("dice %v were accepted", dice)
		}
	}
}
