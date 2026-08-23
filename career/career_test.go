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
		{"negative sanity per terms", valid + `, "sanity_per_terms": -1` + table},
		{"bad characteristic", strings.Replace(valid, `["Str"]`, `["Sta"]`, 1) + table},
		{
			"unknown cell kind",
			strings.Replace(valid, `{"kind": "skill", "name": "Admin"}`, `{"kind": "skil", "name": "Admin"}`, 1) + table,
		},
		{"nameless skill cell", strings.Replace(valid, `{"kind": "skill", "name": "Admin"}`, `{"kind": "skill"}`, 1) + table},
		{"short column", strings.Replace(valid, `{"kind": "skill", "name": "Medic"}`, ``, 1) + table},
		{"misspelled No Skill", valid + strings.Replace(table, `"Athlete"`, `"No skill"`, 1)},
		{"armed forces with a bad Branch check", valid + armedForces("branch_check", `"Sta"`) + table},
		{"armed forces with no Branch table", valid + armedForces("branches", `[]`) + table},
		{"armed forces with no Operations table", valid + armedForces("operations", `[]`) + table},
		{"armed forces with no assignments per term", valid + armedForces("operations_per_term", `0`) + table},
		{"armed forces with a nameless Branch", valid + armedForces("branches", `[{"mod": 1}]`) + table},
		{"armed forces with a nameless Operation", valid + armedForces("operations", `[{"mod": 1}]`) + table},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := career.Load(tt.name, []byte(tt.data)); err == nil {
				t.Error("malformed definition accepted")
			}
		})
	}
}

