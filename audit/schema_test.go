package audit_test

// A checker for the subset of JSON Schema draft 2020-12 that
// character.schema.json uses, and nothing wider.
//
// Written rather than imported. A general validator is a better validator
// than this one, and the trade was six third-party modules against two
// hundred lines whose only job is to read one schema — in a repo that has
// no dependencies and a stated reason for that.
//
// The risk in a hand-written checker is specific and this project has met
// it before: a validator with a bug passes everything, and a gate that
// passes everything looks exactly like a gate that works. So it is not
// trusted on the strength of the fixtures passing. Every rule the schema
// states has a record in brokenRecords that must fail because of it, and
// TestTheCheckerCatchesWhatItClaimsTo is what makes the other tests mean
// something.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// errUnresolvedRef is a $ref naming a $def the schema does not define.
var errUnresolvedRef = errors.New("the schema refers to a $def it does not define")

// checker validates documents against one parsed schema.
type checker struct {
	root map[string]any
}

// newChecker parses a schema document and verifies that every $ref in it
// resolves.
//
// An unresolvable ref is the quietest failure this checker has: check on a
// subschema that does not exist finds nothing wrong with anything, so a
// renamed $def would delete every rule beneath the ref and leave the whole
// gate passing. It is refused at load time, the way the chart data is.
func newChecker(data []byte) (*checker, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing the schema: %w", err)
	}

	c := &checker{root: root}

	for _, ref := range refs(root) {
		if c.resolve(ref) == nil {
			return nil, fmt.Errorf("%w: %s", errUnresolvedRef, ref)
		}
	}

	return c, nil
}

// refs lists every $ref appearing anywhere in a schema document.
func refs(node any) []string {
	var found []string

	switch value := node.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "$ref" {
				if ref, ok := child.(string); ok {
					found = append(found, ref)
				}

				continue
			}

			found = append(found, refs(child)...)
		}
	case []any:
		for _, child := range value {
			found = append(found, refs(child)...)
		}
	}

	return found
}

// check reports every way a document fails the schema, by path. An empty
// result is a document the schema admits.
func (c *checker) check(schema map[string]any, doc any, path string) []string {
	problems := make([]string, 0, 4)

	// A $ref does not replace its siblings: draft 2020-12 applies the
	// keywords beside it as well, so the referenced schema is one more
	// thing the document has to satisfy rather than the only one.
	if ref, ok := schema["$ref"].(string); ok {
		problems = append(problems, c.check(c.resolve(ref), doc, path)...)
	}

	problems = append(problems, c.checkType(schema, doc, path)...)
	problems = append(problems, c.checkMinimum(schema, doc, path)...)
	problems = append(problems, c.checkEnum(schema, doc, path)...)
	problems = append(problems, c.checkObject(schema, doc, path)...)
	problems = append(problems, c.checkArray(schema, doc, path)...)
	problems = append(problems, c.checkCombinators(schema, doc, path)...)

	return problems
}

// resolve follows a local $ref.
func (c *checker) resolve(ref string) map[string]any {
	defs, _ := c.root["$defs"].(map[string]any)

	target, _ := defs[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any)

	return target
}

// checkType applies "type".
//
// JSON numbers all decode as float64, so an integer is a whole number
// rather than a Go int; a schema that accepted 1.5 as an integer would be
// admitting records the engine cannot have written.
func (c *checker) checkType(schema map[string]any, doc any, path string) []string {
	want, ok := schema["type"].(string)
	if !ok {
		return nil
	}

	if got := jsonType(doc); !satisfiesType(got, want) {
		return []string{fmt.Sprintf("%s: is %s, want %s", path, got, want)}
	}

	return nil
}

// satisfiesType reports whether a value jsonType calls got is acceptable
// where the schema asks for want.
//
// The two names are not disjoint in JSON Schema: every integer is a
// number, and only the reverse fails. jsonType reports a whole number as
// "integer", so without this a property declared "number" would be called
// invalid for the value 5. Nothing declares "number" today; the first
// fractional field added to the record — a rate, a multiplier — would
// otherwise fail every record whose value landed whole, and read as a
// schema bug rather than a checker one.
func satisfiesType(got, want string) bool {
	return got == want || (want == "number" && got == "integer")
}

// checkMinimum applies "minimum". It is a rule of its own rather than a
// tail of checkType: a bound written without a type beside it is still a
// bound, and folding it into the type check would silently drop one.
func (c *checker) checkMinimum(schema map[string]any, doc any, path string) []string {
	minimum, ok := schema["minimum"].(float64)
	if !ok {
		return nil
	}

	number, isNumber := doc.(float64)
	if !isNumber || number >= minimum {
		return nil
	}

	return []string{fmt.Sprintf("%s: is %v, want at least %v", path, number, minimum)}
}

// jsonType names a parsed JSON value's type the way a schema does.
//
// A whole number is an integer and a fractional one is not. All JSON
// numbers decode as float64, so without the distinction the schema would
// admit an age of 30.5 — a record the engine cannot have written.
func jsonType(doc any) string {
	switch value := doc.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		if value != float64(int64(value)) {
			return "number"
		}

		return "integer"
	}

	return ""
}

