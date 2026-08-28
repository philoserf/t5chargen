package chargen_test

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/skill"
)

// autoPolicy answers a choice with the default policy, for the test fakes
// that override one choice point and defer the rest. The policy is total
// and never errors, but the Decider contract lets it, so the error is
// threaded rather than dropped.
func autoPolicy(c chargen.Choice) (int, error) {
	index, err := careerOnly{}.Choose(c)
	if err != nil {
		return 0, fmt.Errorf("auto policy: %w", err)
	}

	return index, nil
}

// generate builds a character or fails the test, defaulting to the auto
// policy when no Decider is set.
// generate builds a character for a test, defaulting to the auto policy
// with schooling declined.
//
// Declined because almost nothing here is about education. These tests pin
// seeds to a condition — a career disabled at a term, a benefit rerolled,
// a pension replaced — and a term given over to Later Education is a term
// not served (I-90), so a policy that sends a character back to school
// moves every one of them without saying anything about what they assert.
// Seventeen broke that way at policy 0.20.0.
//
// The goldens pass chargen.DefaultPolicy explicitly: exercising the policy
// end to end is their job, and it is the only place it belongs.
func generate(t *testing.T, opts chargen.Options) chargen.Character {
	t.Helper()

	if opts.Decider == nil {
		opts.Decider = careerOnly{}
	}

	c, err := chargen.Generate(opts)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

// generateIfOpen is generate for a sweep that forces a career the
// character may not be eligible for, reporting whether one was produced.
//
// A closed career is an outcome rather than a fault: chart 11's "if Soc B+"
// and chart 01's "if TWO skill-6 and Craftsman-1" are prerequisites, and
// p. 65 places them "before a character may attempt to Begin" (I-28), so
// forcing one on a character who falls short is a request the rules deny.
// Any other error still fails the test.
func generateIfOpen(t *testing.T, opts chargen.Options) (chargen.Character, bool) {
	t.Helper()

	if opts.Decider == nil {
		opts.Decider = careerOnly{}
	}

	c, err := chargen.Generate(opts)
	if errors.Is(err, chargen.ErrCareerUnavailable) {
		return chargen.Character{}, false
	}

	if err != nil {
		t.Fatal(err)
	}

	return c, true
}

// TestCitizenInvariants checks rule invariants across many seeds: age
// accounting (18 + 4 per term, pp. 59, 66), the Skill-15 cap (p. 134), CC
// membership and rotation in windows of four (p. 65), and Job/Hobby
// consistency (chart 04, p. 78).
func TestCitizenInvariants(t *testing.T) {
	for seed := range uint64(40) {
		c := generate(t, chargen.Options{Seed: seed})

		if len(c.Careers) != 1 || c.Careers[0].Career != "Citizen" {
			t.Fatalf("seed %d: careers = %+v", seed, c.Careers)
		}

		record := c.Careers[0]
		// Exact age accounting (18 + education + terms) is asserted by
		// checkEducationAge; here just the career's contribution.
		if minimum := chargen.StartAge + chargen.TermYears*len(record.Terms); c.Age < minimum {
			t.Errorf("seed %d: age %d below %d for %d terms", seed, c.Age, minimum, len(record.Terms))
		}

		checkSkills(t, seed, c)
		checkTerms(t, seed, record)
	}
}

// checkSkills asserts every skill is within 1..SkillMax.
func checkSkills(t *testing.T, seed uint64, c chargen.Character) {
	t.Helper()

	for _, held := range c.Skills {
		// Level 0 is a real state for a container skill received once
		// or twice: "he has the Knowledges but only Skill-0" (p. 134).
		// The floor is 1 for everything else, and for a container the
		// receipts are what say it was earned.
		floor := 1
		if held.Receipts > 0 {
			floor = 0
		}

		ceiling := chargen.SkillMax
		if entry, ok := skill.Lookup(held.Name); ok && entry.Kind == skill.KindKnowledge {
			ceiling = chargen.KnowledgeMax
		}

		if held.Level < floor || held.Level > ceiling {
			t.Errorf("seed %d: skill %s at level %d outside %d-%d",
				seed, held.Name, held.Level, floor, ceiling)
		}
	}
}

// checkTerms asserts term numbering, CC membership, CC rotation per window
// of four terms, and that only the final term fails Continue.
func checkTerms(t *testing.T, seed uint64, record chargen.CareerRecord) {
	t.Helper()

	valid := map[string]bool{"Str": true, "Dex": true, "End": true, "Int": true}

	for i, term := range record.Terms {
		if term.Term != i+1 {
			t.Errorf("seed %d: term %d numbered %d", seed, i+1, term.Term)
		}

		if !valid[term.ControllingCharacteristic] {
			t.Errorf("seed %d: term %d CC %q", seed, term.Term, term.ControllingCharacteristic)
		}

		if term.Continued != (i != len(record.Terms)-1) {
			t.Errorf("seed %d: term %d continued=%v at position %d/%d",
				seed, term.Term, term.Continued, i+1, len(record.Terms))
		}
	}

	for start := 0; start < len(record.Terms); start += 4 {
		window := map[string]bool{}

		for _, term := range record.Terms[start:min(start+4, len(record.Terms))] {
			if window[term.ControllingCharacteristic] {
				t.Errorf("seed %d: CC %q repeated within rotation window starting at term %d",
					seed, term.ControllingCharacteristic, start+1)
			}

			window[term.ControllingCharacteristic] = true
		}
	}
}

// TestCitizenJobHobbyLadder verifies the chart 04 ladder on a range of
// seeds: the Hobby is only ever set after the Job, is never the Job
// (ERRATA I-3), and under the policy is the first-listed table E entry
// excluding the Job. A record with successes but no Hobby is legitimate
// (the ERRATA I-1 No Skill retry can consume a success), so the ladder is
// asserted on what is set, not on the success count alone.
func TestCitizenJobHobbyLadder(t *testing.T) {
	for seed := range uint64(60) {
		checkLadder(t, seed, generate(t, chargen.Options{Seed: seed}).Careers[0])
	}
}

// checkLadder asserts one record's Job/Hobby state.
func checkLadder(t *testing.T, seed uint64, record chargen.CareerRecord) {
	t.Helper()

	successes := 0

	for _, term := range record.Terms {
		if term.Success {
			successes++
		}
	}

	if successes == 0 && (record.Job != "" || record.Hobby != "") {
		t.Errorf("seed %d: no successes but job %q hobby %q", seed, record.Job, record.Hobby)
	}

	if record.Hobby != "" {
		checkHobby(t, seed, record)
	}
}

// checkHobby asserts a set Hobby: distinct from the Job (I-3) and the
// policy's first-listed pick.
func checkHobby(t *testing.T, seed uint64, record chargen.CareerRecord) {
	t.Helper()

	if record.Job == "" || record.Hobby == record.Job {
		t.Errorf("seed %d: hobby %q with job %q", seed, record.Hobby, record.Job)
	}

	want := "ACV"
	if record.Job == "ACV" {
		want = "Comms" // next first-listed (A1 B1 C2) once the Job is excluded (I-3)
	}

	if record.Hobby != want {
		t.Errorf("seed %d: policy hobby = %q, want first-listed %q", seed, record.Hobby, want)
	}
}

// TestRegistryMatchesAvailable verifies every career.Available name has
// registered mechanics and vice versa — the milestone-3 failure mode is
// adding data without mechanics or mechanics without data.
func TestRegistryMatchesAvailable(t *testing.T) {
	for _, name := range career.Available() {
		_, err := chargen.Generate(chargen.Options{
			Seed: 1, Career: name, Decider: chargen.DefaultPolicy{},
		})

		// Craftsman and Functionary cannot open a lifepath (p. 75,
		// p. 87), so a refusal is the right answer and not a missing
		// registration. Anything else must generate.
		if errors.Is(err, chargen.ErrCareerUnavailable) {
			continue
		}

		if err != nil {
			t.Errorf("available career %q does not generate: %v", name, err)
		}
	}

	registered := chargen.RegistryNames()

	available := slices.Clone(career.Available())
	slices.Sort(available)

	if !slices.Equal(registered, available) {
		t.Errorf("registry %v != career.Available %v", registered, available)
	}

	defNames, err := chargen.RegistryDefinitionNames()
	if err != nil {
		t.Fatal(err)
	}

	for key, name := range defNames {
		if key != name {
			t.Errorf("registry key %q != definition name %q", key, name)
		}
	}
}

// TestCitizenEventIntegrity verifies every consequence references an
// earlier throw or choice event (docs/PRD.md FR10).
func TestCitizenEventIntegrity(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 1})

	kinds := map[int]chargen.EventKind{}
	for _, event := range c.Events {
		kinds[event.Seq] = event.Kind
	}

	for _, event := range c.Events {
		if event.Kind != chargen.EventConsequence {
			continue
		}

		cause := event.Consequence.Cause
		if cause <= 0 || cause >= event.Seq {
			t.Errorf("event %d: cause %d is not an earlier event", event.Seq, cause)
		}

		// A step is a legitimate cause where accumulated state, not
		// dice, produced the consequence: FR10 was amended to say so
		// (interpretation I-87), and the Career and World Knowledges
		// are the plainest case — p. 134 awards them by arithmetic over
		// the finished record and names no die.
		if k := kinds[cause]; k != chargen.EventThrow && k != chargen.EventChoice &&
			k != chargen.EventStep {
			t.Errorf("event %d: cause %d is a %q, want throw, choice or step", event.Seq, cause, k)
		}
	}
}

