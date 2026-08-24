package docs_test

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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// checker validates documents against one parsed schema.
type checker struct {
	root map[string]any
}

// newChecker parses a schema document.
func newChecker(data []byte) (*checker, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing the schema: %w", err)
	}

	return &checker{root: root}, nil
}

// check reports every way a document fails the schema, by path. An empty
// result is a document the schema admits.
func (c *checker) check(schema map[string]any, doc any, path string) []string {
	if ref, ok := schema["$ref"].(string); ok {
		return c.check(c.resolve(ref), doc, path)
	}

	problems := make([]string, 0, 4)

	problems = append(problems, c.checkType(schema, doc, path)...)
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

// checkType applies "type" and "minimum".
//
// JSON numbers all decode as float64, so an integer is a whole number
// rather than a Go int; a schema that accepted 1.5 as an integer would be
// admitting records the engine cannot have written.
func (c *checker) checkType(schema map[string]any, doc any, path string) []string {
	want, ok := schema["type"].(string)
	if !ok {
		return nil
	}

	number, isNumber := doc.(float64)

	if got := jsonType(doc); got != want {
		return []string{fmt.Sprintf("%s: is %s, want %s", path, jsonType(doc), want)}
	}

	if minimum, ok := schema["minimum"].(float64); ok && isNumber && number < minimum {
		return []string{fmt.Sprintf("%s: is %v, want at least %v", path, number, minimum)}
	}

	return nil
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
	if want, ok := schema["const"]; ok && doc != want {
		return []string{fmt.Sprintf("%s: is %v, want %v", path, doc, want)}
	}

	allowed, ok := schema["enum"].([]any)
	if !ok {
		return nil
	}

	if slices.Contains(allowed, doc) {
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

	for name := range strings.FieldsSeq(required(schema)) {
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

// required renders a schema's required list as a space-separated string,
// which is only ever read back by Fields above.
func required(schema map[string]any) string {
	names, _ := schema["required"].([]any)

	var out []string

	for _, name := range names {
		if s, ok := name.(string); ok {
			out = append(out, s)
		}
	}

	return strings.Join(out, " ")
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