// armedForces builds an otherwise valid armed_forces block with one field
// replaced, for the loader's Branch and Operations checks.
func armedForces(key, value string) string {
	fields := []struct{ key, value string }{
		{"branch_check", `"Soc"`},
		{"branches", `[{"name": "Infantry", "mod": 1}]`},
		{"operations", `[{"name": "Combat", "mod": 2}]`},
		{"operations_per_term", `4`},
	}

	parts := make([]string, len(fields))

	for i, field := range fields {
		if field.key == key {
			field.value = value
		}

		parts[i] = `"` + field.key + `": ` + field.value
	}

	return `, "armed_forces": {` + strings.Join(parts, ", ") + `}`
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

// TestByNameCoversAvailable holds the loader map against the career list.
// A career in Available with no loader makes ByName fail for it, and
// FirstCareers fail outright — which is how Craftsman was briefly absent
// from every option list with nothing to show for it.
func TestByNameCoversAvailable(t *testing.T) {
	for _, name := range career.Available() {
		def, err := career.ByName(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		if def.Name != name {
			t.Errorf("ByName(%q) loaded %q", name, def.Name)
		}
	}
}

// m2Conflicts is the divergence set between each career's own table D and
// chart M2's reprint on p. 71, with what the disagreement is. The career
// page governs (interpretation I-71); this pins which pages disagree so a
// later transcription pass cannot quietly resolve one or introduce a
// seventh.
var m2Conflicts = map[string]string{
	"Scholar":     "M2 appends a twelfth row, Cr60,000 / TAS Fellow",
	"Entertainer": "M2 stops at twelve rows and divides Fame by 5, not 3",
	"Citizen":     "M2 has twelve rows: a new first row, and the career page's eleven shifted down",
	"Rogue":       "M2 reads +Terms where the career page reads +Total Terms",
	"Noble":       "M2 reads + Terms where the career page reads +Total Terms",
	"Functionary": "M2 appends a twelfth row, Pension x2 / Knighthood",
}

// musterOutRows is the row count each career page prints for its table D:
// ten for the Soldier and Marine, eleven for the Scholar, Citizen, Spacer
// and Functionary, twelve for six more, thirteen for the Entertainer. A
// band would let a dropped row pass; the counts are pinned instead.
var musterOutRows = map[string]int{
	"Craftsman": 12, "Scholar": 11, "Entertainer": 13, "Citizen": 11,
	"Scout": 12, "Merchant": 12, "Spacer": 11, "Soldier": 10,
	"Agent": 12, "Rogue": 12, "Noble": 12, "Marine": 10,
	"Functionary": 11,
}

// TestEveryCareerHasAMusterOutTable holds that all thirteen table Ds are
// transcribed: "Each career has a Mustering Out table" (p. 68).
func TestEveryCareerHasAMusterOutTable(t *testing.T) {
	for _, name := range career.Available() {
		def, err := career.ByName(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if def.MusterOut == nil {
			t.Errorf("%s has no muster-out table", name)

			continue
		}

		if got := len(def.MusterOut.Rows); got != musterOutRows[name] {
			t.Errorf("%s table D has %d rows, want %d", name, got, musterOutRows[name])
		}
	}
}

// TestM2DivergesExactlyWhereRecorded is the guard the M2 decision rests
// on. Chart M2 (p. 71) reprints all thirteen tables and disagrees with six
// of them; the career page governs, and this test fails if that set
// changes in either direction.
func TestM2DivergesExactlyWhereRecorded(t *testing.T) {
	diverged := map[string]bool{}

	for _, name := range career.Available() {
		def, err := career.ByName(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if def.MusterOutM2 == nil {
			continue
		}

		diverged[name] = true

		if _, known := m2Conflicts[name]; !known {
			t.Errorf("%s carries an M2 reprint but no recorded conflict", name)
		}

		if def.MusterOut == nil {
			t.Errorf("%s carries an M2 reprint but no career-page table", name)

			continue
		}

		if sameMusterOut(def.MusterOut, def.MusterOutM2) {
			t.Errorf("%s carries an M2 reprint identical to its career page; the conflict is gone", name)
		}
	}

	for name := range m2Conflicts {
		if !diverged[name] {
			t.Errorf("%s is recorded as conflicting with M2 but carries no reprint: %s",
				name, m2Conflicts[name])
		}
	}
}

// sameMusterOut reports whether two tables are the same rules, ignoring
// the citation and the box label.
func sameMusterOut(a, b *career.MusterOut) bool {
	if a.MoneyDM != b.MoneyDM || a.BenefitDM != b.BenefitDM || !samePowerDM(a, b) {
		return false
	}

	if len(a.Rows) != len(b.Rows) {
		return false
	}

	for i, row := range a.Rows {
		if !sameMusterOutRow(row, b.Rows[i]) {
			return false
		}
	}

	return true
}

// samePowerDM compares the optional third column's DM line, which chart 11
// alone prints.
func samePowerDM(a, b *career.MusterOut) bool {
	if (a.PowerDM == nil) != (b.PowerDM == nil) {
		return false
	}

	return a.PowerDM == nil || *a.PowerDM == *b.PowerDM
}

// sameMusterOutRow compares one row across the two editions.
func sameMusterOutRow(a, b career.MusterOutRow) bool {
	if a.Money != b.Money || a.Benefit != b.Benefit {
		return false
	}

	if (a.Power == nil) != (b.Power == nil) {
		return false
	}

	return a.Power == nil || *a.Power == *b.Power
}

// TestTheSevenTablesThatAgreeCarryNoReprint holds the other half: a career
// absent from the divergence set must match M2, which it does by carrying
// no reprint at all.
func TestTheSevenTablesThatAgreeCarryNoReprint(t *testing.T) {
	agree := 0

	for _, name := range career.Available() {
		def, err := career.ByName(name)
		if err != nil {
			t.Fatal(err)
		}

		if _, conflicts := m2Conflicts[name]; conflicts {
			continue
		}

		agree++

		if def.MusterOutM2 != nil {
			t.Errorf("%s agrees with M2 but carries a reprint", name)
		}
	}

	if want := len(career.Available()) - len(m2Conflicts); agree != want {
		t.Errorf("%d careers agree with M2, want %d", agree, want)
	}
}
