package chargen

import (
	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/ehex"
)

// Characteristics are the six standard human characteristics (docs/PRD.md
// FR1). "Humans have the six standard characteristics: Strength, Dexterity,
// Endurance, Intelligence, Education, and Social Standing." (Book 1 p. 123)
//
// Values are stored numeric; the UPP hex string is derived (docs/PRD.md,
// JSON conventions).
type Characteristics struct {
	Str int `json:"str"`
	Dex int `json:"dex"`
	End int `json:"end"`
	Int int `json:"int"`
	Edu int `json:"edu"`
	Soc int `json:"soc"`
}

// UPP renders the six-character Universal Personality Profile: "The
// Characteristics for each Character are shown in the convenient six-digit
// UPP Universal Personality Profile" (p. 48), in the order Str Dex End Int
// Edu Soc ("the UPP Human format SDEIES", p. 22), each digit in eHex.
//
// Every characteristic is representable: characteristicAdd floors at zero
// and awardCharacteristicAndLog caps at CharacteristicMax, both inside the
// eHex alphabet. The clamp below is a backstop for an engine fault, and it
// clamps rather than substituting "?" — ehex.Decode rejects "?", so a
// record carrying one could not be read back, which is the round-trip the
// ehex package's own test asserts.
func (c Characteristics) UPP() string {
	upp := make([]byte, 0, 6)

	for _, value := range []int{c.Str, c.Dex, c.End, c.Int, c.Edu, c.Soc} {
		digit, err := ehex.Encode(min(max(value, 0), ehex.Max))
		if err != nil {
			digit = '0'
		}

		upp = append(upp, digit)
	}

	return string(upp)
}

// characteristicValue returns the named characteristic's value; ok is
// false for a name outside the six standard abbreviations (p. 48: "A
// Characteristic can be abbreviated with its first three letters").
// Standalone functions rather than methods so Characteristics keeps a
// uniform value-receiver method set for its read-only API.
func characteristicValue(c *Characteristics, name string) (int, bool) {
	field := characteristicField(c, name)
	if field == nil {
		return 0, false
	}

	return *field, true
}

// characteristicAdd applies a delta to the named characteristic, reporting
// the new value and how much of the delta the floor refused. Callers
// validate the name first (characteristicValue); an unknown name changes
// nothing and reports 0.
//
// The floor is zero. Chart A says "If one Characteristic is reduced to 0,
// it is reset to 1" (p. 89) and the book is silent on overshooting it —
// interpretation I-107 reads the floor as covering both, so a single
// effect large enough to drive a characteristic past zero stops there.
// The value the record holds must be one the UPP can express, and a
// negative is not (ehex is a closed 34-symbol alphabet, p. 22).
//
// lost is what the floor refused, so the caller can record the clamp
// rather than leaving it invisible: CLAUDE.md's rule for a derived value
// outside the rules' range is that it clamps and emits a consequence
// saying so.
func characteristicAdd(c *Characteristics, name string, delta int) (int, int) {
	field := characteristicField(c, name)
	if field == nil {
		return 0, 0
	}

	*field += delta

	if *field < 0 {
		lost := -*field
		*field = 0

		return 0, lost
	}

	return *field, 0
}

// awardCharacteristicAndLog applies a benefit delta subject to the
// characteristic maximum, logging the change or the lost benefit:
// "Characteristics for Humans cannot exceed 15. If a benefit elevates a
// characteristic above 15, that benefit is lost" (p. 68). Callers
// validate the name first (characteristicValue).
func awardCharacteristicAndLog(character *Character, log *Log, name string, delta, cause int) {
	value, _ := characteristicValue(&character.Characteristics, name)
	if value+delta > CharacteristicMax {
		log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceBenefitLost, Characteristic: name})

		return
	}

	value, lost := characteristicAdd(&character.Characteristics, name, delta)
	log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceCharacteristicChange,
		Characteristic: name, Delta: delta, Value: value,
	})

	logClamp(log, name, lost, cause)
}

// logClamp records a characteristic the zero floor caught, so the clamp
// appears in the transcript rather than only in the difference between a
// delta and a value.
func logClamp(log *Log, name string, lost, cause int) {
	if lost == 0 {
		return
	}

	log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceCharacteristicFloored,
		Characteristic: name, Delta: -lost, Value: 0,
	})
}

// characteristicField maps a standard abbreviation to its field.
func characteristicField(c *Characteristics, name string) *int {
	switch name {
	case "Str":
		return &c.Str
	case "Dex":
		return &c.Dex
	case "End":
		return &c.End
	case "Int":
		return &c.Int
	case "Edu":
		return &c.Edu
	case "Soc":
		return &c.Soc
	}

	return nil
}

// RollCharacteristics rolls the six standard human characteristics, emitting
// a throw and a consequence event for each.
//
// "Assume the character is Human with standard characteristics generated by
// 2D each. ... Roll two dice six times and record the results in the order
// they are rolled: Strength, Dexterity, Endurance, Intelligence, Education,
// and Social Standing. Defer rolling for Psi and Sanity until later; knowing
// these values are not necessary at this point." (chart A, p. 56)
//
// The roll order above is stream order and part of the replay contract.
// Psi and Sanity are never rolled: chart A defers them and v1 excludes
// psionics (docs/PRD.md non-goals); rolling them would consume stream faces
// and shift every subsequent throw.
func RollCharacteristics(roller *dice.Roller, log *Log) Characteristics {
	const cite = "Book 1 p. 56 chart A"

	var c Characteristics

	for _, characteristic := range []struct {
		name string
		dst  *int
	}{
		{"Str", &c.Str},
		{"Dex", &c.Dex},
		{"End", &c.End},
		{"Int", &c.Int},
		{"Edu", &c.Edu},
		{"Soc", &c.Soc},
	} {
		roll := roller.Roll(2)
		*characteristic.dst = roll.Total

		cause := log.Roll(roll, cite)
		log.Consequence(ConsequenceEvent{
			Cause:          cause,
			Kind:           ConsequenceCharacteristicSet,
			Characteristic: characteristic.name,
			Value:          roll.Total,
		})
	}

	return c
}
