package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// schoolAt takes Later Education at the nth offer and serves every other
// term, so a test can suspend exactly one term and compare the record
// against the same seed run straight through.
type schoolAt struct {
	program string
	at      int
	seen    int
}

func (d *schoolAt) Choose(c chargen.Choice) (int, error) {
	if c.ID != chargen.ChooseLaterEducation {
		return autoPolicy(c)
	}

	d.seen++
	if d.seen != d.at {
		return 0, nil
	}

	for i, option := range c.Options {
		if option == d.program {
			return i, nil
		}
	}

	return 0, nil
}

func (*schoolAt) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// schoolAlways never serves a term while any program is on offer, which is
// the unbounded case: the offer recurs every term and Apprenticeship has
// no prerequisite, so nothing but the passage of time ends the lifepath.
type schoolAlways struct{}

func (schoolAlways) Choose(c chargen.Choice) (int, error) {
	if c.ID != chargen.ChooseLaterEducation {
		return autoPolicy(c)
	}

	// Index 0 is serving the term. Among the programs, take the first the
	// character actually qualifies for: chart C's rows are all offered
	// now, and reaching for one he falls short of turns the choice into a
	// Prerequisite waiver he will usually lose, which tests the wrong
	// rule.
	for i := 1; i < len(c.Options); i++ {
		if i < len(c.Scores) && c.Scores[i] == 1 {
			return i, nil
		}
	}

	return 0, nil
}

func (schoolAlways) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// laterEducationSeed is pinned: seed 1's Citizen serves enough terms to be
// offered school more than once, so "at the second offer" is reachable.
const laterEducationSeed = 1

// TestLaterEducationSubstitutesTheTerm verifies the p. 59 rule: a
// suspended term costs the term's four years, not the program's duration
// — "substitutes that process for the entire term" (interpretation I-88).
//
// The claim is measured inside one run rather than by comparing against
// the same seed run straight through. Schooling consumes dice, so an
// unsuspended run of the same seed is a different lifepath from the first
// throw onward; its age says nothing about the cost of a term.
func TestLaterEducationSubstitutesTheTerm(t *testing.T) {
	// Trade School is one Pass/Fail year against a four-year term, so a
	// program-duration reading and a whole-term reading differ by three.
	c := generate(t, chargen.Options{
		Seed:    laterEducationSeed,
		Decider: &schoolAt{program: "Trade School", at: 1},
	})

	if got := suspendedTermYears(t, c); got != chargen.TermYears {
		t.Errorf("the suspended term cost %d years, want the term's %d", got, chargen.TermYears)
	}

	last := c.Education[len(c.Education)-1]
	if last.Program != "Trade School" {
		t.Fatalf("the mid-career record is %q, want the program the decider chose", last.Program)
	}
}

// suspendedTermYears sums the years elapsed between the Later Education
// step and the step that follows it, which is the whole cost of the
// suspended term.
func suspendedTermYears(t *testing.T, c chargen.Character) int {
	t.Helper()

	years, counting := 0, false

	for _, event := range c.Events {
		if event.Kind == chargen.EventStep {
			if counting {
				return years
			}

			counting = strings.HasPrefix(event.Step.Name, "Later Education:")

			continue
		}

		if counting && event.Kind == chargen.EventConsequence &&
			event.Consequence.Kind == chargen.ConsequenceYearsElapsed {
			years += event.Consequence.Value
		}
	}

	if !counting {
		t.Fatal("no Later Education step in the transcript")
	}

	return years
}

// servedTerms counts the terms actually resolved, across every career.
func servedTerms(c chargen.Character) int {
	terms := 0
	for _, record := range c.Careers {
		terms += len(record.Terms)
	}

	return terms
}

// TestLaterEducationIsNotATermServed verifies a suspended term is not
// counted as one. Muster out reads len(Terms) for its benefit rolls and
// its pensions, so a term spent at school must leave no TermRecord behind
// or the character is paid for service he did not give.
func TestLaterEducationIsNotATermServed(t *testing.T) {
	suspended := generate(t, chargen.Options{
		Seed:    laterEducationSeed,
		Decider: &schoolAt{program: "Trade School", at: 1},
	})

	for _, record := range suspended.Careers {
		for i, term := range record.Terms {
			if term.Term != i+1 {
				t.Errorf("%s term %d is numbered %d: a suspended term left a gap",
					record.Career, i+1, term.Term)
			}
		}
	}
}

