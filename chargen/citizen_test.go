package chargen_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// generate builds a character or fails the test, defaulting to the auto
// policy when no Decider is set.
func generate(t *testing.T, opts chargen.Options) chargen.Character {
	t.Helper()

	if opts.Decider == nil {
		opts.Decider = chargen.DefaultPolicy{}
	}

	c, err := chargen.Generate(opts)
	if err != nil {
		t.Fatal(err)
	}

	return c
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

	for _, skill := range c.Skills {
		if skill.Level < 1 || skill.Level > chargen.SkillMax {
			t.Errorf("seed %d: skill %s at level %d outside 1-%d", seed, skill.Name, skill.Level, chargen.SkillMax)
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
		if _, err := chargen.Generate(chargen.Options{
			Seed: 1, Career: name, Decider: chargen.DefaultPolicy{},
		}); err != nil {
			t.Errorf("available career %q does not generate: %v", name, err)
		}
	}

	registered := chargen.RegistryNames()
	slices.Sort(registered)

	available := slices.Clone(career.Available())
	slices.Sort(available)

	if !slices.Equal(registered, available) {
		t.Errorf("registry %v != career.Available %v", registered, available)
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

		if k := kinds[cause]; k != chargen.EventThrow && k != chargen.EventChoice {
			t.Errorf("event %d: cause %d is a %q, want throw or choice", event.Seq, cause, k)
		}
	}
}

// TestCitizenMandatoryContinue pins a seed whose career includes a
// Continue roll of exactly 2: "If the Continue roll is 2 exactly, the
// character is required to Continue" (p. 66). The term with the mandatory
// continue must not be the career's last.
func TestCitizenMandatoryContinue(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 3})

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
		t.Fatal("seed 3 no longer exercises mandatory continue; find and pin another seed")
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

	_, err := chargen.Generate(chargen.Options{Seed: 3, Career: "Noble", Decider: chargen.DefaultPolicy{}})
	if !errors.Is(err, chargen.ErrUnknownCareer) {
		t.Errorf("unknown forced career error = %v, want ErrUnknownCareer", err)
	}
}

// playerDecider is a well-behaved non-policy Decider for provenance tests.
type playerDecider struct{}

func (playerDecider) Choose(chargen.Choice) int { return 0 }
func (playerDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// badDecider always answers out of range.
type badDecider struct{}

func (badDecider) Choose(chargen.Choice) int { return 99 }
func (badDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// personalDecider always selects the Personal skill column and otherwise
// answers first-listed, to exercise characteristic increases.
type personalDecider struct{}

func (personalDecider) Choose(c chargen.Choice) int {
	if c.ID == chargen.ChooseSkillColumn {
		return slices.Index(c.Options, "Personal")
	}

	return 0
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
// repaired, and policy_version attests "none" for non-policy deciders.
func TestGenerateDeciderContract(t *testing.T) {
	if _, err := chargen.Generate(chargen.Options{Seed: 1}); err == nil {
		t.Error("nil Decider did not error")
	}

	if _, err := chargen.Generate(chargen.Options{Seed: 1, Decider: badDecider{}}); err == nil {
		t.Error("out-of-range Decider answer did not error")
	}

	c := generate(t, chargen.Options{Seed: 1, Decider: playerDecider{}})
	if c.PolicyVersion != "none" {
		t.Errorf("policy_version with player decider = %q, want %q", c.PolicyVersion, "none")
	}

	auto := generate(t, chargen.Options{Seed: 1})
	if auto.PolicyVersion != chargen.PolicyVersion {
		t.Errorf("policy_version with default policy = %q, want %q", auto.PolicyVersion, chargen.PolicyVersion)
	}
}
