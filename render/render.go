// Package render derives human-readable output from the character record.
// The JSON record is the source of truth; everything here is a pure
// function of it (docs/PRD.md FR8, goal 4, goal 5).
package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/t5chargen/chargen"
)

// Sheet renders the Markdown character sheet, modeled on the Character Card
// (Book 1 chart C1, p. 98). Only fields the record holds are rendered; card
// fields for later mechanics (gender, birthdate, ranks, skills) appear as
// the mechanics land.
func Sheet(c chargen.Character) string {
	var b strings.Builder

	b.WriteString("# Character Card\n\n")

	// Names are blank by default (docs/PRD.md, Decisions); keep the card
	// field present either way, without trailing whitespace.
	if c.Name == "" {
		b.WriteString("**Name**:\n\n")
	} else {
		fmt.Fprintf(&b, "**Name**: %s\n\n", c.Name)
	}

	fmt.Fprintf(&b, "**UPP**: %s\n\n", c.UPP)

	b.WriteString("| Str | Dex | End | Int | Edu | Soc |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	fmt.Fprintf(&b, "| %d | %d | %d | %d | %d | %d |\n\n",
		c.Characteristics.Str, c.Characteristics.Dex, c.Characteristics.End,
		c.Characteristics.Int, c.Characteristics.Edu, c.Characteristics.Soc)

	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "Seed %d (%s) · schema %s · engine %s · policy %s\n\n",
		c.RNG.Seed, c.RNG.Algorithm, c.SchemaVersion, c.EngineVersion, c.PolicyVersion)
	fmt.Fprintf(&b, "Ruleset: %s\n", c.Ruleset)

	return b.String()
}

// History renders the generation record as a Markdown transcript — the
// narrative purpose of the event log (docs/PRD.md FR10): the character's
// biography in game terms.
func History(c chargen.Character) string {
	var b strings.Builder

	b.WriteString("# Generation Record\n")

	for _, event := range c.Events {
		b.WriteString(eventLine(event))
	}

	return b.String()
}

// eventLine renders one event of the transcript.
func eventLine(event chargen.Event) string {
	switch event.Kind {
	case chargen.EventStep:
		return fmt.Sprintf("\n## %s\n\n_%s_\n\n", event.Step.Name, event.Step.Cite)
	case chargen.EventThrow:
		return throwLine(event.Seq, event.Throw)
	case chargen.EventChoice:
		return choiceLine(event.Seq, event.Choice)
	case chargen.EventConsequence:
		return consequenceLine(event.Seq, event.Consequence)
	}

	return fmt.Sprintf("- #%d (%s)\n", event.Seq, event.Kind)
}

// throwLine renders a throw event: dice expression, individual dice, target
// and modifiers when present, and the rule citation.
func throwLine(seq int, throw *chargen.ThrowEvent) string {
	var line strings.Builder

	fmt.Fprintf(&line, "- #%d %s = %s = %d", seq, throw.Expr, joinDice(throw.Dice), throw.Total)

	if throw.Target != nil {
		fmt.Fprintf(&line, " vs %d", *throw.Target)

		for _, mod := range throw.Mods {
			fmt.Fprintf(&line, " (%s %+d)", mod.Name, mod.Value)
		}

		if throw.Success != nil {
			if *throw.Success {
				line.WriteString(": success")
			} else {
				line.WriteString(": failure")
			}
		}
	}

	return line.String() + fmt.Sprintf(" — %s\n", throw.Cite)
}

// choiceLine renders a choice event: who decided, the alternatives, and
// the selection.
func choiceLine(seq int, choice *chargen.ChoiceEvent) string {
	return fmt.Sprintf("- #%d %s chose %q of [%s]: %s — %s\n",
		seq, choice.Decider, choice.Options[choice.Chosen],
		strings.Join(choice.Options, ", "), choice.Prompt, choice.Cite)
}

// consequenceLine renders a consequence event indented under its causing
// throw or choice, deriving the readable line from the structured payload.
func consequenceLine(seq int, consequence *chargen.ConsequenceEvent) string {
	if consequence.Kind == chargen.ConsequenceCharacteristicSet {
		return fmt.Sprintf("  - #%d (from #%d) %s = %d\n",
			seq, consequence.Cause, consequence.Characteristic, consequence.Value)
	}

	return fmt.Sprintf("  - #%d (from #%d) %s\n", seq, consequence.Cause, consequence.Kind)
}

// joinDice renders individual die faces as "6+1".
func joinDice(faces []int) string {
	parts := make([]string, len(faces))
	for i, face := range faces {
		parts[i] = strconv.Itoa(face)
	}

	return strings.Join(parts, "+")
}
