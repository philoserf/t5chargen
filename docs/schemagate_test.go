package docs_test

// The gate on docs/character.schema.json: that it admits every record the
// engine writes, that it refuses records the engine could not have
// written, and that the checker enforcing it is not vacuous.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// schemaPath is the contract docs/PRD.md's JSON conventions require:
// "Full schema (with minimal and complete examples) lives in the tool
// repo, versioned by schema_version".
const schemaPath = "character.schema.json"

// TestEveryFixtureValidates verifies the schema describes what the engine
// actually writes. The fixtures are its output, byte for byte, across all
// thirteen careers and both deciders.
func TestEveryFixtureValidates(t *testing.T) {
	records, err := filepath.Glob(filepath.Join("..", "chargen", "testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	if len(records) == 0 {
		t.Fatal("no fixtures found; this gate would be asserting nothing")
	}

	for _, record := range records {
		t.Run(filepath.Base(record), func(t *testing.T) {
			problems, err := validate(schemaPath, record)
			if err != nil {
				t.Fatal(err)
			}

			for _, problem := range problems {
				t.Error(problem)
			}
		})
	}
}

// TestExamplesValidate verifies the two examples the PRD asks the schema to
// ship with. The minimal one is the smallest record the schema admits; the
// complete one is a real generated record rather than a fabricated maximum,
// which is why the coverage of the schema's properties is checked
// separately rather than assumed of it.
func TestExamplesValidate(t *testing.T) {
	for _, example := range []string{"character.minimal.json", "character.complete.json"} {
		t.Run(example, func(t *testing.T) {
			problems, err := validate(schemaPath, example)
			if err != nil {
				t.Fatal(err)
			}

			for _, problem := range problems {
				t.Error(problem)
			}
		})
	}
}

// brokenRecords are the ways a record can be wrong that the schema claims
// to catch. Each is the minimal example with one thing done to it, and
// each must fail — for a reason naming what was done.
//
// This is the list that makes the gate mean something. A checker that
// silently passed everything would satisfy every other test in this file.
var brokenRecords = map[string]struct {
	blame  string
	damage func(map[string]any)
}{
	"a required field is missing": {
		blame:  "upp is required",
		damage: func(r map[string]any) { delete(r, "upp") },
	},
	"an unknown field is present": {
		blame:  "not a property",
		damage: func(r map[string]any) { r["favourite_colour"] = "blue" },
	},
	"a string where an integer belongs": {
		blame:  "want integer",
		damage: func(r map[string]any) { r["age"] = "thirty" },
	},
	"a fractional integer": {
		blame:  "want integer",
		damage: func(r map[string]any) { r["age"] = 30.5 },
	},
	"a negative seed": {
		blame:  "at least",
		damage: func(r map[string]any) { nested(r, "rng")["seed"] = -1.0 },
	},
	"an event kind outside the vocabulary": {
		blame:  "is not one of",
		damage: func(r map[string]any) { firstEvent(r)["kind"] = "rumination" },
	},
	"an event carrying the wrong payload": {
		blame: "forbids",
		damage: func(r map[string]any) {
			firstEvent(r)["throw"] = map[string]any{
				"expr": "2D", "dice": []any{1.0, 2.0}, "total": 3.0, "cite": "somewhere",
			}
		},
	},
	"an event carrying no payload at all": {
		blame:  "required and absent",
		damage: func(r map[string]any) { delete(firstEvent(r), "step") },
	},
	"a nested required field is missing": {
		blame:  "algorithm is required",
		damage: func(r map[string]any) { delete(nested(r, "rng"), "algorithm") },
	},
	"a nested unknown field": {
		blame:  "not a property",
		damage: func(r map[string]any) { nested(r, "characteristics")["luck"] = 12.0 },
	},
}

// nested reaches into an object property, which every damage function
// knows is there because the minimal example carries it.
func nested(record map[string]any, name string) map[string]any {
	object, _ := record[name].(map[string]any)

	return object
}

// firstEvent reaches the record's first event.
func firstEvent(record map[string]any) map[string]any {
	events, _ := record["events"].([]any)

	event, _ := events[0].(map[string]any)

	return event
}

// TestTheCheckerCatchesWhatItClaimsTo runs the broken records. Each must
// be refused, and refused for the stated reason rather than incidentally.
func TestTheCheckerCatchesWhatItClaimsTo(t *testing.T) {
	for name, tc := range brokenRecords {
		t.Run(name, func(t *testing.T) {
			record := readRecord(t, "character.minimal.json")
			tc.damage(record)

			problems := checkRecord(t, record)
			if len(problems) == 0 {
				t.Fatalf("%s was accepted; the schema claims to refuse it", name)
			}

			if !strings.Contains(strings.Join(problems, "\n"), tc.blame) {
				t.Errorf("refused for the wrong reason.\nwant a problem mentioning %q\ngot:\n  %s",
					tc.blame, strings.Join(problems, "\n  "))
			}
		})
	}
}

// TestTheMinimalExampleIsMinimal verifies the minimal example earns its
// name: removing any one of its top-level fields must break it. A minimal
// example carrying something optional would misdocument what a record
// needs.
func TestTheMinimalExampleIsMinimal(t *testing.T) {
	base := readRecord(t, "character.minimal.json")

	for field := range base {
		record := readRecord(t, "character.minimal.json")
		delete(record, field)

		if problems := checkRecord(t, record); len(problems) == 0 {
			t.Errorf("the minimal example still validates without %q, so %q is not required", field, field)
		}
	}
}

// readRecord loads a record as a plain document.
func readRecord(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}

	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}

	return record
}

