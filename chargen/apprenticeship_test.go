package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/education"
)

// apprenticeSeedSearch is how wide the sweeps look. Apprenticeship has no
// prerequisite and no admission check, so a decider that asks for it gets
// it — what varies is whether the Pass/Fail roll succeeds.
const apprenticeSeedSearch = 200

// takesApprenticeship selects the Apprenticeship at step C, which the
// auto policy never does: it declines every row below a Bachelors
// (POLICY.md), so both rules below are unreachable in auto mode and no
// golden record exercises either.
type takesApprenticeship struct{}

func (takesApprenticeship) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseEducation {
		if i := slices.Index(c.Options, "Apprenticeship"); i >= 0 {
			return i, nil
		}
	}

	return autoPolicy(c)
}

func (takesApprenticeship) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestApprenticeshipChecksTraAtHalfEdu verifies interpretation I-6.
//
// Chart C gives the Apprenticeship a Pass/Fail check of Tra, and a human
// has no Tra: "Training and Education can be substituted for each other
// at half value" (p. 55), rounded in the roller's favour by the p. 19
// practice. So the throw a human makes is against (Edu+1)/2, and the
// transcript has to say so — a target nobody can trace to a
// characteristic the character holds is the kind of number that gets
// mistaken for a bug.
func TestApprenticeshipChecksTraAtHalfEdu(t *testing.T) {
	checks := 0

	for seed := range uint64(apprenticeSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Decider: takesApprenticeship{}})

		if !attendedApprenticeship(c) {
			continue
		}

		want := (c.Characteristics.Edu + 1) / 2

		for _, event := range c.Events {
			if event.Throw == nil || event.Throw.Target == nil {
				continue
			}

			if !containsAll(event.Throw.Cite, "Apprenticeship", "Tra") {
				continue
			}

			checks++

			if got := *event.Throw.Target; got != want {
				t.Errorf("seed %d: Tra checked against %d with Edu %d, want %d",
					seed, got, c.Characteristics.Edu, want)
			}
		}
	}

	if checks == 0 {
		t.Fatalf("no seed under %d throws the Apprenticeship's Tra check; the sweep is asserting nothing",
			apprenticeSeedSearch)
	}
}

// TestApprenticeshipOffersEverySkill verifies interpretation I-7: chart
// C's "Skill+4" names no list, so the selection is the whole Available
// Skills matrix rather than a column of it.
//
// The Apprenticeship is the one row that states no source, and reading it
// as unrestricted is what the chart's silence says. A narrowed list would
// be an invention, and an invisible one — the record would show a legal
// skill either way.
func TestApprenticeshipOffersEverySkill(t *testing.T) {
	all, err := education.AllSkillNames()
	if err != nil {
		t.Fatalf("education: %v", err)
	}

	if len(all) == 0 {
		t.Fatal("the Available Skills matrix is empty; the test is asserting nothing")
	}

	offers := 0

	for seed := range uint64(apprenticeSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Decider: takesApprenticeship{}})

		for _, event := range c.Events {
			if event.Choice == nil || event.Choice.Prompt != "Select the Apprenticeship skill" {
				continue
			}

			offers++

			if !slices.Equal(event.Choice.Options, all) {
				t.Fatalf("seed %d: the Apprenticeship offered %d skills, want the whole matrix of %d",
					seed, len(event.Choice.Options), len(all))
			}
		}
	}

	if offers == 0 {
		t.Fatalf("no seed under %d reaches the Apprenticeship's selection; the sweep is asserting nothing",
			apprenticeSeedSearch)
	}
}

// attendedApprenticeship reports whether the record shows the row.
func attendedApprenticeship(c chargen.Character) bool {
	return slices.ContainsFunc(c.Education, func(r chargen.EducationRecord) bool {
		return r.Program == "Apprenticeship"
	})
}

// containsAll reports whether every fragment appears in the text.
func containsAll(text string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			return false
		}
	}

	return true
}
