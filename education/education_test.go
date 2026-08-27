package education_test

import (
	"testing"

	"github.com/philoserf/t5chargen/education"
)

// TestPrograms verifies the chart C program rows (p. 60): count, the FR3
// implemented set, and spot-checked fields.
func TestPrograms(t *testing.T) {
	programs, err := education.Programs()
	if err != nil {
		t.Fatal(err)
	}

	if len(programs) != 17 {
		t.Errorf("got %d programs, want 17 chart C rows", len(programs))
	}

	implemented := map[string]bool{}

	for _, p := range programs {
		if p.Implemented {
			implemented[p.ID] = true
		}
	}

	// The four graduate rows are gated on a credential rather than a
	// characteristic (I-103), so a character reaches them only after a
	// degree — which is the ladder chart C prints, BA to MA to Professor,
	// with the two professional schools beside it.
	//
	// The last four are not chosen from the step C menu. OTC and NOTC are
	// volunteered for from inside the College or University hosting them
	// (p. 61, I-108); the two assigned schools are handed to the
	// character by a career (p. 59, I-91 to I-93). offeredPrograms drops
	// both prerequisite kinds, so neither reaches the menu.
	want := []string{
		"ed5", "trade_school", "apprenticeship", "college", "university", "academy",
		"masters", "professors", "medical_school", "law_school",
		"otc", "notc", "anm_school", "command_college",
	}
	if len(implemented) != len(want) {
		t.Errorf("implemented = %v, want %v", implemented, want)
	}

	for _, id := range want {
		if !implemented[id] {
			t.Errorf("program %q not implemented", id)
		}
	}

	checkProgramSpots(t)
}

// checkProgramSpots pins two chart C rows field by field.
func checkProgramSpots(t *testing.T) {
	t.Helper()

	university, err := education.ProgramByID("university")
	if err != nil {
		t.Fatal(err)
	}

	if university.Prerequisite.Value != 7 || university.Rolls != 4 ||
		university.GraduationEdu != 9 || university.GraduationDegree != "BA" {
		t.Errorf("university = %+v", university)
	}

	ed5, err := education.ProgramByID("ed5")
	if err != nil {
		t.Fatal(err)
	}

	if ed5.DurationYears != 0 || ed5.GraduationEdu != 5 || len(ed5.ApplyCheck) != 0 {
		t.Errorf("ed5 = %+v", ed5)
	}
}

// TestMajors pins the Available Skills matrix transcription (p. 60): the
// College column count and order head, dedup of the thrice-listed Grav,
// and spot rows.
func TestMajors(t *testing.T) {
	college, err := education.Majors(education.InstitutionCollege)
	if err != nil {
		t.Fatal(err)
	}

	if len(college) != 40 {
		t.Errorf("college majors = %d, want 40", len(college))
	}

	if college[0] != "Athlete" || college[1] != "Broker" {
		t.Errorf("college majors head = %v", college[:3])
	}

	for _, name := range []string{"Language", "Astrogator", "Actor", "Programmer", "Robotics", "Aquanautics"} {
		if !contains(college, name) {
			t.Errorf("college majors missing %s", name)
		}
	}

	for _, name := range []string{"Advocate", "Medic", "Admin", "Driver: Grav", "Diplomat"} {
		if contains(college, name) {
			t.Errorf("college majors wrongly include %s", name)
		}
	}

	checkServiceMajors(t)
}

// checkServiceMajors spot-checks the academy columns.
func checkServiceMajors(t *testing.T) {
	t.Helper()

	army, err := education.Majors(education.InstitutionArmy)
	if err != nil {
		t.Fatal(err)
	}

	// The three chart C Grav rows are distinct knowledges of distinct
	// parent skills, stored qualified so they cannot stack into one
	// entry (ERRATA.md I-10).
	for _, name := range []string{"Driver: Grav", "Flyer: Grav", "Seafarer: Grav"} {
		if occurrences(army, name) != 1 {
			t.Errorf("%s appears %d times in army majors, want 1", name, occurrences(army, name))
		}
	}

	navy, err := education.Majors(education.InstitutionNavy)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"Astrogator", "Fleet Tactics", "Spacecraft ACS", "Bay Weapons"} {
		if !contains(navy, name) {
			t.Errorf("navy majors missing %s", name)
		}
	}
}

// TestAllSkillNames verifies the unrestricted Apprenticeship list is the
// deduplicated full matrix.
func TestAllSkillNames(t *testing.T) {
	names, err := education.AllSkillNames()
	if err != nil {
		t.Fatal(err)
	}

	// All 121 rows are distinct names: the three Grav rows carry their
	// parent skill (ERRATA.md I-10), so nothing collapses.
	if len(names) != 121 {
		t.Errorf("got %d skill names, want 121", len(names))
	}

	for _, name := range []string{"Driver: Grav", "Flyer: Grav", "Seafarer: Grav"} {
		if occurrences(names, name) != 1 {
			t.Errorf("%s appears %d times, want 1", name, occurrences(names, name))
		}
	}
}

func contains(names []string, want string) bool {
	return occurrences(names, want) > 0
}

func occurrences(names []string, want string) int {
	count := 0

	for _, name := range names {
		if name == want {
			count++
		}
	}

	return count
}