// mandatoryContinueSeed rolls a natural 2 on a Continue throw.
// Repinned when aging landed: aging consumes stream, so the seed that
// used to reach one no longer does.
const mandatoryContinueSeed = 5

// TestCitizenMandatoryContinue pins a seed whose career includes a
// Continue roll of exactly 2: "If the Continue roll is 2 exactly, the
// character is required to Continue" (p. 66). The term with the mandatory
// continue must not be the career's last.
func TestCitizenMandatoryContinue(t *testing.T) {
	c := generate(t, chargen.Options{Seed: mandatoryContinueSeed})

	found := false

	for _, event := range c.Events {
		if event.Kind == chargen.EventConsequence && event.Consequence.Kind == chargen.ConsequenceMandatoryContinue {
			found = true

			cause := event.Consequence.Cause
			for _, e := range c.Events {
				if e.Seq == cause && (e.Throw == nil || e.Throw.Total != 2) {
					t.Errorf("mandatory continue caused by %+v, want a throw totaling 2", e)
				}
			}
		}
	}

	if !found {
		t.Fatalf("seed %d no longer exercises mandatory continue; find and pin another seed",
			mandatoryContinueSeed)
	}
}

// TestEventStreamAccounting verifies the FR10 stream invariant: every
// face in every throw event, concatenated in event order, reproduces the
// seeded stream exactly — no roll is consumed without being logged and
// none is logged without being consumed.
func TestEventStreamAccounting(t *testing.T) {
	for _, opts := range []chargen.Options{
		{Seed: 1},
		{Seed: 6, Career: "Scout"},
		{Seed: 11, Career: "Scout", Decider: braveryDecider{}},
	} {
		c := generate(t, opts)

		roller := dice.New(opts.Seed)

		position := 0

		for _, event := range c.Events {
			if event.Kind != chargen.EventThrow {
				continue
			}

			for _, face := range event.Throw.Dice {
				if got := roller.Roll(1).Total; got != face {
					t.Fatalf("seed %d: event %d records face %d at stream position %d, stream has %d",
						opts.Seed, event.Seq, face, position, got)
				}

				position++
			}
		}
	}
}

