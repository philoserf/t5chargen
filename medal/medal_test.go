package medal_test

import (
	"errors"
	"testing"

	"github.com/philoserf/t5chargen/medal"
)

// TestForPinsTheTable pins the p. 70 rows, including the officer
// modifier: "Rew= Successful unmodified Reward Roll. If Officer, increase
// +1".
func TestForPinsTheTable(t *testing.T) {
	tests := []struct {
		reward  int
		officer bool
		code    string
		mod     int
	}{
		{reward: 2, code: "XS", mod: 1},
		{reward: 8, code: "XS", mod: 1},
		{reward: 9, code: "MCUF", mod: 2},
		{reward: 11, code: "MCG", mod: 3},
		{reward: 12, code: "SEH", mod: 4},
		// The worked example: a raw 3 with the officer modifier reads
		// line 4 and awards an XS (p. 65).
		{reward: 3, officer: true, code: "XS", mod: 1},
		// An officer's natural 12 reaches the last row.
		{reward: 12, officer: true, code: "*SEH*", mod: 5},
	}

	for _, tt := range tests {
		got, err := medal.For(tt.reward, tt.officer)
		if err != nil {
			t.Errorf("For(%d, %v) = %v", tt.reward, tt.officer, err)

			continue
		}

		if got.Code != tt.code || got.Mod != tt.mod {
			t.Errorf("For(%d, %v) = %s +%d, want %s +%d",
				tt.reward, tt.officer, got.Code, got.Mod, tt.code, tt.mod)
		}
	}
}

// TestForOffTable verifies a lookup outside the printed rows is an error
// rather than a silent zero.
func TestForOffTable(t *testing.T) {
	if _, err := medal.For(1, false); !errors.Is(err, medal.ErrOffTable) {
		t.Errorf("For(1) error = %v, want ErrOffTable", err)
	}
}
