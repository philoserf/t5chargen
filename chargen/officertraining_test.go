package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// volunteerDecider is the default policy except it takes the named
// officer-training row and steers education toward College, which is what
// hosts it.
//
// The rows exist for exactly the reason POLICY.md gives for declining
// them: a commission removes the career choice, so the auto policy will
// not take one and no golden record reaches these paths. This decider is
// how they are reached instead — the same answer the postgraduate note
// gives, applied to a rule the policy cannot afford to exercise.
type volunteerDecider struct{ want, service string }

func (d volunteerDecider) Choose(c chargen.Choice) (int, error) {
	switch c.ID { //nolint:exhaustive // Only the choice points this decider steers; the rest fall through to the policy.
	case chargen.ChooseOfficerTraining:
		if i := slices.Index(c.Options, d.want); i >= 0 {
			return i, nil
		}
	case chargen.ChooseEducation:
		if i := slices.Index(c.Options, "College"); i >= 0 {
			return i, nil
		}
	case chargen.ChooseService:
		// NOTC confers "Navy Officer1 or Marine Officer1" (p. 61) and
		// the policy takes first-listed, so Marine is reachable only by
		// asking for it.
		if d.service != "" {
			if i := slices.Index(c.Options, d.service); i >= 0 {
				return i, nil
			}
		}
	}

	return autoPolicy(c)
}

func (volunteerDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestOfficerTrainingCommissions verifies chart C's two volunteer rows
// confer the commission their Graduation column prints: OTC "Army
// Officer1", NOTC "Navy Officer1 or Marine Officer1" (p. 60; p. 61).
//
// A commission obliges the character to the service (I-99), which is why
// the assertion is on the career he opens with as well as on the degree.
func TestOfficerTrainingCommissions(t *testing.T) {
	for _, tt := range []struct {
		name     string
		row      string
		service  string
		services []string
	}{
		{"OTC", "OTC", "", []string{"Army"}},
		{"NOTC into the Navy", "NOTC", "Navy", []string{"Navy"}},
		{"NOTC into the Marines", "NOTC", "Marine", []string{"Marine"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			seen := false

			for seed := range uint64(400) {
				c := generate(t, chargen.Options{Seed: seed, Decider: volunteerDecider{want: tt.row, service: tt.service}})

				record, ok := educationRecord(c, tt.row)
				if !ok {
					continue
				}

				seen = true

				if !slices.Contains(tt.services, record.Service) {
					t.Fatalf("seed %d: %s commissioned into %q, want one of %v",
						seed, tt.row, record.Service, tt.services)
				}

				if !record.Graduated {
					continue // a failed roll confers nothing, which is the rule
				}

				if want := record.Service + " Officer1"; record.Degree != want {
					t.Errorf("seed %d: %s graduated with %q, want %q", seed, tt.row, record.Degree, want)
				}

				if len(c.Careers) > 0 && c.Careers[0].Career != careerForService(record.Service) {
					t.Errorf("seed %d: a %s commission opened with %q, not %q",
						seed, record.Service, c.Careers[0].Career, careerForService(record.Service))
				}
			}

			if !seen {
				t.Skipf("no seed in 400 reached %s", tt.row)
			}
		})
	}
}

// TestOfficerTrainingIsTakenOnce verifies a character takes at most one of
// the two rows, and is offered them at most once per hosting institution
// (I-108).
//
// Declining is not attempting, so a character who declines at College and
// later attends University meets the offer again — that is the rule, not a
// leak. What he cannot do is hold two commissions.
func TestOfficerTrainingIsTakenOnce(t *testing.T) {
	for seed := range uint64(200) {
		c := generate(t, chargen.Options{Seed: seed, Decider: volunteerDecider{want: "OTC"}})

		taken, hosts, offers := 0, 0, 0

		for _, record := range c.Education {
			switch record.Program {
			case "OTC", "NOTC":
				taken++
			case "College", "University":
				hosts++
			}
		}

		for _, event := range c.Events {
			if event.Choice != nil && strings.Contains(event.Choice.Prompt, "Volunteer for officer training") {
				offers++
			}
		}

		if taken > 1 {
			t.Fatalf("seed %d: took %d officer-training rows, want at most 1", seed, taken)
		}

		if offers > hosts {
			t.Fatalf("seed %d: offered officer training %d times across %d hosting institutions",
				seed, offers, hosts)
		}
	}
}

// educationRecord returns the character's record for a named chart C row.
func educationRecord(c chargen.Character, program string) (chargen.EducationRecord, bool) {
	for _, record := range c.Education {
		if record.Program == program {
			return record, true
		}
	}

	return chargen.EducationRecord{}, false
}

// careerForService is the career a commission into a service obliges.
func careerForService(service string) string {
	switch service {
	case "Army":
		return "Soldier"
	case "Navy":
		return "Spacer"
	case "Marine":
		return "Marine"
	}

	return ""
}
