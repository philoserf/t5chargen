package chargen_test

import (
	"slices"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// TestDeviationsAreStamped verifies the provenance contract for the
// deviations this engine applies: "Every character JSON carries ... any
// applied `ERRATA.md` deviations" (docs/PRD.md).
//
// A deviation is stamped when its rule governed a value in the record —
// not when the value provably differs from the printed rule, which is
// not a decidable question for I-82. Its counterfactual is the per-title
// hex table, and inventing that is exactly what I-83 declined to do.
func TestDeviationsAreStamped(t *testing.T) {
	for _, tt := range []struct {
		name, career string
		seed         uint64
		want         []string
	}{
		{
			name:   "a Noble holds a Land Grant, so both apply",
			career: "Noble",
			seed:   2978,
			want:   []string{chargen.DeviationLandGrantFloor, chargen.DeviationWorldKnowledgeTerms},
		},
		{
			name:   "a Scout's Discovery grants are priced the same way",
			career: "Scout",
			seed:   26,
			want:   []string{chargen.DeviationLandGrantFloor, chargen.DeviationWorldKnowledgeTerms},
		},
		{
			name:   "a career that grants no land applies only the World Knowledge count",
			career: "Citizen",
			seed:   1,
			want:   []string{chargen.DeviationWorldKnowledgeTerms},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := generate(t, chargen.Options{Seed: tt.seed, Career: tt.career})

			if !slices.Equal(c.Errata, tt.want) {
				t.Errorf("errata = %v, want %v", c.Errata, tt.want)
			}
		})
	}
}

// TestTheLandGrantStampFollowsTheGrant verifies that I-82 is stamped for
// the character who holds a grant and not for the one who does not. The
// stamp is a claim about this record, so a character who never earned a
// Land Grant must not carry it.
func TestTheLandGrantStampFollowsTheGrant(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 1, Career: "Citizen"})

	for _, record := range c.Careers {
		if record.LandGrants != 0 {
			t.Fatalf("the seed no longer produces a character without Land Grants: %s holds %d",
				record.Career, record.LandGrants)
		}
	}

	if slices.Contains(c.Errata, chargen.DeviationLandGrantFloor) {
		t.Errorf("errata = %v, which stamps %s for a character holding no Land Grant",
			c.Errata, chargen.DeviationLandGrantFloor)
	}
}
