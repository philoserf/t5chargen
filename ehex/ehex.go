// Package ehex implements the Traveller Expanded Hex Code.
//
// "The Traveller Expanded Hex Code (eHex) substitutes single letters for
// numbers above 9 and allows the creation of a string of character with
// values from 0 through 33. ... the letters A-F correspond to the values
// 10-15 ... the letters G-Z correspond to the numbers 16-33. Omit I and O.
// To avoid potential for confusion, with the digits one (1) and zero (0),
// the alphabetic letters I and O are omitted." (Book 1 Print Edition 5.1,
// p. 22)
package ehex

import (
	"errors"
	"fmt"
	"strings"
)

// digits maps each value 0-33 to its eHex digit; I and O are omitted (p. 22).
const digits = "0123456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// Max is the largest value representable as a single eHex digit (Z, p. 22).
//
// Derived from the alphabet rather than written beside it: the two were
// independent transcriptions of one fact, and Encode's bound has to be
// the last index of the string it then indexes.
const Max = len(digits) - 1

// ErrRange reports a value outside 0 through Max.
var ErrRange = errors.New("value outside eHex range")

// ErrDigit reports a byte that is not an eHex digit (including the omitted
// I and O, p. 22).
var ErrDigit = errors.New("invalid eHex digit")

// Encode returns the single eHex digit for value.
func Encode(value int) (byte, error) {
	if value < 0 || value > Max {
		return 0, fmt.Errorf("ehex: %w: %d (want 0-%d)", ErrRange, value, Max)
	}

	return digits[value], nil
}

// Decode returns the value of a single eHex digit. I and O are rejected,
// as p. 22 omits them.
func Decode(digit byte) (int, error) {
	if value := strings.IndexByte(digits, digit); value >= 0 {
		return value, nil
	}

	return 0, fmt.Errorf("ehex: %w: %q", ErrDigit, digit)
}
