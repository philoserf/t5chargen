package chargen_test

import (
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// academyPath sends a character through the Service Academy for one
// service and then into one career. The auto policy never selects the
// Academy (POLICY.md), so nothing but a scripted decider reaches the
// Officer1 graduation this test is about.
type academyPath struct {
	service string
}

//nolint:exhaustive // Deliberately partitioned: the rest defer to the auto policy.
func (d academyPath) Choose(c chargen.Choice) (int, error) {
	switch c.ID {
	case chargen.ChooseEducation:
		return pick(c, "Service Academy")
	case chargen.ChooseService:
		return pick(c, d.service)
	default:
		return autoPolicy(c)
	}
}

func (academyPath) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// pick selects a named option, falling back to the auto policy when the
// engine did not offer it.
func pick(c chargen.Choice, want string) (int, error) {
	for i, option := range c.Options {
		if option == want {
			return i, nil
		}
	}

	return autoPolicy(c)
}

// The services and the careers whose own Branch tables title themselves
// with them: "NAVAL BRANCH" (chart 07), "ARMY BRANCH" (chart 08), "MARINE
// BRANCH" (chart 12).
var academyServices = []struct {
	service string
	career  string
}{
	{service: "Army", career: "Soldier"},
	{service: "Navy", career: "Spacer"},
	{service: "Marine", career: "Marine"},
}

// graduatedAcademy reports whether the run reached the Officer1
// graduation, which is the premise every case here depends on.
func graduatedAcademy(c chargen.Character, service string) bool {
	for _, record := range c.Education {
		if record.Program == "Service Academy" && record.Graduated && record.Service == service {
			return true
		}
	}

	return false
}

// careerDef loads one military career's transcribed definition.
func careerDef(t *testing.T, name string) *career.Definition {
	t.Helper()

	load := map[string]func() (*career.Definition, error){
		"Soldier": career.Soldier, "Spacer": career.Spacer, "Marine": career.Marine,
	}[name]

	def, err := load()
	if err != nil {
		t.Fatal(err)
	}

	return def
}

// firstOfficerRank reports the rank a career's ladder opens its officer
// side with.
func firstOfficerRank(t *testing.T, name string) string {
	t.Helper()

	for _, rank := range careerDef(t, name).Ranks {
		if rank.Class == "officer" {
			return rank.ID
		}
	}

	t.Fatalf("%s has no officer rank", name)

	return ""
}

// entryRank reports the rank the character joined at: the title of the
// first rank set in the record.
//
// Read from the log rather than from CareerRecord.Rank, which holds the
// rank at the career's end. Two of the three services passed against that
// field by luck — their runs happened not to promote — while the Spacer
// had reached O3 by muster out and failed a test that was asking the wrong
// question.
func entryRank(t *testing.T, c chargen.Character) string {
	t.Helper()

	for _, event := range c.Events {
		if event.Kind == chargen.EventConsequence && event.Consequence.Kind == chargen.ConsequenceRankSet {
			return rankIDForTitle(t, c, event.Consequence.Skill)
		}
	}

	t.Fatal("the record sets no rank")

	return ""
}

// rankIDForTitle maps a logged rank title back to its ladder id.
func rankIDForTitle(t *testing.T, c chargen.Character, title string) string {
	t.Helper()

	for _, rank := range careerDef(t, c.Careers[0].Career).Ranks {
		if rank.Title == title {
			return rank.ID
		}
	}

	t.Fatalf("no rank titled %q", title)

	return ""
}

// TestAcademyGraduateEntersAsAnOfficer verifies chart C's Graduation
// column for the Service Academy: "C5=8 BA Officer1". A graduate joins the
// service he trained for at its first officer rank rather than at the
// enlisted rank p. 65 gives every other recruit (interpretation I-94).
func TestAcademyGraduateEntersAsAnOfficer(t *testing.T) {
	for _, tc := range academyServices {
		t.Run(tc.service, func(t *testing.T) {
			// Not a Skip: a skip passes silently the day no seed in
			// range reaches the case, leaving the test asserting nothing
			// about the rule it names.
			c, ok := academyRun(t, tc.service, tc.career)
			if !ok {
				t.Fatalf("no seed under %d graduates the %s Academy and begins %s; widen the search",
					academySeedSearch, tc.service, tc.career)
			}

			if want := firstOfficerRank(t, tc.career); entryRank(t, c) != want {
				t.Errorf("an Academy graduate entered %s at %q, want the officer rank %q",
					tc.career, entryRank(t, c), want)
			}
		})
	}
}

// TestAcademyOfficerIsServiceSpecific verifies the linkage does not cross
// services. An Academy trains for the force it names, so its graduate
// joining a different one enters as any other recruit does.
func TestAcademyOfficerIsServiceSpecific(t *testing.T) {
	c, ok := academyRun(t, "Navy", "Soldier")
	if !ok {
		t.Fatalf("no seed under %d graduates the Navy Academy and begins Soldier; widen the search",
			academySeedSearch)
	}

	if want := careerDef(t, "Soldier").Ranks[0].ID; entryRank(t, c) != want {
		t.Errorf("a Navy Academy graduate entered the Army at %q, want the enlisted rank %q",
			entryRank(t, c), want)
	}
}

// academyRun finds a seed that graduates the named Academy and begins the
// named career, returning the character and whether one was found.
func academyRun(t *testing.T, service, careerName string) (chargen.Character, bool) {
	t.Helper()

	for seed := range uint64(academySeedSearch) {
		c, err := chargen.Generate(chargen.Options{
			Seed: seed, Career: careerName, Decider: academyPath{service: service},
		})
		if err != nil || !graduatedAcademy(c, service) || len(c.Careers) == 0 || !c.Careers[0].Began {
			continue
		}

		return c, true
	}

	return chargen.Character{}, false
}

// academySeedSearch bounds the search for a qualifying seed. The Academy
// needs Edu 6+, an Admission pass and four Pass/Fail successes, so most
// seeds do not reach a graduation.
const academySeedSearch = 300

// academyHound takes the Service Academy at every choice point that offers
// it, which is the shape of the defect this file guards against: nothing
// counted attendances, so a player who kept choosing it kept being
// admitted. Twenty-three times, on seed 1, to Edu-F at age 110.
type academyHound struct{ offers []chargen.Choice }

func (d *academyHound) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseLaterEducation {
		d.offers = append(d.offers, c)
	}

	for i, option := range c.Options {
		if option == serviceAcademyName {
			return i, nil
		}
	}

	return autoPolicy(c)
}

