package chargen_test

import (
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// TestHoldsDegree pins what satisfies chart C's credential prerequisites
// (I-103), which gate the four graduate programs: "A University Masters
// Program requires a Bachelors. A Professors Program requires a Masters.
// Medical School or Law School requires an Honors Bachelors" (p. 61).
//
// The cases that matter are the composite degrees. The Service Academy's
// Graduation is "C5=8 BA Officer1", so its graduate holds a Bachelors with
// a commission printed beside it; an Honors run records "Honors BA". Both
// are Bachelors, and neither is what a naive whole-string comparison would
// call one.
func TestHoldsDegree(t *testing.T) {
	for _, tc := range []struct {
		name      string
		education []chargen.EducationRecord
		want      string
		holds     bool
	}{
		{"a plain Bachelors", graduated("BA", false), "BA", true},
		{"an Honors Bachelors is a Bachelors", graduated("BA", true), "BA", true},
		{"the Academy's BA Officer1 is a Bachelors", graduated("BA Officer1", false), "BA", true},
		{"a Masters is not a Bachelors", graduated("MA", false), "BA", false},
		{"a Bachelors is not a Masters", graduated("BA", false), "MA", false},
		{"Honors is asked for and held", graduated("BA", true), "Honors BA", true},
		{"Honors is asked for and missing", graduated("BA", false), "Honors BA", false},
		{"an Honors Academy graduate", graduated("BA Officer1", true), "Honors BA", true},
		{"no schooling at all", nil, "BA", false},
		// No degree chart C prints distinguishes a whole-word match from
		// a substring one, so this case is constructed rather than
		// reachable: it pins the contract for the next credential added,
		// where the difference between "MBA" and "BA" is the whole
		// question. Without it the comparison could be weakened to a
		// substring and no test would notice.
		{"a credential is a whole word, not a substring", graduated("MBA", false), "BA", false},
		{
			"a degree not graduated is not held",
			[]chargen.EducationRecord{{Program: "College", Degree: "BA", Graduated: false}},
			"BA", false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &chargen.Character{Education: tc.education}
			if got := chargen.HoldsDegree(c, tc.want); got != tc.holds {
				t.Errorf("holdsDegree(%q) = %v over %v, want %v", tc.want, got, tc.education, tc.holds)
			}
		})
	}
}

// graduated builds a one-record education history.
func graduated(degree string, honors bool) []chargen.EducationRecord {
	return []chargen.EducationRecord{{
		Program: "College", Degree: degree, Graduated: true, Honors: honors,
	}}
}
