package dice

// FluxKind identifies which Flux variant produced a FluxRoll.
type FluxKind string

// The three Flux variants of Book 1 p. 19 (detailed in Appendix 8, p. 261).
const (
	FluxStandard FluxKind = "flux"
	FluxGood     FluxKind = "good"
	FluxBad      FluxKind = "bad"
)

// FluxRoll is the recorded outcome of a Flux, Good Flux, or Bad Flux roll.
// First and Second are the two die faces in roll order; for standard Flux,
// First is the light die and Second the dark die.
type FluxRoll struct {
	Kind   FluxKind `json:"kind"`
	First  int      `json:"first"`
	Second int      `json:"second"`
	Value  int      `json:"value"`
}

// Expr renders the roll's dice expression in Book 1 notation (pp. 19, 261).
func (f FluxRoll) Expr() string {
	switch f.Kind {
	case FluxGood:
		return "Good Flux"
	case FluxBad:
		return "Bad Flux"
	case FluxStandard:
		return "Flux"
	}

	return "Flux"
}

// newFlux resolves the value of a Flux variant from two die faces. Pure
// arithmetic, split from the stream so tests can enumerate all 36 face
// combinations against the printed probability tables (p. 261).
func newFlux(kind FluxKind, first, second int) FluxRoll {
	roll := FluxRoll{Kind: kind, First: first, Second: second}

	switch kind {
	case FluxGood:
		// "Good Flux is High Die minus Low Die." (p. 261)
		roll.Value = max(first, second) - min(first, second)
	case FluxBad:
		// "Bad Flux is Low Die minus High Die." (p. 261)
		roll.Value = min(first, second) - max(first, second)
	case FluxStandard:
		// "Flux is Light Die minus Dark Die." (p. 261)
		roll.Value = first - second
	}

	return roll
}

// Flux rolls standard Flux, producing a value from -5 to +5.
//
// "Flux is rolled with two dice. Roll 1D. Roll a second 1D and subtract it
// from the first. This process is most easily done with a light die and a
// dark die: roll the two dice and subtract the dark from the light. Flux is
// Light Die minus Dark Die." (p. 261)
//
// The light die is consumed from the stream first.
func (r *Roller) Flux() FluxRoll {
	light, dark := r.face(), r.face()

	return newFlux(FluxStandard, light, dark)
}

// GoodFlux rolls Good Flux, producing a value from 0 to +5.
//
// "Good Flux is a variant of Flux which produces only positive results
// (average +2, ranges from 0 to +5). Roll 2D and subtract the smaller from
// the larger. Good Flux is High Die minus Low Die" (p. 261).
func (r *Roller) GoodFlux() FluxRoll {
	first, second := r.face(), r.face()

	return newFlux(FluxGood, first, second)
}

// BadFlux rolls Bad Flux, producing a value from -5 to 0.
//
// "Bad Flux is a variant of Flux which produces only negative results
// (average -2, ranges from 0 to -5). Roll 2D and subtract the larger from
// the smaller. Bad Flux is Low Die minus High Die" (p. 261).
func (r *Roller) BadFlux() FluxRoll {
	first, second := r.face(), r.face()

	return newFlux(FluxBad, first, second)
}
