package career_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/skill"
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
	if len(d.ControllingCharacteristics) != len(wantCCs) {
		t.Fatalf("controlling characteristics = %v", d.ControllingCharacteristics)
	}

	for i, want := range wantCCs {
		if d.ControllingCharacteristics[i] != want {
			t.Errorf("controlling characteristic %d = %q, want %q", i, d.ControllingCharacteristics[i], want)
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

// TestLoadValidation verifies the loader rejects malformed-but-parseable
// career data instead of letting it panic or corrupt deep in the engine.
func TestLoadValidation(t *testing.T) {
	valid := `{
		"name": "X", "cite": "test", "continue_target": 10, "skills_per_term": 4,
		"controlling_characteristics": ["Str"],
		"skill_columns": [{"name": "C", "entries": [
			{"kind": "skill", "name": "Admin"}, {"kind": "skill", "name": "Broker"},
			{"kind": "skill", "name": "Comms"}, {"kind": "skill", "name": "Driver"},
			{"kind": "skill", "name": "Leader"}, {"kind": "skill", "name": "Medic"}]}]`

	table := `, "job_table": [` + jobGroup() + `,` + jobGroup() + `,` + jobGroup() + `]}`

	if _, err := career.Load("valid.json", []byte(valid+table)); err != nil {
		t.Fatalf("valid definition rejected: %v", err)
	}

	// An empty controlling-characteristic series is legal: chart 03's
	// "Risk & Reward Talent" rotates none.
	noCC := strings.Replace(valid, `["Str"]`, `[]`, 1) + table
	if _, err := career.Load("no-cc.json", []byte(noCC)); err != nil {
		t.Errorf("definition with no controlling characteristics rejected: %v", err)
	}

	tests := []struct {
		name string
		data string
	}{
		{"zero continue target", strings.Replace(valid, `"continue_target": 10`, `"continue_target": 0`, 1) + table},
		{"unmissable continue target", strings.Replace(valid, `"continue_target": 10`, `"continue_target": 12`, 1) + table},
		{"zero skills per term", strings.Replace(valid, `"skills_per_term": 4`, `"skills_per_term": 0`, 1) + table},
		{"bad characteristic", strings.Replace(valid, `["Str"]`, `["Sta"]`, 1) + table},
		{
			"unknown cell kind",
			strings.Replace(valid, `{"kind": "skill", "name": "Admin"}`, `{"kind": "skil", "name": "Admin"}`, 1) + table,
		},
		{"nameless skill cell", strings.Replace(valid, `{"kind": "skill", "name": "Admin"}`, `{"kind": "skill"}`, 1) + table},
		{"short column", strings.Replace(valid, `{"kind": "skill", "name": "Medic"}`, ``, 1) + table},
		{"misspelled No Skill", valid + strings.Replace(table, `"Athlete"`, `"No skill"`, 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := career.Load(tt.name, []byte(tt.data)); err == nil {
				t.Error("malformed definition accepted")
			}
		})
	}
}

// jobGroup builds one syntactically valid table E group of 6x6 cells.
// Cells name real Master Skill List entries, which the loader now
// validates.
func jobGroup() string {
	names := skill.Skills()
	rows := make([]string, 6)

	for b := range 6 {
		cells := make([]string, 6)
		for c := range 6 {
			cells[c] = fmt.Sprintf("%q", names[(b*6+c)%len(names)])
		}

		rows[b] = "[" + strings.Join(cells, ",") + "]"
	}

	return "[" + strings.Join(rows, ",") + "]"
}

// TestCitizenDerivedLists verifies the precomputed column-name and
// hobby-choice lists.
func TestCitizenDerivedLists(t *testing.T) {
	d, err := career.Citizen()
	if err != nil {
		t.Fatal(err)
	}

	names := d.SkillColumnNames()
	if len(names) != 7 || names[0] != "Personal" || names[3] != "General" {
		t.Errorf("SkillColumnNames() = %v", names)
	}

	hobbies := d.HobbyChoices()
	if len(hobbies) == 0 || hobbies[0] != "ACV" {
		t.Fatalf("HobbyChoices() head = %v", hobbies[:min(3, len(hobbies))])
	}

	for _, hobby := range hobbies {
		if hobby == career.NoSkillCell {
			t.Error("HobbyChoices() includes the No Skill cell")
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
