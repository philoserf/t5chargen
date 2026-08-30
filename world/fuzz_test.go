package world_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/ehex"
	"github.com/philoserf/t5chargen/world"
)

// FuzzValidateUWP holds ValidateUWP to two promises against arbitrary
// input: it never panics, and it never accepts a string that is not the
// shape p. 22 describes.
//
// The second is the one worth fuzzing. A validator that rejects too much
// is caught by the ordinary tests, which feed it real UWPs; a validator
// that accepts too much is caught by nothing, because no example test
// supplies the string nobody thought of. So acceptance is re-derived here
// from the definition rather than compared with a fixture.
//
// The seed corpus runs in the ordinary gate; the fuzzing engine runs only
// under -fuzz, so this costs the gate a few microseconds.
func FuzzValidateUWP(f *testing.F) {
	for _, seed := range []string{
		"A788899-C", // chart B's own example
		"X000000-0",
		"E9A7654-3",
		"",
		"A788899",    // too short
		"A788899-CC", // too long
		"F788899-C",  // starport outside A-E, X
		"A788899C-",  // hyphen misplaced
		"A78I899-C",  // I and O are not eHex digits (p. 22)
		"a788899-c",  // lower case
		"A78889 -C",
		"\x00\x00\x00\x00\x00\x00\x00-\x00",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, uwp string) {
		if err := world.ValidateUWP(uwp); err != nil {
			return
		}

		// Accepted. Everything below re-states the format from p. 22
		// rather than trusting the function that just approved it.
		if len(uwp) != 9 {
			t.Fatalf("accepted %q, which is %d characters and not 9", uwp, len(uwp))
		}

		if !strings.ContainsRune("ABCDEX", rune(uwp[0])) {
			t.Fatalf("accepted %q, whose starport %q is not one of A B C D E X", uwp, uwp[0])
		}

		if uwp[7] != '-' {
			t.Fatalf("accepted %q, which has %q where the hyphen belongs", uwp, uwp[7])
		}

		for _, i := range []int{1, 2, 3, 4, 5, 6, 8} {
			if _, err := ehex.Decode(uwp[i]); err != nil {
				t.Fatalf("accepted %q, whose position %d holds %q: %v", uwp, i, uwp[i], err)
			}
		}
	})
}

// FuzzEhexRoundTrip holds the eHex digits to their definition (p. 22):
// every digit Decode accepts, Encode returns, and the value is unchanged.
func FuzzEhexRoundTrip(f *testing.F) {
	for _, seed := range []byte{'0', '9', 'A', 'Z', 'I', 'O', ' ', 0} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, digit byte) {
		value, err := ehex.Decode(digit)
		if err != nil {
			return
		}

		back, err := ehex.Encode(value)
		if err != nil {
			t.Fatalf("Decode(%q) = %d, which Encode refuses: %v", digit, value, err)
		}

		if back != digit {
			t.Fatalf("Decode(%q) = %d, which Encodes back to %q", digit, value, back)
		}
	})
}
