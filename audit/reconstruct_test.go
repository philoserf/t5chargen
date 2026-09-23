package audit_test

import (
	"path/filepath"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// TestTheEventLogAccountsForTheRecord holds CLAUDE.md's "Event log first"
// rule: every rule effect emits an event. Replay cannot catch a breach —
// an effect applied without an event is deterministic, reproduces
// exactly, and is simply missing from the transcript a rules expert reads.
// This can: it folds each record's consequence events back into the
// fields they produced and compares them with what the record says.
//
// Three fields fold cleanly, and only those are held: the six
// characteristics (set, changed, floored, reset and aged, each carrying
// the new value), the skills (each receipt carries the new level), and the
// age (18, plus every span of years elapsed — advanceYears is the only
// ager). Credits, terms, errata and benefits are left out rather than
// special-cased: a gate that needs exceptions to go green is worse than a
// narrower one that does not.
//
// It reads records and never generates them, so the compatibility corpus
// — written by released binaries, and never regenerated — is held too.
func TestTheEventLogAccountsForTheRecord(t *testing.T) {
	records, err := filepath.Glob(filepath.Join("..", "chargen", "testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	records = append(records, corpusRecords(t)...)
	if len(records) == 0 {
		t.Fatal("no records to fold; the gate would pass while proving nothing")
	}

	for _, path := range records {
		t.Run(filepath.Base(path), func(t *testing.T) {
			record := corpusRecord(t, path) // decodes any record, not only the corpus's
			folded := fold(record.Events)

			for _, c := range []struct {
				name       string
				got, whole int
			}{
				{"Str", folded.characteristics["Str"], record.Characteristics.Str},
				{"Dex", folded.characteristics["Dex"], record.Characteristics.Dex},
				{"End", folded.characteristics["End"], record.Characteristics.End},
				{"Int", folded.characteristics["Int"], record.Characteristics.Int},
				{"Edu", folded.characteristics["Edu"], record.Characteristics.Edu},
				{"Soc", folded.characteristics["Soc"], record.Characteristics.Soc},
			} {
				if c.got != c.whole {
					t.Errorf("the record says %s %d, the event log accounts for %d: "+
						"a rule effect was applied without emitting a consequence", c.name, c.whole, c.got)
				}
			}

			for _, skill := range record.Skills {
				if got := folded.skills[skill.Name]; got != skill.Level {
					t.Errorf("the record says %s-%d, the event log accounts for %d: "+
						"a rule effect was applied without emitting a consequence", skill.Name, skill.Level, got)
				}

				delete(folded.skills, skill.Name)
			}

			for name, level := range folded.skills {
				t.Errorf("the event log awards %s-%d, and the record has no such skill", name, level)
			}

			if folded.age != record.Age {
				t.Errorf("the record says age %d, the event log accounts for %d: "+
					"years passed without a years_elapsed consequence", record.Age, folded.age)
			}
		})
	}
}

// folded is what a record's consequence events say its derived fields are.
type folded struct {
	characteristics map[string]int
	skills          map[string]int
	age             int
}

// fold replays the consequence stream into the fields it produced.
func fold(events []chargen.Event) folded {
	f := folded{characteristics: map[string]int{}, skills: map[string]int{}, age: chargen.StartAge}

	for _, event := range events {
		c := event.Consequence
		if c == nil {
			continue
		}

		switch c.Kind { //nolint:exhaustive // only the kinds that carry a derived field's new value
		case chargen.ConsequenceCharacteristicSet,
			chargen.ConsequenceCharacteristicChange,
			chargen.ConsequenceCharacteristicFloored,
			chargen.ConsequenceCharacteristicReset,
			chargen.ConsequenceAgingEffect:
			if c.Characteristic != "" {
				f.characteristics[c.Characteristic] = c.Value
			}
		case chargen.ConsequenceSkillAwarded:
			f.skills[c.Skill] = c.Value
		case chargen.ConsequenceYearsElapsed:
			f.age += c.Value
		}
	}

	return f
}
