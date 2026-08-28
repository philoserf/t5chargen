package chargen

import "slices"

// The deviations this engine applies, as ERRATA.md classifies them.
//
// A Deviation departs knowingly from the printed rule, and docs/PRD.md
// requires every character JSON to carry "any applied `ERRATA.md`
// deviations". These are the identifiers that go in that field; the
// entries themselves carry the reading and the reasoning.
const (
	// DeviationLandGrantFloor is I-82: a Land Grant hex on a world the
	// record does not name is priced at the no-TC floor (p. 88; p. 79).
	DeviationLandGrantFloor = "I-82"

	// DeviationWorldKnowledgeTerms is I-112: a World Knowledge counts the
	// terms from p. 134's age 2 to the age career resolution began, and
	// not the whole life the page's own example counts (p. 134).
	DeviationWorldKnowledgeTerms = "I-112"
)

// Deviations are every ERRATA.md entry classified as a Deviation, in the
// order they are stamped. The order is the entries' own numbering rather
// than a sort: "I-112" sorts before "I-82" as a string, which would read
// as a mistake in every record that carried both.
//
// audit/docs_test.go holds this set equal to ERRATA.md's Deviation
// headings, both directions, so an entry classified as a Deviation that
// the engine does not declare fails the gate rather than passing
// quietly. Declaring is not stamping: what binds a declared entry to a
// site that applies it is its case in TestDeviationsAreStamped, and a
// new entry needs one.
var Deviations = []string{
	DeviationLandGrantFloor,
	DeviationWorldKnowledgeTerms,
}

// applyDeviation records that a deviation governed a value in this
// record, once however many times its rule fired.
//
// "Applied" means the deviating rule governed a value here, not that the
// value provably differs from what the printed rule would have given.
// The stronger reading is not decidable for I-82: its counterfactual is
// the per-title hex table, and inventing that is what I-83 declined to
// do. The weaker one is what character.schema.json describes and what an
// auditor can act on — it names the readings this record was produced
// under.
func (c *Character) applyDeviation(entry string) {
	if !slices.Contains(c.Errata, entry) {
		c.Errata = append(c.Errata, entry)
	}

	// Kept in the declared order rather than append order, so two records
	// applying the same deviations list them the same way regardless of
	// which fired first.
	slices.SortFunc(c.Errata, func(a, b string) int {
		return slices.Index(Deviations, a) - slices.Index(Deviations, b)
	})
}
