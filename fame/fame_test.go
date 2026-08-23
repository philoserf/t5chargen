package fame_test

import (
	"testing"

	"github.com/philoserf/t5chargen/fame"
)

func load(t *testing.T) *fame.Table {
	t.Helper()

	table, err := fame.Load()
	if err != nil {
		t.Fatal(err)
	}

	return table
}

// TestStack pins "A character's Fame is the sum of all Fame points
// received to 20; beyond 20, only the highest Fame applies" (p. 91), read
// as a cap rather than a cliff (interpretation I-63).
func TestStack(t *testing.T) {
	table := load(t)

	for _, tt := range []struct {
		name   string
		points []int
		want   int
	}{
		{name: "nothing", points: nil, want: 0},
		// The chart's own example: "Rogue with one Failed Scheme (and no
		// other applicable factors) has Fame = 1 x 3 = 3".
		{name: "the chart's example", points: []int{3}, want: 3},
		{name: "a plain sum", points: []int{6, 8}, want: 14},
		{name: "exactly the limit", points: []int{12, 8}, want: 20},
		// The discriminating case. Summing gives 24; the rival reading —
		// that a total past 20 collapses to the largest single source —
		// gives 12, which is less than the character had before earning
		// the second accomplishment.
		{name: "a sum past the limit caps", points: []int{12, 12}, want: 20},
		// A single source worth more than 20 carries past it, which is
		// the clause the limit makes room for.
		{name: "one source past the limit", points: []int{36}, want: 36},
		{name: "one source past the limit, plus others", points: []int{36, 2}, want: 36},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := table.Stack(tt.points); got != tt.want {
				t.Errorf("Stack(%v) = %d, want %d", tt.points, got, tt.want)
			}
		})
	}
}

// TestStackIsMonotonic holds the property that decided I-63: earning one
// more accomplishment can never lower a character's Fame.
func TestStackIsMonotonic(t *testing.T) {
	table := load(t)

	for base := range 30 {
		for extra := range 30 {
			before := table.Stack([]int{base})
			after := table.Stack([]int{base, extra})

			if after < before {
				t.Fatalf("Fame fell from %d to %d when a %d-point accomplishment was added",
					before, after, extra)
			}
		}
	}
}

// TestDescriptors pins the printed levels chart F names, including the
// one the page itself calls out: "A world famous Entertainer has Fame-10".
func TestDescriptors(t *testing.T) {
	table := load(t)

	for level, want := range map[int]string{
		0: "Unknown", 1: "Parent", 10: "World", 20: "Sector", 36: "All Reality",
	} {
		if got := table.Descriptor(level); got != want {
			t.Errorf("Fame-%d is %q, want %q", level, got, want)
		}
	}

	// Nothing lies beyond All Reality.
	if got := table.Descriptor(100); got != "All Reality" {
		t.Errorf("Fame-100 is %q, want the last descriptor", got)
	}
}

// TestMedalPoints pins the Armed Forces multipliers, including Exemplary
// Service at x0 — a decoration the chart prices at nothing.
func TestMedalPoints(t *testing.T) {
	table := load(t)

	for code, want := range map[string]int{
		"XS": 0, "WB": 1, "MCUF": 1, "MCG": 2, "SEH": 3, "*SEH*": 4,
	} {
		got, ok := table.MedalPoints(code)
		if !ok {
			t.Errorf("%s is not priced", code)

			continue
		}

		if got != want {
			t.Errorf("%s is worth %d, want %d", code, got, want)
		}
	}

	if _, ok := table.MedalPoints("nonesuch"); ok {
		t.Error("an unknown code is priced")
	}
}
