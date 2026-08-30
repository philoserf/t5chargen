package chargen_test

import "github.com/philoserf/t5chargen/chargen"

// playerKind supplies the Kind every test fake reports.
//
// The engine records which side answered each choice, and a fake stands in
// for a player rather than for the policy — so all forty-odd of them
// returned DeciderPlayer, each from its own copy of the same line.
// Embedding this says the same thing once, and a new fake now has one
// method to write instead of two.
//
// It is deliberately not a Decider itself: Choose is what makes a fake
// worth having, and a base that supplied a default would let a fake
// silently answer nothing.
type playerKind struct{}

func (playerKind) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }
