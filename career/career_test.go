package career_test

import (
	"testing"

	"github.com/philoserf/t5chargen/career"
)

// TestCitizenDefinition verifies the embedded Citizen data parses and
// matches the chart 04 (p. 78) fixed values.
func TestCitizenDefinition(t *testing.T) {
	d, err := career.Citizen()
	if err != nil {
		t.Fatal(err)
	}

	if d.Name != "Citizen" || d.ContinueTarget != 10 || d.SkillsPerTerm != 4 {
		t.Errorf("definition header = %+v", d)
	}

	wantCCs := []string{"Str", "Dex", "End", "Int"}
	if len(d.CitizenLifeCharacteristics) != len(wantCCs) {
		t.Fatalf("citizen life characteristics = %v", d.CitizenLifeCharacteristics)
	}

	for i, want := range wantCCs {
		if d.CitizenLifeCharacteristics[i] != want {
			t.Errorf("citizen life characteristic %d = %q, want %q", i, d.CitizenLifeCharacteristics[i], want)
		}
	}
}

// TestCitizenSkillColumns verifies table C shape: seven named columns of
// six entries each, in chart order (p. 78).
func TestCitizenSkillColumns(t *testing.T) {
	d, err := career.Citizen()
	if err != nil {
		t.Fatal(err)
	}

	wantNames := []string{"Personal", "Academic", "Travel", "General", "Business", "Vocation", "Avocation"}
	if len(d.SkillColumns) != len(wantNames) {
		t.Fatalf("got %d columns, want %d", len(d.SkillColumns), len(wantNames))
	}

	for i, column := range d.SkillColumns {
		if column.Name != wantNames[i] {
			t.Errorf("column %d = %q, want %q", i, column.Name, wantNames[i])
		}

		if len(column.Entries) != 6 {
			t.Errorf("column %q has %d entries, want 6", column.Name, len(column.Entries))
		}
	}

	general := d.SkillColumns[3]
	wantGeneral := []string{"Admin", "Broker", "Computer", "Animals", "Bureaucrat", "Trader"}

	for i, want := range wantGeneral {
		entry := general.Entries[i]
		if entry.Kind != career.EntrySkill || entry.Name != want {
			t.Errorf("General row %d = %+v, want skill %q", i+1, entry, want)
		}
	}
}

// TestCitizenJobTable verifies table E lookups, including the "No Skill"
// cell at A1 B3 C5 (p. 78).
func TestCitizenJobTable(t *testing.T) {
	d, err := career.Citizen()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		a, b, c int
		want    career.Entry
	}{
		{1, 1, 1, career.Entry{Kind: career.EntrySkill, Name: "ACV"}},
		{1, 3, 5, career.Entry{Kind: career.EntryNone}},
		{2, 6, 6, career.Entry{Kind: career.EntrySkill, Name: "Electronics"}},
		{3, 6, 6, career.Entry{Kind: career.EntrySkill, Name: "Spacecraft"}},
		{3, 4, 3, career.Entry{Kind: career.EntrySkill, Name: "Fighter"}},
	}

	for _, tt := range tests {
		if got := d.JobEntry(tt.a, tt.b, tt.c); got != tt.want {
			t.Errorf("JobEntry(%d, %d, %d) = %+v, want %+v", tt.a, tt.b, tt.c, got, tt.want)
		}
	}

	// Every cell holds a non-empty name.
	for a := 1; a <= 3; a++ {
		for b := 1; b <= 6; b++ {
			for c := 1; c <= 6; c++ {
				entry := d.JobEntry(a, b, c)
				if entry.Kind == career.EntrySkill && entry.Name == "" {
					t.Errorf("empty cell at A%d B%d C%d", a, b, c)
				}
			}
		}
	}
}
