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

	if c.Fame > 0 || c.WoundBadges > 0 || c.Disabled || c.Dead {
		b.WriteString(statusLine(c))
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

	switch {
	case record.Degree != "":
		line += " — " + record.Degree
	case record.Graduated && record.Honors:
		line += " — graduated with Honors"
	case record.Graduated:
		line += " — graduated"
	default:
		line += " — did not graduate"
	}

	return line + "\n\n"
}

// careerLine renders one career of the record: name, terms served, and the
// Citizen Job and Hobby when determined.
func careerLine(record chargen.CareerRecord) string {
	line := fmt.Sprintf("**Career**: %s (%s)", record.Career, plural(len(record.Terms), "term"))

	if record.Job != "" {
		line += ", Job " + record.Job
	}

	if record.Hobby != "" {
		line += ", Hobby " + record.Hobby
	}

	line += careerValues(record)

	if !record.Began {
		line += " — did not begin"
	}

	return line + "\n\n"
}

// careerValues renders the values a career tracks of its own: the
// Scholar's publications and Tenure, the Entertainer's specialty and
// Talent, rank, the Scout's Discoveries, and the Merchant's Ship Shares.
func careerValues(record chargen.CareerRecord) string {
	var parts []string

	parts = append(parts, careerSpecificValues(record)...)

	if record.Specialty != "" {
		parts = append(parts, record.Specialty)
	}

	if record.Talent > 0 {
		parts = append(parts, fmt.Sprintf("Talent %d", record.Talent))
	}

	if record.RankTitle != "" {
		parts = append(parts, record.RankTitle+" "+record.Rank)
	}

	if record.Discoveries > 0 {
		parts = append(parts, plural(record.Discoveries, "Discovery"))
	}

	if record.ShipShares > 0 {
		parts = append(parts, plural(record.ShipShares, "Ship Share"))
	}

	if len(parts) == 0 {
		return ""
	}

	return ", " + strings.Join(parts, ", ")
}

// rogueValues renders what chart 10 tracks: the Rogue's scheme, his
// takings, and any sentence still owed.
func rogueValues(record chargen.CareerRecord) []string {
	var parts []string

	if record.Scheme != "" {
		parts = append(parts, "scheming as "+record.Scheme)
	}

	if record.SchemePayoff > 0 {
		parts = append(parts, fmt.Sprintf("Cr%d in payoffs", record.SchemePayoff))
	}

	if record.PrisonYears > 0 {
		parts = append(parts, plural(record.PrisonYears, "year")+" owed in prison")
	}

	return parts
}

// careerSpecificValues renders the state only some careers track: the
// Soldier's Branch and medals, the Noble's land grants and exile, the
// Scholar's publications, the Rogue's schemes.
func careerSpecificValues(record chargen.CareerRecord) []string {
	var parts []string

	parts = append(parts, rogueValues(record)...)

	if record.UndercoverCareer != "" {
		cover := "undercover as " + record.UndercoverCareer
		if record.UndercoverTitle != "" {
			cover += " (" + record.UndercoverTitle + ")"
		}

		parts = append(parts, cover)
	}

	if record.Commendations > 0 {
		parts = append(parts, plural(record.Commendations, "Commendation"))
	}

	if record.Branch != "" {
		parts = append(parts, record.Branch)
	}

	for _, award := range record.Medals {
		parts = append(parts, fmt.Sprintf("%s-%d", award.Code, award.Count))
	}

	if record.LandGrants > 0 {
		parts = append(parts, plural(record.LandGrants, "Land Grant"))
	}

	if record.Exiled {
		parts = append(parts, "in Exile")
	}

	if record.Publications > 0 {
		parts = append(parts, plural(record.Publications, "Publication"))
	}

	if record.Tenured {
		parts = append(parts, "Tenured")
	}

	return parts
}

// plural renders "1 term" / "2 terms" / "1 Discovery" / "2 Discoveries".
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}

	if noun == "Discovery" {
		noun = "Discoveries"
	} else {
		noun += "s"
	}

	return strconv.Itoa(n) + " " + noun
}

// statusLine renders fame, wounds, and disabled/dead status.
func statusLine(c chargen.Character) string {
	parts := []string{}

	if c.Fame > 0 {
		parts = append(parts, fmt.Sprintf("Fame %d", c.Fame))
	}

	if c.WoundBadges > 0 {
		parts = append(parts, fmt.Sprintf("Wound Badges %d", c.WoundBadges))
	}

	if c.Disabled {
		parts = append(parts, "Disabled")
	}

	if c.Dead {
		parts = append(parts, "DEAD")
	}

	return "**Status**: " + strings.Join(parts, ", ") + "\n\n"
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
		return consequenceInjuryText(c)
	}
}

// consequenceInjuryText renders the injury and reward consequence kinds.
//
//nolint:exhaustive // Deliberately partitioned: earlier kinds are handled upstream.
func consequenceInjuryText(c *chargen.ConsequenceEvent) string {
	switch c.Kind {
	case chargen.ConsequenceCareerNotBegun:
		return "career not begun (" + c.Career + ")"
	case chargen.ConsequenceWoundBadge:
		return fmt.Sprintf("Wound Badge (total %d)", c.Value)
	case chargen.ConsequenceDisabled:
		return "disabled (" + c.Characteristic + " reduced by 4+); musters out at term end"
	case chargen.ConsequenceDead:
		return "DEAD (" + c.Characteristic + " reduced to zero)"
	case chargen.ConsequenceDiscovery:
		return fmt.Sprintf("Discovery (total %d)", c.Value)
	case chargen.ConsequenceFameChange:
		return fmt.Sprintf("Fame %+d = %d", c.Delta, c.Value)
	case chargen.ConsequenceRankSet:
		return "rank " + c.Skill
	case chargen.ConsequenceShipShares:
		return fmt.Sprintf("%s (total %d)", plural(c.Delta, "Ship Share"), c.Value)
	default:
		return consequenceCareerValueText(c)
	}
}

// consequenceCareerValueText renders the consequence kinds for the values a
// career tracks of its own (chart 03's Fame and Talent, p. 77).
//
//nolint:exhaustive // Deliberately partitioned: earlier kinds are handled upstream.
func consequenceCareerValueText(c *chargen.ConsequenceEvent) string {
	switch c.Kind {
	case chargen.ConsequenceSpecialtySet:
		return "specialty " + c.Skill
	case chargen.ConsequenceTalentSet:
		return fmt.Sprintf("Talent = %d", c.Value)
	case chargen.ConsequenceComeback:
		return fmt.Sprintf("Comeback (Fame reset to %d)", c.Value)
	default:
		return consequenceScholarText(c)
	}
}

// consequenceScholarText renders the chart 02 consequence kinds.
//
//nolint:exhaustive // Deliberately partitioned: earlier kinds are handled upstream.
func consequenceScholarText(c *chargen.ConsequenceEvent) string {
	switch c.Kind {
	case chargen.ConsequencePublication:
		if c.Delta > 1 {
			return fmt.Sprintf("Award-Winning Publication, counting as %d (total %d)", c.Delta, c.Value)
		}

		return fmt.Sprintf("Publication (total %d)", c.Value)
	case chargen.ConsequenceTenure:
		return "Tenure granted"
	case chargen.ConsequenceWaived:
		return "waived"
	case chargen.ConsequenceMajorSet:
		return "Major = " + c.Skill
	case chargen.ConsequenceMinorSet:
		return "Minor = " + c.Skill
	default:
		return consequenceNobleText(c)
	}
}

// consequenceNobleText renders the chart 11 consequence kinds.
//
//nolint:exhaustive // Deliberately partitioned: earlier kinds are handled upstream.
func consequenceNobleText(c *chargen.ConsequenceEvent) string {
	switch c.Kind {
	case chargen.ConsequenceExiled:
		return fmt.Sprintf("Exiled (total %d)", c.Value)
	case chargen.ConsequenceReturned:
		return "returned from Exile"
	case chargen.ConsequenceIntrigue:
		return fmt.Sprintf("successful Intrigue (total %d)", c.Value)
	case chargen.ConsequenceElevated:
		return "Elevated"
	case chargen.ConsequenceLandGrant:
		return fmt.Sprintf("Land Grant (total %d)", c.Value)
	default:
		return consequenceArmedForcesText(c)
	}
}

// consequenceArmedForcesText renders the chart 08 Armed Forces and
// chart 09 Agent consequence kinds.
//
//nolint:exhaustive // Deliberately partitioned: earlier kinds are handled upstream.
func consequenceArmedForcesText(c *chargen.ConsequenceEvent) string {
	switch c.Kind {
	case chargen.ConsequenceUndercover:
		if c.Skill == "" {
			return "undercover as " + c.Career
		}

		return "undercover as " + c.Career + " (" + c.Skill + ")"
	case chargen.ConsequenceCommendation:
		return fmt.Sprintf("Commendation (total %d)", c.Value)
	case chargen.ConsequenceBranchSet:
		return "Branch " + c.Skill
	case chargen.ConsequenceOperation:
		return fmt.Sprintf("assignment %s (Mod %d)", c.Skill, c.Value)
	case chargen.ConsequenceMedal:
		return fmt.Sprintf("%s (total %d)", c.Skill, c.Value)
	case chargen.ConsequenceServiceBadge:
		return fmt.Sprintf("Exemplary Service Badge (total %d)", c.Value)
	default:
		return consequenceRogueText(c)
	}
}

// consequenceRogueText renders the chart 10 consequence kinds.
//
//nolint:exhaustive // Deliberately partitioned: earlier kinds are handled upstream.
func consequenceRogueText(c *chargen.ConsequenceEvent) string {
	switch c.Kind {
	case chargen.ConsequenceScheme:
		return fmt.Sprintf("Scheme: %s (Flux %+d)", c.Skill, c.Value)
	case chargen.ConsequenceSentenced:
		return "sentenced to " + plural(c.Value, "year") + " in prison"
	case chargen.ConsequenceImprisoned:
		return "in prison, serving " + plural(c.Value, "year")
	case chargen.ConsequencePayoff:
		return fmt.Sprintf("Payoff %s: Cr%d (total Cr%d)", c.Skill, c.Delta, c.Value)
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
