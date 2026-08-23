package dice

// Test bridges for the pure resolution arithmetic, so the external test
// package can enumerate every face combination exhaustively.
var (
	NewFlux      = newFlux
	ResolveThrow = resolveThrow
	ResolveCheck = resolveCheck
	ResolveUnder = resolveUnder
)