// TestGenerateDeterminism verifies the same options reproduce the identical
// record (docs/PRD.md goal 3).
func TestGenerateDeterminism(t *testing.T) {
	a := generate(t, chargen.Options{Seed: 17, Name: "X"})
	b := generate(t, chargen.Options{Seed: 17, Name: "X"})

	if !reflect.DeepEqual(a, b) {
		t.Error("same options produced different records")
	}
}

// TestGenerateForcedCareer verifies the forced-career path and the
// unknown-career error.
func TestGenerateForcedCareer(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 3, Career: "Citizen"})
	if c.Careers[0].Career != "Citizen" {
		t.Errorf("forced career = %+v", c.Careers)
	}

	// A name that is not a career at all.
	_, err := chargen.Generate(chargen.Options{Seed: 3, Career: "Cook", Decider: chargen.DefaultPolicy{}})
	if !errors.Is(err, chargen.ErrUnknownCareer) {
		t.Errorf("unknown forced career error = %v, want ErrUnknownCareer", err)
	}

	// A career that exists but cannot open a lifepath. Chart 01's entry
	// is automatic only "if TWO skill-6 and Craftsman-1" (p. 75), which a
	// character leaving education does not have — a different refusal
	// from an unknown name, and worth telling apart.
	_, err = chargen.Generate(chargen.Options{Seed: 3, Career: "Craftsman", Decider: chargen.DefaultPolicy{}})
	if !errors.Is(err, chargen.ErrCareerUnavailable) {
		t.Errorf("unavailable forced career error = %v, want ErrCareerUnavailable", err)
	}
}

