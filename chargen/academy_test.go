package chargen_test

import (
	"strings"
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

// entryRankFor is entryRank for a career that is not the first: the rank
// set the first time the named career sets one. A character who changes
// service has two, and the second is the one under test.
func entryRankFor(t *testing.T, c chargen.Character, careerName string) string {
	t.Helper()

	for _, event := range c.Events {
		if event.Kind != chargen.EventConsequence || event.Consequence == nil {
			continue
		}

		if event.Consequence.Kind == chargen.ConsequenceRankSet && event.Consequence.Career == careerName {
			return rankIDIn(t, careerName, event.Consequence.Skill)
		}
	}

	t.Fatalf("the record sets no rank in %s", careerName)

	return ""
}

// rankIDIn maps a logged rank title back to its ladder id in a named
// career.
func rankIDIn(t *testing.T, careerName, title string) string {
	t.Helper()

	for _, rank := range careerDef(t, careerName).Ranks {
		if rank.Title == title {
			return rank.ID
		}
	}

	t.Fatalf("no rank titled %q in %s", title, careerName)

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

// commissionedRun finds a seed that reaches the path's commission and
// begins the career it obliges. It is academyRun generalised: the forced
// career is the one the commission already owes, so it narrows the search
// rather than contradicting it.
func commissionedRun(t *testing.T, path commissionedPath) (chargen.Character, bool) {
	t.Helper()

	for seed := range uint64(academySeedSearch) {
		c, err := chargen.Generate(chargen.Options{
			Seed: seed, Career: path.career, Decider: path.decider,
		})
		if err != nil || !path.commissioned(c) || len(c.Careers) == 0 || !c.Careers[0].Began {
			continue
		}

		return c, true
	}

	return chargen.Character{}, false
}

// TestCommissionedGraduateEntersAsAnOfficer verifies the Graduation
// column of every chart C row that commissions: the Academy's "C5=8 BA
// Officer1" (p. 60) and OTC's and NOTC's from p. 61. The holder joins the
// service he trained for at its first officer rank rather than at the
// enlisted rank p. 65 gives every other recruit (interpretation I-94).
//
// This is the other half of the commission, and a different function
// answers it: career selection asks which service the commission owes,
// entry rank asks whether this character is that service's officer.
func TestCommissionedGraduateEntersAsAnOfficer(t *testing.T) {
	for _, tc := range commissionedPaths() {
		t.Run(tc.name, func(t *testing.T) {
			// Not a Skip: a skip passes silently the day no seed in
			// range reaches the case, leaving the test asserting nothing
			// about the rule it names.
			c, ok := commissionedRun(t, tc)
			if !ok {
				t.Fatalf("no seed under %d commissions via %s and begins %s; widen the search",
					academySeedSearch, tc.name, tc.career)
			}

			if want := firstOfficerRank(t, tc.career); entryRank(t, c) != want {
				t.Errorf("a %s commission entered %s at %q, want the officer rank %q",
					tc.name, tc.career, entryRank(t, c), want)
			}
		})
	}
}

// TestAcademyOfficerIsServiceSpecific verifies the linkage does not cross
// services. An Academy trains for the force it names, so its graduate
// joining a different one enters as any other recruit does.
//
// This used to be a first career and can no longer be one: a Navy Academy
// graduate owes the Navy a term (I-99), so the only way he reaches the
// Army is by serving what he owes and then changing careers — which p. 62
// names as the thing he may do next, "at the end of that term". The claim
// is unchanged; the only route to it moved.
func TestAcademyOfficerIsServiceSpecific(t *testing.T) {
	c, ok := crossServiceRun(t)
	if !ok {
		t.Fatalf("no seed under %d graduates the Navy Academy, serves the Navy and then joins the Army; widen the search",
			academySeedSearch)
	}

	got := entryRankFor(t, c, "Soldier")

	if want := careerDef(t, "Soldier").Ranks[0].ID; got != want {
		t.Errorf("a Navy Academy graduate entered the Army at %q, want the enlisted rank %q", got, want)
	}
}

// crossServiceRun finds a seed whose character graduates the Navy Academy,
// serves the term he owes as a Spacer, then changes to Soldier.
func crossServiceRun(t *testing.T) (chargen.Character, bool) {
	t.Helper()

	for seed := range uint64(academySeedSearch) {
		c, err := chargen.Generate(chargen.Options{
			Seed: seed, Decider: &changesToArmy{academyPath{service: "Navy"}},
		})
		if err != nil || !graduatedAcademy(c, "Navy") || len(c.Careers) < 2 {
			continue
		}

		if c.Careers[0].Career != "Spacer" || c.Careers[len(c.Careers)-1].Career != "Soldier" {
			continue
		}

		if !c.Careers[len(c.Careers)-1].Began {
			continue
		}

		return c, true
	}

	return chargen.Character{}, false
}

// changesToArmy serves the owed Navy term, then leaves for the Army at the
// first opportunity.
type changesToArmy struct{ academyPath }

//nolint:exhaustive // Deliberately partitioned: the rest defer to the embedded path.
func (d *changesToArmy) Choose(c chargen.Choice) (int, error) {
	switch c.ID {
	case chargen.ChooseCareerChange:
		return 1, nil // "Change careers"
	case chargen.ChooseCareer:
		if i, err := pick(c, "Soldier"); err == nil {
			return i, nil
		}
	}

	return d.academyPath.Choose(c)
}

func (*changesToArmy) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// commissionedPath is one route to an Officer1 commission and the career
// it obliges. There are two: the Service Academy (p. 62) and chart C's
// two volunteer rows (p. 61), which confer the same Officer1 token and so
// owe the same term.
type commissionedPath struct {
	name    string
	career  string
	decider chargen.Decider
	// commissioned reports whether the run actually reached the
	// commission, which is the premise every case depends on.
	commissioned func(chargen.Character) bool
}

// commissionedPaths crosses both routes with all three services. The
// obligation is to the force that trained him, so a commission that
// pointed anywhere would satisfy a weaker claim than the rule makes.
func commissionedPaths() []commissionedPath {
	paths := make([]commissionedPath, 0, 2*len(academyServices))

	for _, tc := range academyServices {
		paths = append(paths, commissionedPath{
			name:    tc.service + " Academy",
			career:  tc.career,
			decider: academyPath{service: tc.service},
			commissioned: func(c chargen.Character) bool {
				return graduatedAcademy(c, tc.service)
			},
		})

		// OTC commissions into the Army, NOTC into the Navy or the
		// Marines (p. 61), so the row follows from the service.
		row, want := "NOTC", tc.service

		if tc.service == "Army" {
			// OTC offers no service choice, so asking for one would
			// steer a choice point this path never reaches.
			row, want = "OTC", ""
		}

		paths = append(paths, commissionedPath{
			name:    row + " into the " + tc.service,
			career:  tc.career,
			decider: volunteerDecider{want: row, service: want},
			commissioned: func(c chargen.Character) bool {
				return commissionedBy(c, row, tc.service)
			},
		})
	}

	return paths
}

// commissionedBy reports whether a chart C row conferred a commission
// into the named service. The degree is what carries Officer1, and NOTC's
// Graduation column prints two of them, so the record's own Service is
// what settles which one this run took.
func commissionedBy(c chargen.Character, program, service string) bool {
	for _, record := range c.Education {
		if record.Program == program && record.Graduated &&
			record.Service == service && strings.Contains(record.Degree, "Officer1") {
			return true
		}
	}

	return false
}

// TestCommissionedGraduateOwesHisService is the p. 62 obligation:
// "The character is required to serve one term in the service."
//
// Measured at the record's first career, and swept across every route to
// a commission rather than the Academy alone: p. 61 gives OTC and NOTC
// the same obligation in the same words, and `commissionedCareer` reads
// all of them through one degree token.
func TestCommissionedGraduateOwesHisService(t *testing.T) {
	for _, tc := range commissionedPaths() {
		t.Run(tc.name, func(t *testing.T) {
			found := 0

			for seed := range uint64(academySeedSearch) {
				c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: tc.decider})
				if err != nil || !tc.commissioned(c) || len(c.Careers) == 0 {
					continue
				}

				found++

				if got := c.Careers[0].Career; got != tc.career {
					t.Errorf("seed %d: a %s commission opened with %q, want %q",
						seed, tc.name, got, tc.career)
				}
			}

			if found == 0 {
				t.Fatalf("no seed under %d commissions via %s; the sweep is asserting nothing",
					academySeedSearch, tc.name)
			}
		})
	}
}

