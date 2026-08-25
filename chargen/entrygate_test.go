package chargen_test

import (
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// TestARefusalToBeginNamesAThrow holds the line between a prerequisite and
// a failed attempt, which p. 65 draws: "Pre-Requisites. Some Careers have
// requirements before a character may attempt to Begin", and separately
// "If both Begin and Retry fail, this career may not be used. Each failed
// attempt (both Begin or Retry) takes one year."
//
// A career that refuses a character must therefore have rolled for it. A
// refusal caused by the selection itself is a prerequisite enforced too
// late — the character was offered a career he could never have, chose it,
// and was told afterwards. That is how chart 11's "if Soc B+" hid until it
// was reported from play (I-28), and this is the guard that keeps the next
// one from doing the same.
func TestARefusalToBeginNamesAThrow(t *testing.T) {
	refusals := 0

	for _, name := range career.Available() {
		for seed := uint64(1); seed <= 40; seed++ {
			c, open := generateIfOpen(t, chargen.Options{Seed: seed, Career: name})
			if !open {
				continue // the rules deny the career outright; nothing was attempted
			}

			kinds := make(map[int]chargen.EventKind, len(c.Events))
			for _, e := range c.Events {
				kinds[e.Seq] = e.Kind
			}

			for _, e := range c.Events {
				if e.Kind != chargen.EventConsequence ||
					e.Consequence.Kind != chargen.ConsequenceCareerNotBegun {
					continue
				}

				refusals++

				if kind := kinds[e.Consequence.Cause]; kind != chargen.EventThrow {
					t.Errorf("%s seed %d: refused to begin on a %s at event %d, not a throw — "+
						"a prerequisite enforced after the character chose it",
						name, seed, kind, e.Consequence.Cause)
				}
			}
		}
	}

	if refusals == 0 {
		t.Fatal("no career refused anybody across the sweep; the guard is asserting nothing")
	}
}
