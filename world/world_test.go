package world_test

import (
	"errors"
	"testing"

	"github.com/philoserf/t5chargen/world"
)

// TestValidateUWP verifies structural validation: "Invalid or partial
// UWPs are rejected with an error, never silently repaired" (docs/PRD.md
// FR2).
func TestValidateUWP(t *testing.T) {
	valid := []string{"A788899-C", "X000000-0", "E55769C-5", "B000453-E"}
	for _, uwp := range valid {
		if err := world.ValidateUWP(uwp); err != nil {
			t.Errorf("ValidateUWP(%q): %v", uwp, err)
		}
	}

	invalid := []string{
		"",           // empty
		"A788899",    // partial: no hyphen or tech level
		"A788899-",   // partial: no tech level
		"A788899-CX", // too long
		"Z788899-C",  // bad starport
		"A7888991C",  // missing hyphen
		"AI88899-C",  // omitted eHex letter I
		"AO88899-C",  // omitted eHex letter O
		"A78 899-C",  // space digit
		"a788899-C",  // lowercase starport
	}
	for _, uwp := range invalid {
		if err := world.ValidateUWP(uwp); !errors.Is(err, world.ErrInvalidUWP) {
			t.Errorf("ValidateUWP(%q) = %v, want ErrInvalidUWP", uwp, err)
		}
	}
}

// TestValidateHomeworld verifies whole-homeworld validation: bad UWPs,
// unknown TCs, and repeated TCs are rejected ("one specified skill for
// each Trade Classification", p. 58; docs/PRD.md FR2).
func TestValidateHomeworld(t *testing.T) {
	good := world.Homeworld{UWP: "A788899-C", TradeClassifications: []string{"Pa", "Ri"}}
	if err := good.Validate(); err != nil {
		t.Errorf("Validate() = %v", err)
	}

	tests := []struct {
		name string
		h    world.Homeworld
		want error
	}{
		{"bad UWP", world.Homeworld{UWP: "A78899"}, world.ErrInvalidUWP},
		{"unknown TC", world.Homeworld{UWP: "A788899-C", TradeClassifications: []string{"Qq"}}, world.ErrUnknownTC},
		{"duplicate TC", world.Homeworld{UWP: "A788899-C", TradeClassifications: []string{"Pa", "Pa"}}, world.ErrDuplicateTC},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.h.Validate(); !errors.Is(err, tt.want) {
				t.Errorf("Validate() = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestGrantFor verifies chart B rows (p. 56), including the multi-skill
// Ds Deep Space row and the selection rows.
func TestGrantFor(t *testing.T) {
	tests := []struct {
		tc     string
		kind   world.GrantKind
		skills []string
	}{
		{"Ag", world.GrantSkill, []string{"Animals"}},
		{"Ds", world.GrantSkill, []string{"Vacc Suit", "Zero-G"}},
		{"Hi", world.GrantSkill, []string{"Streetwise"}},
		{"Ph", world.GrantNone, nil},
		{"Ri", world.GrantArt, nil},
		{"In", world.GrantTrade, nil},
		{"Wa", world.GrantSkill, []string{"Seafarer"}},
	}

	for _, tt := range tests {
		grant, err := world.GrantFor(tt.tc)
		if err != nil {
			t.Fatalf("GrantFor(%q): %v", tt.tc, err)
		}

		if grant.Kind != tt.kind || len(grant.Skills) != len(tt.skills) {
			t.Errorf("GrantFor(%q) = %+v, want kind %q skills %v", tt.tc, grant, tt.kind, tt.skills)

			continue
		}

		for i, want := range tt.skills {
			if grant.Skills[i] != want {
				t.Errorf("GrantFor(%q) skill %d = %q, want %q", tt.tc, i, grant.Skills[i], want)
			}
		}
	}

	if _, err := world.GrantFor("Zz"); !errors.Is(err, world.ErrUnknownTC) {
		t.Errorf("GrantFor(Zz) = %v, want ErrUnknownTC", err)
	}
}

// TestChoices verifies the chart B selection lists (p. 56).
func TestChoices(t *testing.T) {
	arts, err := world.ArtChoices()
	if err != nil {
		t.Fatal(err)
	}

	if len(arts) != 6 || arts[0] != "Actor" || arts[5] != "Musician" {
		t.Errorf("ArtChoices() = %v", arts)
	}

	trades, err := world.TradeChoices()
	if err != nil {
		t.Fatal(err)
	}

	if len(trades) != 10 || trades[0] != "Biologics" || trades[9] != "Programmer" {
		t.Errorf("TradeChoices() = %v", trades)
	}
}

// TestDefault verifies the tool-owned default homeworld (docs/PRD.md FR2):
// Regina, chart B row R (p. 56).
func TestDefault(t *testing.T) {
	d, err := world.Default()
	if err != nil {
		t.Fatal(err)
	}

	if d.Name != "Regina" || d.UWP != "A788899-C" {
		t.Errorf("Default() = %+v", d)
	}

	if len(d.TradeClassifications) != 3 || d.TradeClassifications[0] != "Ph" ||
		d.TradeClassifications[1] != "Pa" || d.TradeClassifications[2] != "Ri" {
		t.Errorf("Default() TCs = %v, want [Ph Pa Ri]", d.TradeClassifications)
	}

	if got, want := d.Label(), "Regina A788899-C (Ph Pa Ri)"; got != want {
		t.Errorf("Label() = %q, want %q", got, want)
	}
}