// checkRecord validates an in-memory record.
//
// The record goes back through JSON first. A damaged record is built with
// Go values, and Go has integers where parsed JSON has only float64 — so a
// case written as -1 would reach the checker as a type it can never meet
// in a file, and be refused for the wrong reason. Round-tripping puts the
// damage in the value space the checker actually validates.
func checkRecord(t *testing.T, record map[string]any) []string {
	t.Helper()

	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}

	var parsed any
	if err := json.Unmarshal(encoded, &parsed); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Clean(schemaPath))
	if err != nil {
		t.Fatal(err)
	}

	c, err := newChecker(data)
	if err != nil {
		t.Fatal(err)
	}

	return c.check(c.root, parsed, "record")
}

// TestEverySchemaPropertyIsExercised is the dead-data gate applied to the
// schema. A property nothing ever produces is either a field the engine no
// longer writes or a shape nobody has checked, and both are worth knowing
// about; the schema is a claim about real records, not a wish list.
//
// The complete example is a real generated record rather than a fabricated
// maximum, so it does not reach every optional field on its own. What is
// unreached is listed here with a reason, the same bargain the chart-data
// gate strikes.
func TestEverySchemaPropertyIsExercised(t *testing.T) {
	seen := map[string]bool{}

	records, err := filepath.Glob(filepath.Join("..", "chargen", "testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range append(records, "character.minimal.json", "character.complete.json") {
		var doc any
		if data, err := os.ReadFile(filepath.Clean(path)); err == nil {
			_ = json.Unmarshal(data, &doc)
			collectKeys(doc, seen)
		}
	}

	for _, property := range schemaProperties(t) {
		if seen[property] || unexercised[property] != "" {
			continue
		}

		t.Errorf("the schema describes %q and no fixture or example carries it; "+
			"produce one, or say why not in unexercised", property)
	}

	for property, why := range unexercised {
		if seen[property] {
			t.Errorf("%q is exercised now (%s); remove it from unexercised", property, why)
		}
	}
}

// unexercised are schema properties no fixture or example carries, and
// why. Every entry is a rule the engine implements down a path the pinned
// seeds do not reach — not a field that does nothing.
var unexercised = map[string]string{
	"errata": "the applied ERRATA.md deviations; no rule records one against a character yet, " +
		"so the list is always absent",
	"deep_space":       "chart B's cell 6 6 (I-97); no fixture rolls its homeworld, and none is assigned deep space",
	"rolled_homeworld": "--homeworld random; every fixture takes the assigned world, so the input is false and omitted",
	"service":          "the Service Academy's service; the auto policy never selects the Academy (POLICY.md)",
	"disabled":         "the disabled outcome; pinned on other seeds by TestEveryTermElapsesFourYears",
	"exiled":           "chart 11's Noble exile; the pinned Noble seed is not exiled",
	"prison_years":     "a Rogue serving a sentence; the pinned Rogue seed serves none",
}

// collectKeys gathers every property name appearing anywhere in a document.
func collectKeys(doc any, seen map[string]bool) {
	switch value := doc.(type) {
	case map[string]any:
		for key, child := range value {
			seen[key] = true

			collectKeys(child, seen)
		}
	case []any:
		for _, child := range value {
			collectKeys(child, seen)
		}
	}
}

// schemaProperties lists every property name the schema describes.
func schemaProperties(t *testing.T) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Clean(schemaPath))
	if err != nil {
		t.Fatal(err)
	}

	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	defs, _ := schema["$defs"].(map[string]any)

	for _, def := range defs {
		object, _ := def.(map[string]any)

		properties, _ := object["properties"].(map[string]any)
		for name := range properties {
			seen[name] = true
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// consequenceFields is what each consequence kind is ever seen to carry,
// beyond the kind and cause every one of them has.
//
// Observed and pinned, not derived from the rules: it is a golden, and a
// change to it is a change to read rather than to wave through. It exists
// because the schema leaves all seven payload fields optional — which
// field a consequence carries turns on its kind, and fifty-five if/then
// branches in the schema would be a second copy of the code.
//
// The sets are unions rather than exact shapes. Payload fields are
// omitempty, so a zero value simply is not written: characteristic_change
// carries "value" where the resulting value is non-zero and drops it where
// it is not. An earlier version of this gate asserted that a kind carries
// the same fields every time, and the fixtures disproved it in eleven
// places within a second of running.
var consequenceFields = map[string]string{
	"aging_effect":            "characteristic delta value",
	"associated":              "career skill",
	"benefit":                 "career delta skill value",
	"benefit_lost":            "",
	"birthdate":               "detail value",
	"branch_set":              "career skill",
	"career_changed":          "career",
	"career_ended":            "career",
	"characteristic_change":   "characteristic delta value",
	"characteristic_reset":    "characteristic delta value",
	"characteristic_set":      "characteristic value",
	"comeback":                "value",
	"commendation":            "career delta skill value",
	"dead":                    "characteristic value",
	"discovery":               "value",
	"elevated":                "career",
	"entitlement":             "delta skill value",
	"exiled":                  "career value",
	"extremely_major_illness": "value",
	"fame_change":             "delta value",
	"fame_computed":           "mods skill value",
	"hobby_set":               "skill",
	"imprisoned":              "career value",
	"intrigue":                "career value",
	"job_set":                 "skill",
	"land_grant":              "career value",
	"major_illness":           "value",
	"major_set":               "skill",
	"mandatory_continue":      "",
	"masterpiece":             "career delta value",
	"medal":                   "career skill value",
	"minor_set":               "skill",
	"no_award":                "career skill value",
	"operation":               "career skill value",
	"payoff":                  "career delta skill value",
	"publication":             "delta value",
	"rank_set":                "career skill",
	"reserve":                 "career skill",
	"returned_from_exile":     "career",
	"sanity_mod":              "career delta",
	"scheme":                  "career skill value",
	"sentenced":               "career value",
	"service_badge":           "career value",
	"ship_shares":             "delta value",
	"skill_awarded":           "delta skill value",
	"specialty_set":           "career skill",
	"talent_set":              "value",
	"tenure":                  "career",
	"undercover_assignment":   "career skill",
	"wound_badge":             "value",
	"years_elapsed":           "value",
}

// TestEachConsequenceKindKeepsItsShape verifies the claim the schema makes
// about consequences: their payload turns on the kind, and the rule is
// pinned here rather than in the schema.
func TestEachConsequenceKindKeepsItsShape(t *testing.T) {
	records, err := filepath.Glob(filepath.Join("..", "chargen", "testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}

	for _, path := range records {
		for kind, fields := range consequenceShapes(t, path) {
			seen[kind] = true

			want, known := consequenceFields[kind]
			if !known {
				t.Errorf("consequence %q is new (%s in %s); add it to consequenceFields",
					kind, fields, filepath.Base(path))

				continue
			}

			for field := range strings.FieldsSeq(fields) {
				if !strings.Contains(" "+want+" ", " "+field+" ") {
					t.Errorf("consequence %q carries %q in %s, which it was not pinned to carry",
						kind, field, filepath.Base(path))
				}
			}
		}
	}

	for kind := range consequenceFields {
		if !seen[kind] {
			t.Errorf("consequence %q is pinned and no fixture emits it; remove it or produce one",
				kind)
		}
	}
}

// consequenceShapes reports the payload fields each consequence kind
// carries in one record, space-separated and sorted.
func consequenceShapes(t *testing.T, path string) map[string]string {
	t.Helper()

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}

	var record struct {
		Events []struct {
			Kind        string         `json:"kind"`
			Consequence map[string]any `json:"consequence"`
		} `json:"events"`
	}

	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}

	shapes := map[string]string{}

	for _, event := range record.Events {
		if event.Kind != "consequence" {
			continue
		}

		kind, _ := event.Consequence["kind"].(string)

		var fields []string

		for name := range event.Consequence {
			if name != "kind" && name != "cause" {
				fields = append(fields, name)
			}
		}

		sort.Strings(fields)
		shapes[kind] = strings.Join(fields, " ")
	}

	return shapes
}

// unemittedKinds are consequence kinds the engine declares and no pinned
// seed produces, and why. Each is a path the rules reach and the fourteen
// fixtures do not — a real outcome rather than a kind that does nothing.
var unemittedKinds = map[string]string{
	"career_not_begun": "a failed To Begin; every fixture's character enters his career",
	"disabled":         "the disabled outcome, pinned on other seeds by TestEveryTermElapsesFourYears",
	"job_undetermined": "chart 04's Job left undetermined; no fixture's Citizen reaches it",
	"waived":           "a waiver that succeeded; the auto policy's waivers all fail on these seeds",
}

// TestEveryConsequenceKindIsAccountedFor verifies the schema's vocabulary
// against what the engine can actually emit. A kind in the enum that
// nothing produces is either a rule nobody exercises or one that has
// quietly stopped working, and the difference is worth writing down.
func TestEveryConsequenceKindIsAccountedFor(t *testing.T) {
	declared := scan(t, isGoSource, regexp.MustCompile(`ConsequenceKind = "([a-z_0-9]+)"`))
	if len(declared) == 0 {
		t.Fatal("no consequence kinds found; this gate would be asserting nothing")
	}

	for _, kind := range declared {
		_, emitted := consequenceFields[kind]
		if emitted == (unemittedKinds[kind] != "") {
			if emitted {
				t.Errorf("%q is emitted by a fixture; remove it from unemittedKinds", kind)
			} else {
				t.Errorf("%q is declared and no fixture emits it; produce one, or say why not in unemittedKinds", kind)
			}
		}
	}

	inSchema := map[string]bool{}
	for _, kind := range schemaEnum(t, "consequenceEvent", "kind") {
		inSchema[kind] = true
	}

	for _, kind := range declared {
		if !inSchema[kind] {
			t.Errorf("the engine emits %q and the schema's vocabulary does not hold it", kind)
		}
	}
}

// isGoSource keeps the engine's own source, excluding tests.
func isGoSource(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

// schemaEnum reads one property's enum out of the schema.
func schemaEnum(t *testing.T, def, property string) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Clean(schemaPath))
	if err != nil {
		t.Fatal(err)
	}

	var schema struct {
		Defs map[string]struct {
			Properties map[string]struct {
				Enum []string `json:"enum"`
			} `json:"properties"`
		} `json:"$defs"`
	}

	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}

	return schema.Defs[def].Properties[property].Enum
}