// playerDecider is a well-behaved non-policy Decider for provenance tests.
type playerDecider struct{}

func (playerDecider) Choose(chargen.Choice) (int, error) { return 0, nil }
func (playerDecider) Kind() chargen.DeciderKind          { return chargen.DeciderPlayer }

// badDecider always answers out of range.
type badDecider struct{}

func (badDecider) Choose(chargen.Choice) (int, error) { return 99, nil }
func (badDecider) Kind() chargen.DeciderKind          { return chargen.DeciderPlayer }

// errRefused is the sentinel refusingDecider returns; the point of the
// test is that it survives to the caller intact.
var errRefused = errors.New("refused")

// refusingDecider declines the first choice it is offered, standing in for
// an abandoned interactive session or a replay whose recorded choice no
// longer matches the one presented.
type refusingDecider struct{}

func (refusingDecider) Choose(chargen.Choice) (int, error) { return 0, errRefused }
func (refusingDecider) Kind() chargen.DeciderKind          { return chargen.DeciderPlayer }

// personalDecider always selects the Personal skill column and otherwise
// answers first-listed, to exercise characteristic increases.
type personalDecider struct{}

func (personalDecider) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseSkillColumn {
		return slices.Index(c.Options, "Personal"), nil
	}

	return 0, nil
}