// checkEnum applies "enum" and "const".
func (c *checker) checkEnum(schema map[string]any, doc any, path string) []string {
	// reflect.DeepEqual rather than ==, and slices.ContainsFunc rather
	// than slices.Contains, because Go's == on two any values panics when
	// the dynamic type is uncomparable. A JSON array parses to []any and
	// an object to map[string]any, both uncomparable, so a schema stating
	// either as a const or an enum member would crash the test binary
	// instead of reporting a mismatch. Every const in
	// character.schema.json is a string today, so this cannot fire yet.
	if want, ok := schema["const"]; ok && !reflect.DeepEqual(doc, want) {
		return []string{fmt.Sprintf("%s: is %v, want %v", path, doc, want)}
	}

	allowed, ok := schema["enum"].([]any)
	if !ok {
		return nil
	}

	if slices.ContainsFunc(allowed, func(a any) bool { return reflect.DeepEqual(a, doc) }) {
		return nil
	}

	return []string{fmt.Sprintf("%s: %v is not one of the %d the schema allows", path, doc, len(allowed))}
}

// checkObject applies "properties", "required" and "additionalProperties".
func (c *checker) checkObject(schema map[string]any, doc any, path string) []string {
	object, ok := doc.(map[string]any)
	if !ok {
		return nil
	}

	properties, _ := schema["properties"].(map[string]any)

	var problems []string

	for _, name := range required(schema) {
		if _, present := object[name]; !present {
			problems = append(problems, fmt.Sprintf("%s: %s is required and absent", path, name))
		}
	}

	for name, value := range object {
		sub, known := properties[name].(map[string]any)
		if !known {
			if allow, ok := schema["additionalProperties"].(bool); ok && !allow {
				problems = append(problems, fmt.Sprintf("%s: %s is not a property the schema knows", path, name))
			}

			continue
		}

		problems = append(problems, c.check(sub, value, path+"."+name)...)
	}

	return problems
}

// required lists the property names a schema requires.
func required(schema map[string]any) []string {
	names, _ := schema["required"].([]any)

	var out []string

	for _, name := range names {
		if s, ok := name.(string); ok {
			out = append(out, s)
		}
	}

	return out
}

// checkArray applies "items".
func (c *checker) checkArray(schema map[string]any, doc any, path string) []string {
	array, ok := doc.([]any)
	if !ok {
		return nil
	}

	items, ok := schema["items"].(map[string]any)
	if !ok {
		return nil
	}

	var problems []string

	for i, value := range array {
		problems = append(problems, c.check(items, value, path+"["+strconv.Itoa(i)+"]")...)
	}

	return problems
}

// checkCombinators applies "allOf", "anyOf", "not" and "if"/"then", which
// together are what let the schema say an event carries exactly the
// payload its kind names.
func (c *checker) checkCombinators(schema map[string]any, doc any, path string) []string {
	var problems []string

	for _, sub := range subschemas(schema["allOf"]) {
		problems = append(problems, c.check(sub, doc, path)...)
	}

	if options := subschemas(schema["anyOf"]); len(options) > 0 && !c.matchesAny(options, doc, path) {
		problems = append(problems, fmt.Sprintf("%s: matches none of the %d alternatives", path, len(options)))
	}

	if sub, ok := schema["not"].(map[string]any); ok && c.matches(sub, doc, path) {
		problems = append(problems, path+": matches a shape the schema forbids")
	}

	if condition, ok := schema["if"].(map[string]any); ok {
		if then, ok := schema["then"].(map[string]any); ok && c.matches(condition, doc, path) {
			problems = append(problems, c.check(then, doc, path)...)
		}
	}

	return problems
}

// matches reports whether a document satisfies a subschema.
func (c *checker) matches(schema map[string]any, doc any, path string) bool {
	return len(c.check(schema, doc, path)) == 0
}

// matchesAny reports whether a document satisfies any of the subschemas.
func (c *checker) matchesAny(options []map[string]any, doc any, path string) bool {
	for _, option := range options {
		if c.matches(option, doc, path) {
			return true
		}
	}

	return false
}

// subschemas reads a keyword holding a list of schemas.
func subschemas(value any) []map[string]any {
	list, _ := value.([]any)

	var out []map[string]any

	for _, item := range list {
		if sub, ok := item.(map[string]any); ok {
			out = append(out, sub)
		}
	}

	return out
}

// validate checks one document against the schema, returning its problems.
func validate(schemaPath, docPath string) ([]string, error) {
	schema, err := os.ReadFile(filepath.Clean(schemaPath))
	if err != nil {
		return nil, fmt.Errorf("reading the schema: %w", err)
	}

	c, err := newChecker(schema)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Clean(docPath))
	if err != nil {
		return nil, fmt.Errorf("reading the record: %w", err)
	}

	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", docPath, err)
	}

	return c.check(c.root, doc, "record"), nil
}