// TestLaterEducationSuspendsResolution verifies the suspended term runs no
// career mechanics. "Suspend career resolution" (p. 59) suspends the
// Continue throw with everything else (interpretation I-90), so the
// transcript must show the education step where the term's step would be,
// with no career step opened for it.
func TestLaterEducationSuspendsResolution(t *testing.T) {
	suspended := generate(t, chargen.Options{
		Seed:    laterEducationSeed,
		Decider: &schoolAt{program: "Trade School", at: 1},
	})

	steps, found := 0, false

	for _, event := range suspended.Events {
		if event.Kind != chargen.EventStep {
			continue
		}

		if strings.HasPrefix(event.Step.Name, "Later Education:") {
			found = true
		}

		if strings.Contains(event.Step.Name, ": Term ") {
			steps++
		}
	}

	if !found {
		t.Fatal("no Later Education step in the transcript")
	}

	if want := servedTerms(suspended); steps != want {
		t.Errorf("%d career term steps for %d terms served: a suspended term opened one", steps, want)
	}

	// The counts above stay consistent even if a suspended term is also
	// served, because then both go up together. What cannot happen if the
	// term is truly suspended is a career term between two consecutive
	// suspensions: a decider that accepts every offer must produce
	// schooling back to back, broken only where an application was
	// refused.
	if !hasBackToBackSchooling(t) {
		t.Error("every suspension was followed by a served term, so the term is not being suspended")
	}
}

// hasBackToBackSchooling reports whether a character who accepts every
// offer ever attends twice with no term served in between.
func hasBackToBackSchooling(t *testing.T) bool {
	t.Helper()

	for seed := range uint64(5) {
		c := generate(t, chargen.Options{Seed: seed, Decider: schoolAlways{}})
		previousWasSchool := false

		for _, event := range c.Events {
			if event.Kind != chargen.EventStep {
				continue
			}

			school := strings.HasPrefix(event.Step.Name, "Later Education:")
			if school && previousWasSchool {
				return true
			}

			if school || strings.Contains(event.Step.Name, ": Term ") {
				previousWasSchool = school
			}
		}
	}

	return false
}

// TestLaterEducationReplays verifies a suspended term survives the replay
// contract. No golden fixture exercises this path — the auto policy
// declines every offer — so TestReplayRoundTrip cannot reach it, and a
// mid-career education that did not replay would be a hole the fixture
// sweep could not see.
func TestLaterEducationReplays(t *testing.T) {
	c := generate(t, chargen.Options{
		Seed:    laterEducationSeed,
		Decider: &schoolAt{program: "Trade School", at: 1},
	})

	if _, err := chargen.Replay(c); err != nil {
		t.Fatalf("a record with a suspended term does not replay: %v", err)
	}
}

// terminationSeeds is wide on purpose. This sweep first ran over five
// seeds, passed, and the interpretation it backs claimed the lifepath
// always terminates — which was false: seed 111 hung outright, because a
// character who died at school was never checked for death and aging stops
// checking once Dead is set. Five seeds was not a sweep, it was an
// anecdote.
const terminationSeeds = 150

// TestLaterEducationTerminates verifies the unbounded case ends. The offer
// recurs at the beginning of every term and Apprenticeship has no
// prerequisite, so a character can accept forever; a suspended term throws
// no Continue, so nothing inside the career stops him.
//
// Two things do, and the test requires one of them. Aging kills (chart A
// p. 89), because the years pass whether they are spent serving or
// studying. Or an application is refused, which suspends nothing
// ("if accepted", p. 59, interpretation I-89) — the term is served after
// all and throws its Continue. A run that ended alive having served no
// term would mean neither holds and the lifepath can run forever.
//
// A regression here hangs the suite rather than failing it, which is why
// it is pinned.
func TestLaterEducationTerminates(t *testing.T) {
	for seed := range uint64(terminationSeeds) {
		c := generate(t, chargen.Options{Seed: seed, Decider: schoolAlways{}})

		if len(c.Education) < 2 {
			t.Errorf("seed %d: %d education records, want the mid-career path exercised", seed, len(c.Education))
		}

		if !c.Dead && servedTerms(c) == 0 {
			t.Errorf("seed %d: ended alive at age %d having served no term, so nothing ended the lifepath",
				seed, c.Age)
		}
	}
}

