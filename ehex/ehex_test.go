package ehex_test

import (
	"testing"

	"github.com/philoserf/t5chargen/ehex"
)

// TestRoundTrip encodes every representable value 0-33 and decodes it back
// (Book 1 p. 22: values 0 through 33, I and O omitted).
func TestRoundTrip(t *testing.T) {
	seen := map[byte]bool{}

	for value := 0; value <= ehex.Max; value++ {
		digit, err := ehex.Encode(value)
		if err != nil {
			t.Fatalf("Encode(%d): %v", value, err)
		}

		if seen[digit] {
			t.Fatalf("digit %q produced twice", digit)
		}

		seen[digit] = true

		got, err := ehex.Decode(digit)
		if err != nil {
			t.Fatalf("Decode(%q): %v", digit, err)
		}

		if got != value {
			t.Errorf("Decode(Encode(%d)) = %d", value, got)
		}
	}
}

func TestEncodeKnownDigits(t *testing.T) {
	tests := []struct {
		value int
		want  byte
	}{
		{0, '0'},
		{9, '9'},
		{10, 'A'},
		{15, 'F'},
		{16, 'G'},
		{17, 'H'},
		{18, 'J'}, // I omitted (p. 22)
		{22, 'N'},
		{23, 'P'}, // O omitted (p. 22)
		{33, 'Z'},
	}

	for _, tt := range tests {
		got, err := ehex.Encode(tt.value)
		if err != nil {
			t.Fatalf("Encode(%d): %v", tt.value, err)
		}

		if got != tt.want {
			t.Errorf("Encode(%d) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestEncodeRejectsOutOfRange(t *testing.T) {
	for _, value := range []int{-1, 34, 100} {
		if _, err := ehex.Encode(value); err == nil {
			t.Errorf("Encode(%d) succeeded, want error", value)
		}
	}
}

func TestDecodeRejectsInvalidDigits(t *testing.T) {
	for _, digit := range []byte{'I', 'O', 'a', ' ', '-'} {
		if _, err := ehex.Decode(digit); err == nil {
			t.Errorf("Decode(%q) succeeded, want error", digit)
		}
	}
}
