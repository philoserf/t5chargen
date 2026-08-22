package skill_test

import (
	"errors"
	"testing"

	"github.com/philoserf/t5chargen/skill"
)

// TestListLoads pins that the embedded Master Skill List parses and passes
// its load-time validation, including the printed "64 SKILLS" count
// (p. 132).
func TestListLoads(t *testing.T) {
	if err := skill.Err(); err != nil {
		t.Fatalf("loading the master skill list: %v", err)
	}

	if got := len(skill.Skills()); got != skill.SkillCount {
		t.Errorf("skills = %d, chart MS prints %d", got, skill.SkillCount)
	}
}

// TestSkillGroups pins the five chart MS skill headings and their counts,
// which sum to the printed 64.
func TestSkillGroups(t *testing.T) {
	want := map[string]int{
		"Skills":          35,
		"Starship Skills": 7,
		"Trades":          10,
		"Arts":            6,
		"Soldier Skills":  6,
	}

	got := map[string]int{}

	for _, name := range skill.Skills() {
		entry, ok := skill.Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) missing", name)
		}

		got[entry.Group]++
	}

	for group, count := range want {
		if got[group] != count {
			t.Errorf("group %q = %d entries, want %d", group, got[group], count)
		}
	}

	if len(got) != len(want) {
		t.Errorf("groups = %v, want the %d printed headings", got, len(want))
	}
}

// TestLookupKinds spot-checks one entry of each kind against the printed
// page.
func TestLookupKinds(t *testing.T) {
	tests := []struct {
		name   string
		kind   skill.Kind
		parent string
	}{
		{name: "Admin", kind: skill.KindSkill},
		{name: "Navigator", kind: skill.KindSkill},
		{name: "Battle Dress", kind: skill.KindKnowledge, parent: "Fighter"},
		{name: "Planetology", kind: skill.KindKnowledge, parent: "The Sciences"},
		{name: "Memorize", kind: skill.KindTalent},
		{name: "Carouse", kind: skill.KindPersonal},
		{name: "Luck", kind: skill.KindIntuition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := skill.Lookup(tt.name)
			if !ok {
				t.Fatalf("Lookup(%q) not found", tt.name)
			}

			if entry.Kind != tt.kind {
				t.Errorf("kind = %q, want %q", entry.Kind, tt.kind)
			}

			if entry.Parent != tt.parent {
				t.Errorf("parent = %q, want %q", entry.Parent, tt.parent)
			}
		})
	}
}

// TestDefaultSkills pins the chart MS Default Skills list, including
// Turrets — printed as a Default although it is a Gunner knowledge rather
// than one of the 64 skills.
func TestDefaultSkills(t *testing.T) {
	for _, name := range []string{"Actor", "Comms", "Vacc Suit", "Turrets"} {
		entry, ok := skill.Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) not found", name)
		}

		if !entry.Default {
			t.Errorf("%q is not flagged Default; chart MS lists it", name)
		}
	}

	if entry, _ := skill.Lookup("Admin"); entry.Default {
		t.Error("Admin is flagged Default; chart MS does not list it")
	}
}

// TestQualifiedKnowledges pins that a knowledge printed under several
// parent skills is stored qualified, so the three Grav knowledges cannot
// stack into one entry (ERRATA.md I-10).
func TestQualifiedKnowledges(t *testing.T) {
	for _, name := range []string{"Driver: Grav", "Flyer: Grav", "Seafarer: Grav"} {
		if _, ok := skill.Lookup(name); !ok {
			t.Errorf("Lookup(%q) not found", name)
		}
	}

	// A knowledge printed under exactly one parent stays unqualified.
	for _, name := range []string{"Wheeled", "Turrets", "Ship", "Small Craft"} {
		if _, ok := skill.Lookup(name); !ok {
			t.Errorf("Lookup(%q) not found; single-parent knowledges stay unqualified", name)
		}
	}
}

// TestResolveAmbiguous pins the two chart cells the engine resolves by
// choice (ERRATA.md I-10, I-11) and the Master Skill List order of their
// alternatives.
func TestResolveAmbiguous(t *testing.T) {
	tests := []struct {
		label string
		want  []string
	}{
		{label: "Grav", want: []string{"Driver: Grav", "Flyer: Grav", "Seafarer: Grav"}},
		{label: "Spacecraft", want: []string{"Spacecraft ACS", "Spacecraft BCS"}},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			_, err := skill.Resolve(tt.label)

			var ambiguous *skill.AmbiguousError
			if !errors.As(err, &ambiguous) {
				t.Fatalf("Resolve(%q) error = %v, want *AmbiguousError", tt.label, err)
			}

			if len(ambiguous.Options) != len(tt.want) {
				t.Fatalf("options = %v, want %v", ambiguous.Options, tt.want)
			}

			for i, want := range tt.want {
				if ambiguous.Options[i] != want {
					t.Errorf("option %d = %q, want %q", i, ambiguous.Options[i], want)
				}
			}

			if got := skill.Options(tt.label); len(got) != len(tt.want) {
				t.Errorf("Options(%q) = %v, want %v", tt.label, got, tt.want)
			}
		})
	}
}

// TestResolveCanonical pins that canonical names resolve to themselves and
// unknown names are rejected — the load-time guarantee the chart
// transcriptions rely on.
func TestResolveCanonical(t *testing.T) {
	got, err := skill.Resolve("Forward Observer")
	if err != nil {
		t.Fatalf("Resolve(Forward Observer) = %v", err)
	}

	if got != "Forward Observer" {
		t.Errorf("resolved to %q, want itself", got)
	}

	// "Navigation" is chart 04 table E's spelling of the Master Skill
	// List's Navigator (ERRATA.md I-9); it must not be a valid name.
	for _, name := range []string{"Navigation", "Fwd Obs", "Naval Arch", ""} {
		if _, err := skill.Resolve(name); !errors.Is(err, skill.ErrUnknownSkill) {
			t.Errorf("Resolve(%q) error = %v, want ErrUnknownSkill", name, err)
		}
	}
}

// TestValidate pins that Validate accepts canonical and ambiguous names
// and rejects unknown ones.
func TestValidate(t *testing.T) {
	for _, name := range []string{"Admin", "Grav", "Spacecraft", "Driver: Grav"} {
		if err := skill.Validate(name); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", name, err)
		}
	}

	if err := skill.Validate("Navigation"); err == nil {
		t.Error("Validate(Navigation) = nil, want an error")
	}
}