func (personalDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestCharacteristicMaximum verifies "Characteristics for Humans cannot
// exceed 15. If a benefit elevates a characteristic above 15, that benefit
// is lost." (p. 68) under a decider that always takes Personal-column
// increases.
func TestCharacteristicMaximum(t *testing.T) {
	capped := false

	for seed := range uint64(40) {
		c := generate(t, chargen.Options{Seed: seed, Decider: personalDecider{}})

		ch := c.Characteristics
		for _, value := range []int{ch.Str, ch.Dex, ch.End, ch.Int, ch.Edu, ch.Soc} {
			if value > chargen.CharacteristicMax {
				t.Errorf("seed %d: characteristic %d exceeds %d", seed, value, chargen.CharacteristicMax)
			}
		}

		for _, event := range c.Events {
			if event.Kind == chargen.EventConsequence &&
				event.Consequence.Kind == chargen.ConsequenceBenefitLost &&
				event.Consequence.Characteristic != "" {
				capped = true
			}
		}
	}

	if !capped {
		t.Error("no seed exercised the characteristic maximum; extend the sweep")
	}
}

// TestGenerateDeciderContract verifies the Decider requirements: nil
// errors, an out-of-range answer errors instead of being silently
// repaired, a refusal reaches the caller as its own error naming the
// choice point, and policy_version attests "none" for non-policy
// deciders.
func TestGenerateDeciderContract(t *testing.T) {
	if _, err := chargen.Generate(chargen.Options{Seed: 1}); err == nil {
		t.Error("nil Decider did not error")
	}

	if _, err := chargen.Generate(chargen.Options{Seed: 1, Decider: badDecider{}}); err == nil {
		t.Error("out-of-range Decider answer did not error")
	}

	// A refusal must reach the caller as its own error, not be flattened
	// into the out-of-range one: interactive mode has to tell an abandoned
	// session from a decider that answered wrongly, and replay has to
	// report its own divergence.
	_, err := chargen.Generate(chargen.Options{Seed: 1, Decider: refusingDecider{}})
	if !errors.Is(err, errRefused) {
		t.Fatalf("refused choice = %v, want it to wrap errRefused", err)
	}

	if !strings.Contains(err.Error(), string(chargen.ChooseHomeworld)) {
		t.Errorf("refusal %q does not name the choice point it refused", err)
	}

	c := generate(t, chargen.Options{Seed: 1, Decider: playerDecider{}})
	if c.PolicyVersion != "none" {
		t.Errorf("policy_version with player decider = %q, want %q", c.PolicyVersion, "none")
	}

	// Explicit, because generate defaults to the schooling-free policy
	// and this is the one assertion in the file that is about which
	// decision table governed the run.
	auto := generate(t, chargen.Options{Seed: 1, Decider: chargen.DefaultPolicy{}})
	if auto.PolicyVersion != chargen.PolicyVersion {
		t.Errorf("policy_version with default policy = %q, want %q", auto.PolicyVersion, chargen.PolicyVersion)
	}
}

// TestNoSkillCellRetriesOnTheNextSuccess pins interpretation I-1: a Job
// determination landing on table E's "No Skill" cell (chart 04, p. 78)
// leaves the Job undetermined, and the next success retries it rather
// than moving on to the Hobby.
//
// Both halves are pinned, because the retry is structural — the dispatch
// gates on an empty Job — and a change that let the ladder advance would
// silently hand the character a Hobby he has no Job for. Seed 169 is the
// case where no further success arrives: the Job stays unset, and no
// Hobby is determined ahead of it.
func TestNoSkillCellRetriesOnTheNextSuccess(t *testing.T) {
	t.Run("the next success retries the Job", func(t *testing.T) {
		c := generate(t, chargen.Options{Seed: 114, Career: "Citizen"})

		kinds := jobLadder(c)
		want := []string{"job_undetermined", "job_set", "hobby_set"}

		if !slices.Equal(kinds, want) {
			t.Errorf("ladder = %v, want %v", kinds, want)
		}

		if got := c.Careers[0].Job; got != "Launcher" {
			t.Errorf("Job = %q, want %q", got, "Launcher")
		}
	})

	t.Run("with no further success the Job stays undetermined", func(t *testing.T) {
		c := generate(t, chargen.Options{Seed: 169, Career: "Citizen"})

		if kinds := jobLadder(c); !slices.Equal(kinds, []string{"job_undetermined"}) {
			t.Errorf("ladder = %v, want only job_undetermined", kinds)
		}

		if got := c.Careers[0]; got.Job != "" || got.Hobby != "" {
			t.Errorf("Job = %q, Hobby = %q, want both empty", got.Job, got.Hobby)
		}
	})
}

// jobLadderKinds are the consequences the Job and Hobby ladder emits.
var jobLadderKinds = []chargen.ConsequenceKind{
	chargen.ConsequenceJobUndetermined,
	chargen.ConsequenceJobSet,
	chargen.ConsequenceHobbySet,
}

// jobLadder lists the Job and Hobby consequences a record carries, in the
// order the log records them.
func jobLadder(c chargen.Character) []string {
	var kinds []string

	for _, e := range c.Events {
		if e.Consequence == nil {
			continue
		}

		if slices.Contains(jobLadderKinds, e.Consequence.Kind) {
			kinds = append(kinds, string(e.Consequence.Kind))
		}
	}

	return kinds
}
