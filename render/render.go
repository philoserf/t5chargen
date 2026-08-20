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

	if c.Homeworld.UWP != "" {
		fmt.Fprintf(&b, "**Homeworld**: %s\n\n", c.Homeworld.Label())
	}

	fmt.Fprintf(&b, "**Age**: %d\n\n", c.Age)

	b.WriteString("| Str | Dex | End | Int | Edu | Soc |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	fmt.Fprintf(&b, "| %d | %d | %d | %d | %d | %d |\n\n",
		c.Characteristics.Str, c.Characteristics.Dex, c.Characteristics.End,
		c.Characteristics.Int, c.Characteristics.Edu, c.Characteristics.Soc)

	for _, educationRecord := range c.Education {
		b.WriteString(educationLine(educationRecord))
	}

	for _, careerRecord := range c.Careers {
		b.WriteString(careerLine(careerRecord))
	}

	if len(c.Skills) > 0 {
		fmt.Fprintf(&b, "**Skills**: %s\n\n", skillList(c.Skills))
	}

	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "Seed %d (%s) · schema %s · engine %s · policy %s\n\n",
		c.RNG.Seed, c.RNG.Algorithm, c.SchemaVersion, c.EngineVersion, c.PolicyVersion)
	fmt.Fprintf(&b, "Ruleset: %s\n", c.Ruleset)

	return b.String()
}

// educationLine renders one educational process of the record.
func educationLine(record chargen.EducationRecord) string {
	line := "**Education**: " + record.Program

	if record.Service != "" {
		line += " (" + record.Service + ")"
	}

	if record.Skill != "" {
		line += ", " + record.Skill
	}

	if record.Major != "" {
		line += ", Major " + record.Major
	}

	if record.Minor != "" {
		line += ", Minor " + record.Minor
	}

	if record.Degree != "" {
		line += " — " + record.Degree
	} else if !record.Graduated {
		line += " — did not graduate"
	}

	return line + "\n\n"
}

// careerLine renders one career of the record: name, terms served, and the
// Citizen Job and Hobby when determined.
func careerLine(record chargen.CareerRecord) string {
	line := fmt.Sprintf("**Career**: %s (%d terms)", record.Career, len(record.Terms))

	if record.Job != "" {
		line += ", Job " + record.Job
	}

	if record.Hobby != "" {
		line += ", Hobby " + record.Hobby
	}

	return line + "\n\n"
}

// skillList renders skills as "Admin-2, Broker-1" in the record's sorted
// order.
func skillList(skills []chargen.Skill) string {
	parts := make([]string, len(skills))
	for i, skill := range skills {
		parts[i] = skill.Name + "-" + strconv.Itoa(skill.Level)
	}

	return strings.Join(parts, ", ")
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

// eventLine renders one event of the transcript. Records come from disk
// with minimal validation, so a kind whose payload is missing renders as a
// marked malformed line instead of panicking.
func eventLine(event chargen.Event) string {
	switch {
	case event.Kind == chargen.EventStep && event.Step != nil:
		return fmt.Sprintf("\n## %s\n\n_%s_\n\n", event.Step.Name, event.Step.Cite)
	case event.Kind == chargen.EventThrow && event.Throw != nil:
		return throwLine(event.Seq, event.Throw)
	case event.Kind == chargen.EventChoice && event.Choice != nil:
		return choiceLine(event.Seq, event.Choice)
	case event.Kind == chargen.EventConsequence && event.Consequence != nil:
		return consequenceLine(event.Seq, event.Consequence)
	}

	return fmt.Sprintf("- #%d (%s) [malformed event]\n", event.Seq, event.Kind)
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
// the selection. An out-of-range Chosen (corrupted or hand-edited record)
// renders marked rather than panicking.
func choiceLine(seq int, choice *chargen.ChoiceEvent) string {
	selected := fmt.Sprintf("[chosen %d out of range]", choice.Chosen)
	if choice.Chosen >= 0 && choice.Chosen < len(choice.Options) {
		selected = fmt.Sprintf("%q", choice.Options[choice.Chosen])
	}

	return fmt.Sprintf("- #%d %s chose %s of [%s]: %s — %s\n",
		seq, choice.Decider, selected,
		strings.Join(choice.Options, ", "), choice.Prompt, choice.Cite)
}

// consequenceLine renders a consequence event indented under its causing
// throw or choice, deriving the readable line from the structured payload.
func consequenceLine(seq int, consequence *chargen.ConsequenceEvent) string {
	return fmt.Sprintf("  - #%d (from #%d) %s\n", seq, consequence.Cause, consequenceText(consequence))
}

// consequenceText derives the readable body of a consequence line; award
// kinds here, career-flow kinds in consequenceFlowText.
//
//nolint:exhaustive // Deliberately partitioned: the default defers the flow kinds to consequenceFlowText.
func consequenceText(c *chargen.ConsequenceEvent) string {
	switch c.Kind {
	case chargen.ConsequenceCharacteristicSet:
		return fmt.Sprintf("%s = %d", c.Characteristic, c.Value)
	case chargen.ConsequenceCharacteristicChange:
		return fmt.Sprintf("%s %+d = %d", c.Characteristic, c.Delta, c.Value)
	case chargen.ConsequenceSkillAwarded:
		return fmt.Sprintf("%s %+d = %s-%d", c.Skill, c.Delta, c.Skill, c.Value)
	case chargen.ConsequenceJobSet:
		return "Job = " + c.Skill
	case chargen.ConsequenceHobbySet:
		return "Hobby = " + c.Skill
	default:
		return consequenceFlowText(c)
	}
}

// consequenceFlowText renders the career-flow consequence kinds.
//
//nolint:exhaustive // Deliberately partitioned: the award kinds are handled by consequenceText.
func consequenceFlowText(c *chargen.ConsequenceEvent) string {
	switch c.Kind {
	case chargen.ConsequenceNoAward:
		if c.Skill != "" {
			// The cap-absorption path sets Skill (p. 134 Skill-15 cap).
			return fmt.Sprintf("no award (%s at the Skill-%d cap)", c.Skill, chargen.SkillMax)
		}

		return "no award"
	case chargen.ConsequenceJobUndetermined:
		return "Job undetermined (No Skill); retries next success — ERRATA I-1"
	case chargen.ConsequenceBenefitLost:
		if c.Characteristic != "" {
			// The p. 68 human characteristic maximum.
			return fmt.Sprintf("benefit lost (%s at the characteristic-%d maximum)",
				c.Characteristic, chargen.CharacteristicMax)
		}

		return "benefit lost (no Major/Minor)"
	case chargen.ConsequenceMandatoryContinue:
		return "mandatory continue"
	case chargen.ConsequenceYearsElapsed:
		return fmt.Sprintf("%+d years", c.Value)
	case chargen.ConsequenceCareerEnded:
		return "career ended (" + c.Career + ")"
	default:
		return string(c.Kind)
	}
}

// joinDice renders individual die faces as "6+1".
func joinDice(faces []int) string {
	parts := make([]string, len(faces))
	for i, face := range faces {
		parts[i] = strconv.Itoa(face)
	}

	return strings.Join(parts, "+")
}