// TestACommissionAndAForcedCareerCannotBothBeHonoured verifies the two are
// refused together rather than one quietly winning. A character who owes
// the Navy a term and was told on the command line to begin as a Soldier
// is a contradiction, and no silent repair is the house rule.
func TestACommissionAndAForcedCareerCannotBothBeHonoured(t *testing.T) {
	tried := 0

	for seed := range uint64(academySeedSearch) {
		probe, err := chargen.Generate(chargen.Options{
			Seed: seed, Decider: academyPath{service: "Navy"},
		})
		if err != nil || !graduatedAcademy(probe, "Navy") {
			continue
		}

		tried++

		_, err = chargen.Generate(chargen.Options{
			Seed: seed, Career: "Soldier", Decider: academyPath{service: "Navy"},
		})
		if err == nil {
			t.Errorf("seed %d: --career Soldier and a Navy commission both succeeded", seed)

			continue
		}

		if !strings.Contains(err.Error(), "Spacer") {
			t.Errorf("seed %d: the refusal %q does not name the career he owes", seed, err)
		}
	}

	if tried == 0 {
		t.Fatalf("no seed under %d graduates the Navy Academy; the sweep is asserting nothing", academySeedSearch)
	}
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

// TestGraduatesAlwaysEnterTheirService is interpretation I-101: acceptance
// to the Academy is acceptance to the first term, so a graduate makes no
// To Begin throw and cannot fail to take up the commission he owes.
//
// Before it, sixty-five of a hundred and sixty-three Army graduates never
// began the service they were required to serve — and because I-99 had
// already made that service their only option, they served nothing at all.
func TestGraduatesAlwaysEnterTheirService(t *testing.T) {
	for _, tc := range academyServices {
		t.Run(tc.service, func(t *testing.T) {
			graduates := 0

			for seed := range uint64(academySeedSearch) {
				c, err := chargen.Generate(chargen.Options{
					Seed: seed, Decider: academyPath{service: tc.service},
				})
				if err != nil || !graduatedAcademy(c, tc.service) {
					continue
				}

				graduates++

				if len(c.Careers) == 0 || !c.Careers[0].Began {
					t.Errorf("seed %d: a %s Academy graduate did not begin %s", seed, tc.service, tc.career)
				}
			}

			if graduates == 0 {
				t.Fatalf("no seed under %d graduates the %s Academy; the sweep is asserting nothing",
					academySeedSearch, tc.service)
			}
		})
	}
}

// TestAFailedCadetStillRollsToBegin holds the other half of I-101. The
// exemption belongs to the commission, not to having turned up: a cadet who
// failed out owes no term and enters a career the way anyone else does.
//
// Measured by the throw rather than the outcome, because a cadet who
// happens to pass To Begin looks identical at the record.
func TestAFailedCadetStillRollsToBegin(t *testing.T) {
	cadets := 0

	for seed := range uint64(academySeedSearch) {
		c, err := chargen.Generate(chargen.Options{
			Seed: seed, Career: "Spacer", Decider: academyPath{service: "Navy"},
		})
		if err != nil || graduatedAcademy(c, "Navy") || !attendedAcademy(c) {
			continue
		}

		cadets++

		if !hasStep(c, "Spacer: To Begin") {
			t.Errorf("seed %d: a cadet who failed out entered the Navy without a To Begin throw", seed)
		}
	}

	if cadets == 0 {
		t.Fatalf("no seed under %d attends the Navy Academy without graduating; the sweep is asserting nothing",
			academySeedSearch)
	}
}

// attendedAcademy reports whether the character went to the Academy at all,
// graduating or not.
func attendedAcademy(c chargen.Character) bool {
	for _, record := range c.Education {
		if record.Program == serviceAcademyName {
			return true
		}
	}

	return false
}

// hasStep reports whether the record holds a step of the given name.
func hasStep(c chargen.Character, name string) bool {
	for _, event := range c.Events {
		if event.Kind == chargen.EventStep && event.Step != nil && event.Step.Name == name {
			return true
		}
	}

	return false
}
