package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// resignPrompt identifies the offer in a record's event log. ChoiceEvent
// records the prompt rather than the ID, so this is what a reader of the
// log has to go on too.
const resignPrompt = "Resign from the Reserves?"

// reserveSeedSearch is how wide the sweeps look. A character has to
// survive an Armed Forces career to be enrolled at all, which is the
// narrow part.
const reserveSeedSearch = 300

// resigns takes p. 67's resignation wherever it is offered, and forces
// the named service.
//
// The auto policy remains (POLICY.md), which is why no golden record
// reaches this path and a decider is what reaches it.
type resigns struct{ playerKind }

func (resigns) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseResignReserves {
		if i := slices.Index(c.Options, "Resign"); i >= 0 {
			return i, nil
		}
	}

	return autoPolicy(c)
}

// TestAResignedCharacterLeavesTheReserves verifies p. 67: "A character may
// resign from the Reserves (Check Continue) and forego its benefits and
// responsibilities."
//
// The Check can fail, and a failed Check leaves him enrolled — so the
// assertion is that resignation and enrolment agree with each other, not
// that asking is enough.
func TestAResignedCharacterLeavesTheReserves(t *testing.T) {
	resigned, remained := 0, 0

	for seed := range uint64(reserveSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Career: "Marine", Decider: resigns{}})

		for _, record := range c.Careers {
			if record.Career != "Marine" || !record.Began {
				continue
			}

			if resignedFrom(c) {
				resigned++

				if record.Reserve {
					t.Errorf("seed %d: resigned and is still in the Reserves", seed)
				}
			} else if record.Reserve {
				remained++
			}
		}
	}

	if resigned == 0 {
		t.Fatalf("no seed under %d resigns; the sweep is asserting nothing", reserveSeedSearch)
	}

	if remained == 0 {
		t.Fatalf("no seed under %d stays enrolled; a failed Check is never seen", reserveSeedSearch)
	}
}

// declinedOffers counts the resignations offered in one record and checks
// each: that the policy remained, and that remaining threw nothing.
//
// The throw is matched on the substring the cite is built from rather
// than the whole string, which ends in the career's own Continue label —
// a wrong guess at that label would make this look like a pass.
func declinedOffers(t *testing.T, c chargen.Character, seed uint64) int {
	t.Helper()

	offers := 0

	for i, event := range c.Events {
		if event.Choice == nil || event.Choice.Prompt != resignPrompt {
			continue
		}

		offers++

		if got := event.Choice.Options[event.Choice.Chosen]; got != "Remain in the Reserves" {
			t.Fatalf("seed %d: the policy answered %q, want to remain", seed, got)
		}

		if i+1 < len(c.Events) && c.Events[i+1].Throw != nil &&
			strings.Contains(c.Events[i+1].Throw.Cite, "Resign from the Reserves") {
			t.Errorf("seed %d: declining threw %q anyway", seed, c.Events[i+1].Throw.Cite)
		}
	}

	return offers
}

// resignedFrom reports whether the record shows a resignation.
func resignedFrom(c chargen.Character) bool {
	return slices.ContainsFunc(c.Events, func(e chargen.Event) bool {
		return e.Consequence != nil && e.Consequence.Kind == chargen.ConsequenceResigned
	})
}

// TestDecliningToResignThrowsNothing verifies the ordering the rule needs:
// the offer comes first and the Check follows only acceptance
// (interpretation I-55).
//
// This is what makes the rule implementable. A Check thrown before the
// decision would spend two faces of the seeded stream in every Armed
// Forces character, for a decision the default policy never takes — which
// is the reason the rule was deferred for four milestones.
func TestDecliningToResignThrowsNothing(t *testing.T) {
	offers := 0

	for seed := range uint64(reserveSeedSearch) {
		offers += declinedOffers(t, generate(t, chargen.Options{Seed: seed, Career: "Marine"}), seed)
	}

	if offers == 0 {
		t.Fatalf("no seed under %d is offered the resignation; the sweep is asserting nothing",
			reserveSeedSearch)
	}
}

// TestADeadMarineIsNotOfferedTheReserves verifies enrolment and the offer
// both stop at death. "A character who leaves a military, naval, or
// marine career is automatically in the Reserves" (p. 67) is about a
// character who left it, and death ends career resolution (I-51).
func TestADeadMarineIsNotOfferedTheReserves(t *testing.T) {
	dead := 0

	for seed := range uint64(reserveSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Career: "Marine"})
		if !c.Dead {
			continue
		}

		dead++

		for _, event := range c.Events {
			if event.Choice != nil && event.Choice.Prompt == resignPrompt {
				t.Fatalf("seed %d: a dead character was offered the resignation", seed)
			}
		}
	}

	if dead == 0 {
		t.Fatalf("no seed under %d dies as a Marine; the sweep is asserting nothing", reserveSeedSearch)
	}
}
