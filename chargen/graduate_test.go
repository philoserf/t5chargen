package chargen_test

import (
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// scholarPath climbs chart C's Higher Education block as far as it can,
// taking Honors on the way so the professional schools become reachable:
// "Medical School or Law School requires an Honors Bachelors" (p. 61).
//
// It reaches for the highest rung on offer rather than following chart
// order. Each programme may be attempted once (I-100), so a decider that
// took them in chart order would spend the lower rows first and never
// present the case these tests are about — the mistake #72 caught.
type scholarPath struct {
	playerKind

	wants  []string
	honors bool

	// watch names the programme whose Major and Minor prompts are
	// collected; at is the one the character is in now.
	watch  string
	at     string
	majors []chargen.Choice
}

//nolint:exhaustive // Deliberately partitioned: the rest defer to the auto policy.
func (d *scholarPath) Choose(c chargen.Choice) (int, error) {
	switch c.ID {
	case chargen.ChooseEducation, chargen.ChooseLaterEducation:
		return d.rung(c), nil
	case chargen.ChooseHonors:
		if d.honors {
			return 0, nil // "Attempt Honors"
		}

		return 1, nil // "Decline" — the auto policy would attempt, and
		// Honors prefixes whatever degree follows
	case chargen.ChooseMajor, chargen.ChooseMinor:
		// Only what the programme under test asked for. A character
		// reaches a graduate row through College or University, which do
		// select a Major and a Minor, so collecting every such prompt in
		// the run would report their choices against this one.
		if d.at == d.watch {
			d.majors = append(d.majors, c)
		}
	}

	return autoPolicy(c)
}

// rung takes the first wanted programme the character actually qualifies
// for, and otherwise serves.
//
// Qualification is the point. p. 61 says the credential prerequisites "can
// be waived", so a decider that simply asked for the highest row would
// reach Professors on a waiver without ever holding a Masters — which is a
// legal lifepath but not the ladder these tests are about, and it silently
// made an earlier draft of the Masters case unreachable.
func (d *scholarPath) rung(c chargen.Choice) int {
	for _, want := range d.wants {
		for i, option := range c.Options {
			if option == want && i < len(c.Scores) && c.Scores[i] == 1 {
				d.at = want

				return i
			}
		}
	}

	d.at = ""

	return 0
}

// climbTo finds a seed whose character graduates the named programme,
// returning the character and the choices he was put through.
func climbTo(t *testing.T, program string, wants []string, honors bool) (chargen.Character, *scholarPath, bool) {
	t.Helper()

	for seed := range uint64(academySeedSearch) {
		path := &scholarPath{wants: wants, honors: honors, watch: program}

		c, err := chargen.Generate(chargen.Options{Seed: seed, CurrentYear: 1105, Decider: path})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		for _, record := range c.Education {
			if record.Program == program && record.Graduated {
				return c, path, true
			}
		}
	}

	return chargen.Character{}, nil, false
}

// TestTheAcademicLadder walks chart C's Higher Education block end to end:
// a Bachelors opens the Masters, a Masters opens the Professors programme,
// and each Graduation lands the character on the Edu its credential stands
// for (p. 62).
func TestTheAcademicLadder(t *testing.T) {
	for _, tc := range []struct {
		program, degree string
		edu             int
	}{
		{"Masters", "MA", 9},
		{"Professors", "Professor", 12},
	} {
		t.Run(tc.program, func(t *testing.T) {
			// No Honors: it prefixes whatever degree follows, and this
			// case is about the credential the programme confers.
			c, _, ok := climbTo(t, tc.program,
				[]string{"Professors", "Masters", "University", "College"}, false)
			if !ok {
				t.Fatalf("no seed under %d graduates %s; widen the search", academySeedSearch, tc.program)
			}

			var degree string

			for _, record := range c.Education {
				if record.Program == tc.program {
					degree = record.Degree
				}
			}

			if degree != tc.degree {
				t.Errorf("%s awarded the degree %q, want %q", tc.program, degree, tc.degree)
			}

			if c.Characteristics.Edu < tc.edu {
				t.Errorf("%s left the character at Edu %d, below the %d its Graduation names",
					tc.program, c.Characteristics.Edu, tc.edu)
			}
		})
	}
}

// TestTheProfessionalSchools covers the two rows whose Provides names the
// skill they teach, and the credential each confers.
func TestTheProfessionalSchools(t *testing.T) {
	for _, tc := range []struct {
		program, degree, skill string
		level, edu             int
	}{
		{"Medical School", "Doctor", "Medic", 4, 10},
		{"Law School", "Attorney", "Advocate", 2, 10},
	} {
		t.Run(tc.program, func(t *testing.T) {
			// Honors, because "Medical School or Law School requires an
			// Honors Bachelors" (p. 61).
			c, path, ok := climbTo(t, tc.program,
				[]string{tc.program, "University", "College"}, true)
			if !ok {
				t.Fatalf("no seed under %d graduates %s; widen the search", academySeedSearch, tc.program)
			}

			var degree string

			for _, record := range c.Education {
				if record.Program == tc.program {
					degree = record.Degree
				}
			}

			if degree != tc.degree {
				t.Errorf("%s awarded the degree %q, want %q", tc.program, degree, tc.degree)
			}

			if got := skillLevel(c, tc.skill); got < tc.level {
				t.Errorf("%s left %s at %d, want at least the %d a full run of passes reaches",
					tc.program, tc.skill, got, tc.level)
			}

			// The Provides is the whole award: these two rows select no
			// Major or Minor, so no such prompt may name them.
			for _, choice := range path.majors {
				t.Errorf("%s put a %q prompt, and its Provides is the whole award", tc.program, choice.Prompt)
			}
		})
	}
}

// skillLevel reads one skill off the record, or 0 where it is not held.
func skillLevel(c chargen.Character, name string) int {
	for _, held := range c.Skills {
		if held.Name == name {
			return held.Level
		}
	}

	return 0
}

// TestGraduationEdu pins chart C's Graduation column against the
// parenthetical above it, "(If Edu already at this level, award Edu+1)"
// (p. 60), read as p. 62 frames those values: positions on a scale, where
// Edu 9 "can function at the equivalent of a Masters" (I-105).
//
// The last case is the one that changed. A character above the value used
// to take the +1, once per programme, which is what made a second degree
// worth collecting for its own sake.
func TestGraduationEdu(t *testing.T) {
	for _, tc := range []struct {
		name            string
		edu, graduation int
		delta           int
		awarded         bool
	}{
		{"below the value is raised to it", 5, 8, 3, true},
		{"far below", 2, 12, 10, true},
		{"exactly at the value takes the consolation", 8, 8, 1, true},
		{"above the value gains nothing", 9, 8, 0, false},
		{"far above", 14, 5, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			delta, awarded := chargen.GraduationEdu(tc.edu, tc.graduation)
			if delta != tc.delta || awarded != tc.awarded {
				t.Errorf("graduationEdu(%d, %d) = (%d, %v), want (%d, %v)",
					tc.edu, tc.graduation, delta, awarded, tc.delta, tc.awarded)
			}
		})
	}
}