func (*academyHound) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

const serviceAcademyName = "Service Academy"

// TestServiceAcademyIsNeverOfferedMidCareer verifies the Academy is absent
// from every Later Education offer.
//
// It reads the offers rather than the outcome, because an option a
// character happens not to take is still an option he was shown — and the
// options list is recorded in the choice event, so an offer that should
// not exist is a divergence waiting for the next engine version.
func TestServiceAcademyIsNeverOfferedMidCareer(t *testing.T) {
	offers := 0

	for seed := uint64(1); seed <= 40; seed++ {
		hound := &academyHound{}

		if _, err := chargen.Generate(chargen.Options{
			Seed: seed, CurrentYear: 1105, Decider: hound,
		}); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		for _, offer := range hound.offers {
			offers++

			for _, option := range offer.Options {
				if option == serviceAcademyName {
					t.Errorf("seed %d: Later Education offered %q: %v",
						seed, serviceAcademyName, offer.Options)
				}
			}
		}
	}

	// A sweep that produced no offers would pass while asserting nothing.
	if offers < 20 {
		t.Fatalf("only %d Later Education offers across the sweep; it is not reaching the choice point", offers)
	}
}

// TestServiceAcademyIsAttendedAtMostOnce is the same claim measured at the
// record: however hard a player reaches for it, a character has one
// Service Academy in his history at most.
//
// Attendance, not graduation. A failed applicant has spent his one chance
// — "A failure disallows admission and consumes one year" (p. 59) — and
// step C is over.
func TestServiceAcademyIsAttendedAtMostOnce(t *testing.T) {
	reached := 0

	for seed := uint64(1); seed <= 40; seed++ {
		character, err := chargen.Generate(chargen.Options{
			Seed: seed, CurrentYear: 1105, Decider: &academyHound{},
		})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		attended := 0

		for _, record := range character.Education {
			if record.Program == serviceAcademyName {
				attended++
			}
		}

		if attended > 1 {
			t.Errorf("seed %d: attended the Service Academy %d times", seed, attended)
		}

		reached += attended
	}

	if reached == 0 {
		t.Fatal("no character reached the Service Academy; the sweep is asserting nothing")
	}
}