// TestLaterEducationDeathEndsTheLifepath verifies that a character killed
// by aging while at school stops being resolved. The years pass at school
// as they do in service (chart A p. 89), and a refused application costs a
// year of its own (interpretation I-89), so both entry points can leave a
// corpse where the term loop expects a survivor.
//
// Two things go wrong without the check, and the sweep catches both. A
// dead character served the term he had applied out of — earning skills, a
// TermRecord and the muster-out benefit rolls that count it. And with a
// decider that accepts every offer the loop never ended at all: Dead stops
// ageEffects from checking, so once the aging that has to end the lifepath
// has fired, nothing can fire again. Seed 111 hung; seed 2 resolved a step
// past the death.
func TestLaterEducationDeathEndsTheLifepath(t *testing.T) {
	for seed := range uint64(150) {
		c := generate(t, chargen.Options{Seed: seed, Decider: schoolAlways{}})

		dead := false

		for _, event := range c.Events {
			if event.Kind == chargen.EventConsequence &&
				event.Consequence.Kind == chargen.ConsequenceDead {
				dead = true

				continue
			}

			if !dead || event.Kind != chargen.EventStep {
				continue
			}

			if strings.HasPrefix(event.Step.Name, "Later Education:") ||
				strings.Contains(event.Step.Name, ": Term ") {
				t.Fatalf("seed %d: %q resolved after the character died", seed, event.Step.Name)
			}
		}
	}
}

// TestLaterEducationIsNotACareerReceipt verifies that a skill earned at
// school mid-career does not demote a later career award of the same skill
// to Skill-1. "Receipts" under the Job/Hobby first-receipt rule are career
// receipts, and levels held from education are not (interpretation I-2,
// ERRATA.md) — the same reason a homeworld grant does not demote. The
// baseline that rule reads is the levels held at career entry, so
// schooling taken after entry has to be credited to it or going to school
// leaves the character strictly worse off.
//
// Seed 5 is pinned because it is the case: the character apprentices in
// Chef for Chef-4, then a later Citizen term selects Chef as its Hobby.
// "Skill-2 (later receipts are Skill-1)" (p. 78) must award the full +2.
func TestLaterEducationIsNotACareerReceipt(t *testing.T) {
	const (
		seed        = 5
		schooled    = "Chef"
		hobbyLevels = 2
	)

	c := generate(t, chargen.Options{Seed: seed, Decider: &apprenticeIn{skill: schooled}})

	got, ok := hobbyAward(c, schooled)
	if !ok {
		t.Fatalf("seed %d no longer schools %s and then takes it as a Hobby", seed, schooled)
	}

	if got != hobbyLevels {
		t.Errorf("the Hobby award of a skill held from school was %+d, want %+d: schooling was read as a career receipt",
			got, hobbyLevels)
	}
}

// hobbyAward reports the levels awarded by the Hobby determination that
// selected the named skill.
func hobbyAward(c chargen.Character, name string) (int, bool) {
	determined := false

	for _, event := range c.Events {
		if event.Kind != chargen.EventConsequence {
			continue
		}

		consequence := event.Consequence

		if consequence.Kind == chargen.ConsequenceHobbySet && consequence.Skill == name {
			determined = true

			continue
		}

		if determined && consequence.Kind == chargen.ConsequenceSkillAwarded &&
			consequence.Skill == name {
			return consequence.Delta, true
		}
	}

	return 0, false
}

// apprenticeIn takes the first Later Education offer as an Apprenticeship
// in one named skill, serves every later term, and takes the same skill as
// its Hobby wherever the ladder offers one.
type apprenticeIn struct {
	skill string
	taken bool
}

func (d *apprenticeIn) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseLaterEducation {
		if d.taken {
			return 0, nil
		}

		index := indexOf(c.Options, "Apprenticeship")
		d.taken = index > 0

		return max(index, 0), nil
	}

	if c.ID == chargen.ChooseSkill || c.ID == chargen.ChooseHobby {
		if index := indexOf(c.Options, d.skill); index >= 0 {
			return index, nil
		}
	}

	return autoPolicy(c)
}

func (*apprenticeIn) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }
