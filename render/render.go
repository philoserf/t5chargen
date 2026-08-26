// Package render derives human-readable output from the character record.
// The JSON record is the source of truth; everything here is a pure
// function of it (docs/PRD.md FR8, goal 4, goal 5).
package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/t5chargen/benefit"
	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/fame"
	"github.com/philoserf/t5chargen/lifestage"
	"github.com/philoserf/t5chargen/ship"
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

	// Keyed off the label rather than the UWP: chart B's last cell is
	// "Born In Deep Space" and carries none (p. 56, ERRATA I-97), and a
	// homeworld the sheet drops is one the character still has. The zero
	// homeworld labels empty and is still omitted.
	if label := c.Homeworld.Label(); label != "" {
		fmt.Fprintf(&b, "**Homeworld**: %s\n\n", label)
	}

	b.WriteString(ageLines(c))

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

	b.WriteString(benefitsLine(c))
	b.WriteString(automaticsLine(c))

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
// Talent, rank, the Scout's Discoveries and pending Sanity modifier, and
// the Merchant's Ship Shares.
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
		// "A character in the Reserves maintains his or her last held
		// rank as a Reserve Rank ('I'm a Marine Reserve Sergeant')"
		// (p. 67) — which is what the character is once the career ends,
		// so the sheet says so rather than printing the active rank.
		rank := record.RankTitle + " " + record.Rank
		if record.Reserve {
			rank = "Reserve " + rank
		}

		parts = append(parts, rank)
	}

	parts = append(parts, scoutValues(record)...)
	parts = append(parts, functionaryValues(record)...)
	parts = append(parts, craftsmanValues(record)...)

	if record.ShipShares > 0 {
		parts = append(parts, plural(record.ShipShares, "Ship Share"))
	}

	if len(parts) == 0 {
		return ""
	}

	return ", " + strings.Join(parts, ", ")
}

// scoutValues renders what a Scout career carries: the Discoveries it
// earned (chart 05 p. 79) and what its terms will cost Sanity. The
// modifier is a pending reduction rather than a value, because Sanity "is
// not normally indicated in references to a character" and is not
// generated during character generation at all (p. 52).
func scoutValues(record chargen.CareerRecord) []string {
	var parts []string

	if record.Discoveries > 0 {
		parts = append(parts, plural(record.Discoveries, "Discovery"))
	}

	if record.SanityMod != 0 {
		parts = append(parts, fmt.Sprintf("San %+d when generated", record.SanityMod))
	}

	return parts
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
//
// A name the chart already spells with a count — "Pension x2",
// "Retirement x2" — takes the count in front and nothing on the end.
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}

	switch {
	case strings.HasSuffix(noun, "x2"):
		// The chart already spells the count into the name.
	case strings.HasSuffix(noun, "ss"), strings.HasSuffix(noun, "ch"), strings.HasSuffix(noun, "sh"),
		strings.HasSuffix(noun, "x"), strings.HasSuffix(noun, "z"):
		// A sibilant takes -es. Chart M1's "StarPass" — and this must
		// precede the already-plural case, which it also matches.
		noun += "es"
	case strings.HasSuffix(noun, "s"):
		// Already plural on the chart: "Ship Shares".
	case strings.HasSuffix(noun, "y"):
		// "Discovery", and chart M1's "Proxy".
		noun = strings.TrimSuffix(noun, "y") + "ies"
	default:
		noun += "s"
	}

	return strconv.Itoa(n) + " " + noun
}

// statusLine renders fame, wounds, and disabled/dead status.
func statusLine(c chargen.Character) string {
	parts := []string{}

	if c.Fame > 0 {
		// "Fame is noted as Fame-<level>" with a descriptor naming its
		// reach: "A world famous Entertainer has Fame-10" (chart F p. 91).
		parts = append(parts, fmt.Sprintf("Fame %d (%s)", c.Fame, fameDescriptor(c.Fame)))
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
	case chargen.ConsequenceCharacteristicFloored:
		// Delta is what the floor refused, so the transcript shows both
		// the size of the effect and what the character could lose
		// (interpretation I-107).
		return fmt.Sprintf("%s floored at %d, %d refused", c.Characteristic, c.Value, -c.Delta)
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
		return benefitLostText(c)
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

// benefitLostText names which benefit a chart cell could not deliver. The
// payload says which: a characteristic at the p. 68 maximum, a Master
// Skill List group with nothing left to give (chart 01's "New Trade***:
// Any Trade not already held; if all are already held; this benefit is
// lost", p. 75), or neither, which is the p. 78 Major/Minor cell.
func benefitLostText(c *chargen.ConsequenceEvent) string {
	if c.Characteristic != "" {
		return fmt.Sprintf("benefit lost (%s at the characteristic-%d maximum)",
			c.Characteristic, chargen.CharacteristicMax)
	}

	if c.Skill != "" {
		return "benefit lost (every " + c.Skill + " already held)"
	}

	return "benefit lost (no Major/Minor)"
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
		return consequenceDeadText(c)
	case chargen.ConsequenceDiscovery:
		return fmt.Sprintf("Discovery (total %d)", c.Value)
	case chargen.ConsequenceRankSet:
		return "rank " + c.Skill
	case chargen.ConsequenceShipShares:
		return fmt.Sprintf("%s (total %d)", plural(c.Delta, "Ship Share"), c.Value)
	default:
		return consequenceCareerValueText(c)
	}
}

// consequenceDeadText names the death's cause. Injury names the
// characteristic it reduced ("If the Controlling Characteristic is reduced
// to zero or less, the Character is dead", p. 65); aging names none ("The
// second time three characteristics are reduced to 0, the character dies",
// chart A p. 89).
func consequenceDeadText(c *chargen.ConsequenceEvent) string {
	if c.Characteristic == "" {
		return fmt.Sprintf("DEAD at %d (the second extremely major illness)", c.Value)
	}

	return "DEAD (" + c.Characteristic + " reduced to zero)"
}

// consequenceCareerValueText renders the consequence kinds for the values a
// career tracks of its own (chart 03's Fame and Talent, p. 77).
//
//nolint:exhaustive // Deliberately partitioned: earlier kinds are handled upstream.
func consequenceCareerValueText(c *chargen.ConsequenceEvent) string {
	switch c.Kind {
	case chargen.ConsequenceSanityMod:
		return fmt.Sprintf("Sanity %+d when generated", c.Delta)
	case chargen.ConsequenceCareerChanged:
		return "leaves " + c.Career + " for another career"
	case chargen.ConsequenceAssociated:
		return "position associated with the character's " + c.Skill + " career"
	case chargen.ConsequenceReserve:
		return reserveText(c)
	case chargen.ConsequenceSpecialtySet:
		return "specialty " + c.Skill
	case chargen.ConsequenceTalentSet:
		return fmt.Sprintf("Talent = %d", c.Value)
	case chargen.ConsequenceComeback:
		return fmt.Sprintf("Comeback (Fame reset to %d)", c.Value)
	case chargen.ConsequenceMasterpiece:
		// "The Masterpiece can be sold at Cr150,000 plus Cr10,000 per
		// Master Point over 40" (chart 01 p. 75). Whether it is Perfect
		// is a threshold the chart owns, so the line reports the points
		// and leaves the count to the sheet.
		return fmt.Sprintf("Masterpiece created (%d Master Points, worth Cr%d)", c.Value, c.Delta)
	default:
		return consequenceAgingText(c)
	}
}

// consequenceAgingText renders the chart A consequence kinds (p. 89).
//
//nolint:exhaustive // Deliberately partitioned: earlier kinds are handled upstream.
func consequenceAgingText(c *chargen.ConsequenceEvent) string {
	switch c.Kind {
	case chargen.ConsequenceAgingEffect:
		return fmt.Sprintf("aging: %s %+d = %d", c.Characteristic, c.Delta, c.Value)
	case chargen.ConsequenceCharacteristicReset:
		return c.Characteristic + " reduced to zero, reset to 1"
	case chargen.ConsequenceEntitlement:
		return entitlementText(c)
	case chargen.ConsequenceBenefit:
		return benefitText(c)
	case chargen.ConsequenceFameComputed:
		return fameComputedText(c)
	case chargen.ConsequenceFameChange:
		return fmt.Sprintf("Fame %+d = %d", c.Delta, c.Value)
	case chargen.ConsequenceMajorIllness:
		return fmt.Sprintf("major illness (%d characteristics at zero); four weeks recuperating", c.Value)
	case chargen.ConsequenceExtremelyMajorIllness:
		return fmt.Sprintf("extremely major illness (%d at zero); four months recuperating", c.Value)
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
	case chargen.ConsequenceBirthdate:
		return "born " + c.Detail
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
		// "<Undercover Career> Commendation-N" (chart F p. 91).
		return fmt.Sprintf("%s-%d (total %d)", c.Skill, c.Delta, c.Value)
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

// lifeStageName renders the stage's printed name (chart A p. 89). An
// unreadable table degrades to the number rather than failing the sheet:
// the stage is context for the age, not the record.
func lifeStageName(stage int) string {
	table, err := lifestage.Load()
	if err != nil {
		return strconv.Itoa(stage)
	}

	if name := table.Name(stage); name != "" {
		return name
	}

	return strconv.Itoa(stage)
}

// reserveText names the Reserve Rank where the career held one: "A
// character in the Reserves maintains his or her last held rank as a
// Reserve Rank" (p. 67).
func reserveText(c *chargen.ConsequenceEvent) string {
	if c.Skill == "" {
		return "enters the " + c.Career + " Reserves"
	}

	return "enters the Reserves as a Reserve " + c.Skill + " (" + c.Career + ")"
}

// craftsmanValues renders what a Craftsman career carries: the
// Masterpieces it produced, and how many of them are Perfect ("A Perfect
// Masterpiece has 55 or more Master Points", chart 01 p. 75). Chart F
// multiplies both for Fame, and muster out prices them.
func craftsmanValues(record chargen.CareerRecord) []string {
	if record.Masterpieces == 0 {
		return nil
	}

	made := plural(record.Masterpieces, "Masterpiece")
	if record.PerfectMasterpieces == 0 {
		return []string{made}
	}

	return []string{fmt.Sprintf("%s (%d Perfect)", made, record.PerfectMasterpieces)}
}

// functionaryValues renders what a Functionary career carries beyond its
// rank: "The Functionary character must identify with which prior career
// his position is associated" (chart 13 p. 87), which muster out reads.
func functionaryValues(record chargen.CareerRecord) []string {
	if record.AssociatedCareer == "" {
		return nil
	}

	return []string{"associated with " + record.AssociatedCareer}
}

// fameComputedText renders chart F's calculation with its working, as the
// chart's own example does ("Rogue with one Failed Scheme ... has Fame =
// 1 x 3 = 3", p. 91). The sources are worth more than the total: they say
// why a character is known.
func fameComputedText(c *chargen.ConsequenceEvent) string {
	line := fmt.Sprintf("Fame %d (%s)", c.Value, c.Skill)
	if len(c.Mods) == 0 {
		return line
	}

	sources := make([]string, 0, len(c.Mods))
	for _, mod := range c.Mods {
		sources = append(sources, fmt.Sprintf("%s %+d", mod.Name, mod.Value))
	}

	return line + " = " + strings.Join(sources, ", ")
}

// fameDescriptor names a Fame level's reach (chart F p. 91). An unreadable
// table degrades to the bare level rather than failing the sheet.
func fameDescriptor(level int) string {
	table, err := fame.Load()
	if err != nil {
		return strconv.Itoa(level)
	}

	if name := table.Descriptor(level); name != "" {
		return name
	}

	return strconv.Itoa(level)
}

// benefitText renders one muster-out award (chart M1 p. 70): what it was,
// how many, and what it paid.
func benefitText(c *chargen.ConsequenceEvent) string {
	name := c.Skill
	if c.Characteristic != "" {
		name += " (" + c.Characteristic + ")"
	}

	if c.Value > 1 {
		name = fmt.Sprintf("%s x%d", name, c.Value)
	}

	if c.Delta != 0 {
		return fmt.Sprintf("%s, worth Cr%d", name, c.Delta)
	}

	return name
}

// benefitsLine renders what muster out awarded: the money first, then the
// benefits themselves, gathered by kind (chart M1 p. 70).
func benefitsLine(c chargen.Character) string {
	if c.Credits == 0 && len(c.Benefits) == 0 {
		return ""
	}

	var line strings.Builder

	if c.Credits > 0 {
		fmt.Fprintf(&line, "**Credits**: Cr%d\n\n", c.Credits)
	}

	counts := map[string]int{}
	order := make([]string, 0, len(c.Benefits))

	for _, got := range c.Benefits {
		// Money is the credits line above, not a benefit to list again.
		if got.Kind == benefit.Money {
			continue
		}

		name := got.Name
		if got.Detail != "" {
			name += " (" + got.Detail + ")"
		}

		if counts[name] == 0 {
			order = append(order, name)
		}

		counts[name] += max(got.Count, 1)
	}

	parts := make([]string, 0, len(order))
	for _, name := range order {
		parts = append(parts, plural(counts[name], name))
	}

	if len(parts) > 0 {
		fmt.Fprintf(&line, "**Benefits**: %s\n\n", strings.Join(parts, ", "))
	}

	return line.String()
}

// entitlementText renders an annual payment, or the lump taken instead
// (chart M1 p. 70; p. 69).
func entitlementText(c *chargen.ConsequenceEvent) string {
	if c.Characteristic != "" {
		return c.Skill + ", " + c.Characteristic
	}

	return fmt.Sprintf("%s, Cr%d a year from age %d", c.Skill, c.Delta, c.Value)
}

// shipSharesLine renders the shares and what chart S says they reach.
// Shares are never money — Book 1 prices ships in shares and never prices
// a share (p. 90, interpretation I-84) — so this is the only figure there
// is to give.
func shipSharesLine(shares int) string {
	if shares == 0 {
		return ""
	}

	line := fmt.Sprintf("**Ship Shares**: %d", shares)

	if table, err := ship.Load(); err == nil {
		if best, ok := table.Largest(shares); ok {
			line += fmt.Sprintf(", enough for a %d-ton %s (%d)", best.Tons, best.Name, best.Shares)

			if spare := shares - best.Shares; spare > 0 {
				line += fmt.Sprintf(" with %d to spare", spare)
			}
		}
	}

	return line + "\n\n"
}

// ageLines renders the age, the life stage it falls in (chart A p. 89),
// and the birthdate — "Sigg's birthdate is Wonday 044-1075" (p. 263).
func ageLines(c chargen.Character) string {
	line := fmt.Sprintf("**Age**: %d (%s)\n\n", c.Age, lifeStageName(c.LifeStage))

	if c.Birthdate != "" {
		line += fmt.Sprintf("**Born**: %s\n\n", c.Birthdate)
	}

	return line
}

// automaticsLine renders what a character already owns at muster out
// (chart M1's Automatics, p. 70), and the pensions he retires on.
func automaticsLine(c chargen.Character) string {
	var line strings.Builder

	if len(c.Automatics) > 0 {
		fmt.Fprintf(&line, "**Automatics**: %s\n\n", strings.Join(c.Automatics, ", "))
	}

	// Land Grant income is annual and is never money: the grant is
	// retained at muster out, not sold (Book 1 p. 68).
	if c.LandGrantIncome > 0 {
		fmt.Fprintf(&line, "**Land Grants**: Cr%d a year\n\n", c.LandGrantIncome)
	}

	if s := shipSharesLine(c.ShipShares); s != "" {
		line.WriteString(s)
	}

	for _, e := range c.Entitlements {
		fmt.Fprintf(&line, "**%s**: Cr%d a year from age %d\n\n", e.Name, e.AnnualCredits, e.FromAge)
	}

	return line.String()
}
