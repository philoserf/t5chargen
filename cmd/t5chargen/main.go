// Command t5chargen generates Traveller5 characters. See docs/PRD.md.
//
// Planned subcommands: new, batch, render, replay. Nothing is implemented
// yet; milestone 1 (dice engine + characteristics + record/render walking
// skeleton) is the next step.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "t5chargen: not yet implemented — see docs/PRD.md")
	os.Exit(1)
}
