package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// noInput is the empty stdin for tests that answer no interactive prompt.
func noInput() io.Reader { return strings.NewReader("") }

// noSeed is a seed source for tests that must not draw a default seed.
func noSeed(t *testing.T) func() (uint64, error) {
	t.Helper()

	return func() (uint64, error) {
		t.Error("seed source called; --seed should have been used")

		return 0, nil
	}
}

// fixedSeed is a seed source returning a known value.
func fixedSeed(seed uint64) func() (uint64, error) {
	return func() (uint64, error) { return seed, nil }
}

// TestNewSeedGolden verifies `new --seed 1` writes the exact seed-1 golden
// record to stdout.
func TestNewSeedGolden(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run([]string{"new", "--auto", "--seed", "1"}, noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}

	want, err := os.ReadFile(filepath.Join("..", "..", "chargen", "testdata", "seed1.json"))
	if err != nil {
		t.Fatal(err)
	}

	if stdout.String() != string(want) {
		t.Errorf("stdout differs from chargen/testdata/seed1.json:\n%s", stdout.String())
	}
}

// TestReplaySubcommand verifies the end-to-end loop the replay contract
// describes: generate a record, then re-run it from the file alone and
// have it reproduce itself (docs/PRD.md, Replay and provenance contract).
// The engine-level sweep over every fixture lives in
// chargen.TestReplayRoundTrip; this pins the wiring and the exit code.
func TestReplaySubcommand(t *testing.T) {
	record := filepath.Join(t.TempDir(), "character.json")

	var stdout, stderr bytes.Buffer
	if code := run(
		[]string{
			"new",
			"--auto",
			"--seed",
			"1",
			"-o",
			record,
		},
		noSeed(t),
		noInput(),
		&stdout,
		&stderr,
	); code != exitOK {
		t.Fatalf("new: exit %d, stderr: %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	if code := run([]string{"replay", record}, noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("replay: exit %d, stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "reproduced from seed 1") {
		t.Errorf("replay said %q, which does not report the seed it reproduced", stdout.String())
	}
}

// TestReplayReportsTheDivergingEvent verifies a tampered record fails with
// the sequence number of the event that disagreed, which is the whole of
// what the PRD promises a reader: "exits non-zero at the first mismatch,
// reporting the diverging event's sequence number".
func TestReplayReportsTheDivergingEvent(t *testing.T) {
	record := filepath.Join(t.TempDir(), "character.json")

	var stdout, stderr bytes.Buffer
	if code := run(
		[]string{
			"new",
			"--auto",
			"--seed",
			"1",
			"-o",
			record,
		},
		noSeed(t),
		noInput(),
		&stdout,
		&stderr,
	); code != exitOK {
		t.Fatalf("new: exit %d, stderr: %s", code, stderr.String())
	}

	tampered := tamperFirstThrow(t, record)

	stdout.Reset()
	stderr.Reset()

	if code := run([]string{"replay", record}, noSeed(t), noInput(), &stdout, &stderr); code != exitError {
		t.Fatalf("replay of a tampered record: exit %d, want %d", code, exitError)
	}

	if !strings.Contains(stderr.String(), fmt.Sprintf("event %d", tampered)) {
		t.Errorf("replay reported %q, which does not name diverging event %d", stderr.String(), tampered)
	}
}

// tamperFirstThrow adds one to the total of the record's first throw and
// returns that event's sequence number, so the caller can require the
// divergence report to name it.
func tamperFirstThrow(t *testing.T, record string) int {
	t.Helper()

	data, err := os.ReadFile(record) //nolint:gosec // the path is a temp file the test just wrote
	if err != nil {
		t.Fatal(err)
	}

	var character chargen.Character
	if err := json.Unmarshal(data, &character); err != nil {
		t.Fatal(err)
	}

	for _, event := range character.Events {
		if event.Kind != chargen.EventThrow {
			continue
		}

		event.Throw.Total++

		edited, err := json.Marshal(character)
		if err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(record, edited, 0o600); err != nil {
			t.Fatal(err)
		}

		return event.Seq
	}

	t.Fatal("the record holds no throw to tamper with")

	return 0
}

// TestNewSeedZero verifies --seed 0 is honored as an explicit seed rather
// than triggering the default seed source.
func TestNewSeedZero(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run([]string{"new", "--auto", "--seed", "0"}, noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), `"seed": 0`) {
		t.Errorf("record does not carry seed 0:\n%s", stdout.String())
	}
}

// TestNewDefaultSeed verifies the injected seed source supplies the seed
// when --seed is absent, and that the drawn seed lands in the provenance.
func TestNewDefaultSeed(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run([]string{"new", "--auto"}, fixedSeed(7), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), `"seed": 7`) {
		t.Errorf("record does not carry drawn seed 7:\n%s", stdout.String())
	}
}

// TestNewName verifies --name lands in the record.
func TestNewName(t *testing.T) {
	var stdout, stderr bytes.Buffer

	args := []string{"new", "--auto", "--seed", "1", "--name", "Eneri Dinsha"}
	if code := run(args, noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), `"name": "Eneri Dinsha"`) {
		t.Errorf("record does not carry the name:\n%s", stdout.String())
	}
}

// TestNewOutputFile verifies -o writes the file, refuses to overwrite
// without --force, and overwrites with it (docs/PRD.md CLI sketch).
func TestNewOutputFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "character.json")

	var stdout, stderr bytes.Buffer

	if code := run(
		[]string{
			"new",
			"--auto",
			"--seed",
			"1",
			"-o",
			path,
		},
		noSeed(t),
		noInput(),
		&stdout,
		&stderr,
	); code != exitOK {
		t.Fatalf("first write: exit %d, stderr: %s", code, stderr.String())
	}

	if stdout.Len() != 0 {
		t.Error("-o also wrote to stdout")
	}

	stderr.Reset()

	if code := run(
		[]string{
			"new",
			"--auto",
			"--seed",
			"2",
			"-o",
			path,
		},
		noSeed(t),
		noInput(),
		&stdout,
		&stderr,
	); code != exitError {
		t.Fatalf("overwrite without --force: exit %d, want %d", code, exitError)
	}

	if !strings.Contains(stderr.String(), "--force") {
		t.Errorf("overwrite refusal does not mention --force: %s", stderr.String())
	}

	forceArgs := []string{"new", "--auto", "--seed", "2", "-o", path, "--force"}
	if code := run(forceArgs, noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("overwrite with --force: exit %d, stderr: %s", code, stderr.String())
	}

	data, err := os.ReadFile(path) //nolint:gosec // G304: fixed test-owned temp path.
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), `"seed": 2`) {
		t.Error("--force did not replace the file contents")
	}
}

// TestNewForcedCareer verifies --career maps case-insensitively to the
// canonical Book 1 name.
func TestNewForcedCareer(t *testing.T) {
	var stdout, stderr bytes.Buffer

	args := []string{"new", "--auto", "--seed", "1", "--career", "citizen"}
	if code := run(args, noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), `"career": "Citizen"`) {
		t.Errorf("record does not carry the forced career:\n%s", stdout.String())
	}
}

// TestNewHomeworldFlag verifies --homeworld "UWP TC..." lands in the
// record with its trade classifications.
func TestNewHomeworldFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	args := []string{"new", "--auto", "--seed", "1", "--homeworld", "C200423-7 Va Ni"}
	if code := run(args, noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}

	for _, want := range []string{`"uwp": "C200423-7"`, `"Va"`, `"Ni"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("record missing %s", want)
		}
	}
}

// TestRenderGoldens verifies render reproduces the seed-1 sheet and history
// goldens from a record written by new.
func TestRenderGoldens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "character.json")

	var stdout, stderr bytes.Buffer

	if code := run(
		[]string{
			"new",
			"--auto",
			"--seed",
			"1",
			"-o",
			path,
		},
		noSeed(t),
		noInput(),
		&stdout,
		&stderr,
	); code != exitOK {
		t.Fatalf("new: exit %d, stderr: %s", code, stderr.String())
	}

	tests := []struct {
		name   string
		args   []string
		golden string
	}{
		{"sheet", []string{"render", path}, "seed1_sheet.md"},
		{"history", []string{"render", "--history", path}, "seed1_history.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer

			if code := run(tt.args, noSeed(t), noInput(), &out, &errOut); code != exitOK {
				t.Fatalf("exit %d, stderr: %s", code, errOut.String())
			}

			want, err := os.ReadFile(filepath.Join("..", "..", "render", "testdata", tt.golden))
			if err != nil {
				t.Fatal(err)
			}

			if out.String() != string(want) {
				t.Errorf("output differs from %s:\n%s", tt.golden, out.String())
			}
		})
	}
}

// TestErrors verifies exit codes: 1 for operational errors, 2 for usage
// errors.
// errorCase is one CLI invocation and the exit code it must produce.
type errorCase struct {
	name string
	args []string
	code int
}

// errorCases enumerates the exit-code contract: 1 for operational
// errors, 2 for usage errors.
func errorCases(garbage, noSchema, foreign string) []errorCase {
	return []errorCase{
		{"no arguments", nil, exitUsage},
		{"unknown subcommand", []string{"bogus"}, exitUsage},
		{"unknown flag", []string{"new", "--bogus"}, exitUsage},
		// Without --auto the session is interactive, and empty input
		// ends it before the first answer: abandoned, not misused.
		{"new abandoned at once", []string{"new", "--seed", "1"}, exitError},
		{"new stray arguments", []string{"new", "--auto", "--seed", "1", "out.json"}, exitUsage},
		{"unknown career", []string{"new", "--auto", "--seed", "1", "--career", "craftsman"}, exitUsage},
		// Chart 01 entry is automatic only "if TWO skill-6 and Craftsman-1"
		// (p. 75), which no school leaver has; chart 13 "is never a first
		// career" (p. 87). Both exist, and neither can open a lifepath.
		{"career unavailable at the start", []string{"new", "--auto", "--seed", "1", "--career", "Craftsman"}, exitUsage},
		{"never a first career", []string{"new", "--auto", "--seed", "1", "--career", "Functionary"}, exitUsage},
		// A year that is not one is caught by the flag check; a year the
		// character has outlived is only knowable once the age is, so it
		// comes back from the engine — both are the caller's fault.
		{"current year zero", []string{"new", "--auto", "--seed", "1", "--current-year", "0"}, exitUsage},
		{"current year negative", []string{"new", "--auto", "--seed", "1", "--current-year", "-5"}, exitUsage},
		{"current year the character outlived", []string{"new", "--auto", "--seed", "1", "--current-year", "30"}, exitUsage},
		{"partial UWP", []string{"new", "--auto", "--seed", "1", "--homeworld", "A78899"}, exitUsage},
		{"unknown TC", []string{"new", "--auto", "--seed", "1", "--homeworld", "A788899-C Qq"}, exitUsage},
		//nolint:dupword // the repeated TC is the case under test
		{"duplicate TC", []string{"new", "--auto", "--seed", "1", "--homeworld", "A788899-C Pa Pa"}, exitUsage},
		{"render without file", []string{"render"}, exitUsage},
		{"render missing file", []string{"render", "does-not-exist.json"}, exitError},
		{"render garbage file", []string{"render", garbage}, exitError},
		{"render non-record", []string{"render", noSchema}, exitError},
		// render takes no --format: Markdown is the only format, and an
		// unknown flag must be refused rather than ignored.
		{"render rejects --format", []string{"render", "--format", "md", noSchema}, exitUsage},
		{"batch without --auto", []string{"batch", "--count", "2"}, exitUsage},
		{"batch without --count", []string{"batch", "--auto"}, exitUsage},
		{"batch negative count", []string{"batch", "--auto", "--count", "-2"}, exitUsage},
		{"batch stray arguments", []string{"batch", "--auto", "--count", "2", "out.jsonl"}, exitUsage},
		{"batch unknown career", []string{"batch", "--auto", "--count", "2", "--career", "bogus"}, exitUsage},
		{"batch career unavailable", []string{"batch", "--auto", "--count", "2", "--career", "Craftsman"}, exitUsage},
		{"batch outlived current year", []string{"batch", "--auto", "--count", "2", "--current-year", "30"}, exitUsage},
		// One check for the flags both subcommands share, so a name new
		// refuses is one batch refuses too.
		{"name with a line break", []string{"new", "--auto", "--seed", "1", "--name", "a\nb"}, exitUsage},
		{"batch name with a line break", []string{"batch", "--auto", "--count", "1", "--name", "a\nb"}, exitUsage},
		{"batch current year zero", []string{"batch", "--auto", "--count", "2", "--current-year", "0"}, exitUsage},
		{"replay without file", []string{"replay"}, exitUsage},
		{"replay stray arguments", []string{"replay", noSchema, "extra"}, exitUsage},
		{"replay missing file", []string{"replay", "does-not-exist.json"}, exitError},
		{"replay garbage file", []string{"replay", garbage}, exitError},
		{"replay non-record", []string{"replay", noSchema}, exitError},
		// A divergence is an operational error, not a usage one: the
		// command worked and the answer is no.
		{"replay a foreign record", []string{"replay", foreign}, exitError},
	}
}

// TestTheExitStatusIsThePromisedOne holds the numbers docs/COMPATIBILITY.md
// promises a script, so renumbering them breaks a test that names the
// promise rather than seventy that only use the constants.
func TestTheExitStatusIsThePromisedOne(t *testing.T) {
	if exitOK != 0 || exitError != 1 || exitUsage != 2 {
		t.Errorf("exit statuses are %d/%d/%d; docs/COMPATIBILITY.md promises 0 success, 1 failed, 2 the caller's fault",
			exitOK, exitError, exitUsage)
	}
}

// TestEveryErrorSentinelIsClassified is the ratchet under isUsageError.
// Exit 2 means the caller's fault, and the engine says which errors those
// are by putting chargen.ErrInput on their chain. What nothing used to say
// is whether a new sentinel had been thought about at all: the old
// hand-kept list defaulted it to exit 1, and nothing failed. This table
// names every exported Err* in the module as the caller's fault or not,
// and a sentinel missing from it fails here until someone decides.
func TestEveryErrorSentinelIsClassified(t *testing.T) {
	callersFault := map[string]bool{
		"career.ErrUnknownCareer":      false, // a registry lookup; --career reaches chargen's own
		"benefit.ErrUnknownKind":       false,
		"calendar.ErrDay":              false,
		"calendar.ErrDice":             false,
		"chargen.ErrCareerUnavailable": true,
		"chargen.ErrCurrentYear":       true,
		"chargen.ErrInput":             true, // the mark itself
		"chargen.ErrReplayDiverged":    false,
		"chargen.ErrReplayProvenance":  false,
		"chargen.ErrUnknownCareer":     true,
		"education.ErrUnknownProgram":  false,
		"ehex.ErrDigit":                false, // reaches --homeworld only inside world.ErrInvalidUWP
		"ehex.ErrRange":                false,
		"interactive.ErrAbandoned":     false,
		"medal.ErrOffTable":            false,
		"skill.ErrUnknownSkill":        false,
		// world cannot import chargen, so these are marked where the
		// engine meets a supplied homeworld (chargen/homeworld.go) and
		// exercised through --homeworld in errorCases.
		"world.ErrDuplicateTC": true,
		"world.ErrInvalidUWP":  true,
		"world.ErrUnknownTC":   true,
	}

	found := exportedSentinels(t, filepath.Join("..", ".."))

	for _, name := range found {
		if _, ok := callersFault[name]; !ok {
			t.Errorf("%s is not classified: decide whether a caller can cause it, and if so mark it with chargen.ErrInput", name)
		}
	}

	for name := range callersFault {
		if !slices.Contains(found, name) {
			t.Errorf("%s is classified but no longer exists", name)
		}
	}

	// chargen's own carry the mark themselves, wherever they are returned.
	for _, sentinel := range []error{chargen.ErrCareerUnavailable, chargen.ErrCurrentYear, chargen.ErrUnknownCareer} {
		if !errors.Is(sentinel, chargen.ErrInput) {
			t.Errorf("%v does not carry chargen.ErrInput", sentinel)
		}
	}
}

// exportedSentinels lists every package-level exported Err* variable in
// the module's non-test source, as "package.Name".
func exportedSentinels(t *testing.T, root string) []string {
	t.Helper()

	var found []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case entry.IsDir() && path != root && !isSourceDir(entry.Name()):
			return filepath.SkipDir
		case entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go"):
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		found = append(found, sentinelsIn(file)...)

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return found
}

// isSourceDir reports whether a directory can hold the module's source:
// not a dot-directory (worktrees live under .claude) and not testdata.
func isSourceDir(name string) bool {
	return !strings.HasPrefix(name, ".") && name != "testdata"
}

// sentinelsIn lists one file's package-level exported Err* variables.
func sentinelsIn(file *ast.File) []string {
	var found []string

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}

		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range value.Names {
				if strings.HasPrefix(name.Name, "Err") && name.IsExported() {
					found = append(found, file.Name.Name+"."+name.Name)
				}
			}
		}
	}

	return found
}

func TestErrors(t *testing.T) {
	garbage := filepath.Join(t.TempDir(), "garbage.json")
	if err := os.WriteFile(garbage, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	noSchema := filepath.Join(t.TempDir(), "noschema.json")
	if err := os.WriteFile(noSchema, []byte(`{"upp":"777777"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	foreign := filepath.Join(t.TempDir(), "foreign.json")
	if err := os.WriteFile(foreign, []byte(`{"schema_version":"0.0.1"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := errorCases(garbage, noSchema, foreign)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			if code := run(tt.args, fixedSeed(1), noInput(), &stdout, &stderr); code != tt.code {
				t.Errorf("exit %d, want %d (stderr: %s)", code, tt.code, stderr.String())
			}
		})
	}
}

// readJSONL parses a batch's JSONL output.
func readJSONL(t *testing.T, out string) []chargen.Character {
	t.Helper()

	var characters []chargen.Character

	for line := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
		var character chargen.Character
		if err := json.Unmarshal([]byte(line), &character); err != nil {
			t.Fatalf("batch emitted a line that is not a record: %v", err)
		}

		characters = append(characters, character)
	}

	return characters
}

// runBatchOK runs a batch and fails the test if it does not succeed.
func runBatchOK(t *testing.T, args ...string) string {
	t.Helper()

	var stdout, stderr bytes.Buffer
	if code := run(args, noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("batch: exit %d, stderr: %s", code, stderr.String())
	}

	return stdout.String()
}

// TestBatchDerivesSeeds verifies the seed rule the PRD states: "derives
// each member's seed from the base seed + index, recorded in each record".
// The recorded seed is the point — it is what makes a member replayable
// from the line it lands in.
func TestBatchDerivesSeeds(t *testing.T) {
	characters := readJSONL(t, runBatchOK(t, "batch", "--count", "4", "--auto", "--seed", "100"))

	if len(characters) != 4 {
		t.Fatalf("batch --count 4 emitted %d records", len(characters))
	}

	for i, character := range characters {
		if want := uint64(100 + i); character.RNG.Seed != want {
			t.Errorf("member %d records seed %d, want %d", i, character.RNG.Seed, want)
		}
	}
}

// TestBatchMembersReplay verifies every member of a batch verifies on its
// own. A batch record that cannot be replayed is not a character record;
// this is the property the seed derivation exists to preserve.
func TestBatchMembersReplay(t *testing.T) {
	for _, character := range readJSONL(t, runBatchOK(t, "batch", "--count", "5", "--auto", "--seed", "700")) {
		if _, err := chargen.Replay(character); err != nil {
			t.Errorf("batch member with seed %d does not replay: %v", character.RNG.Seed, err)
		}
	}
}

// TestBatchEmitsTheDead verifies a character who died during generation
// still reaches the output. Interpretation I-51 leaves open whether "all
// efforts are lost" governs the tool's output; a batch that silently
// dropped members would make --count a lie, so the record is emitted with
// its dead flag set and the caller decides.
func TestBatchEmitsTheDead(t *testing.T) {
	characters := readJSONL(t, runBatchOK(t, "batch", "--count", "30", "--auto", "--seed", "1"))

	if len(characters) != 30 {
		t.Fatalf("batch --count 30 emitted %d records", len(characters))
	}

	dead := 0

	for _, character := range characters {
		if character.Dead {
			dead++
		}
	}

	// Not a Skip: a skip here would pass silently the day the seed range
	// stops reaching a death, and the test would then be asserting
	// nothing about the behaviour it names.
	if dead == 0 {
		t.Error("no member of this run died, so this test no longer exercises a dead character; " +
			"widen the count or repin the seed")
	}
}

// TestBatchDirectory verifies -o dir writes one file per character, named
// for the seed that produced it.
func TestBatchDirectory(t *testing.T) {
	dir := t.TempDir()

	var stdout, stderr bytes.Buffer
	if code := run([]string{"batch", "--count", "3", "--auto", "--seed", "100", "-o", dir},
		noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("batch -o dir: exit %d, stderr: %s", code, stderr.String())
	}

	if stdout.Len() != 0 {
		t.Errorf("batch -o dir also wrote to stdout: %s", stdout.String())
	}

	for _, seed := range []int{100, 101, 102} {
		path := filepath.Join(dir, fmt.Sprintf("character-%d.json", seed))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("batch -o dir did not write %s", filepath.Base(path))
		}
	}
}

// TestBatchDirectoryTrailingSlash verifies -o with a trailing separator is
// taken as a directory and created. Without this the README's own example
// (`-o npcs/`) fails on a fresh checkout, and the same path without the
// slash silently writes the whole run into one file named npcs.
func TestBatchDirectoryTrailingSlash(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "npcs") + string(os.PathSeparator)

	var stdout, stderr bytes.Buffer
	if code := run([]string{"batch", "--count", "2", "--auto", "--seed", "300", "-o", dir},
		noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
		t.Fatalf("batch -o npcs/: exit %d, stderr: %s", code, stderr.String())
	}

	for _, seed := range []int{300, 301} {
		path := filepath.Join(dir, fmt.Sprintf("character-%d.json", seed))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("batch -o npcs/ did not write %s", filepath.Base(path))
		}
	}
}

// TestBatchWritesNothingOnConflict verifies a batch that cannot write every
// file writes none of them. Writing until the conflict would leave a
// directory holding part of a run that failed, which is worse than the
// failure: the caller cannot tell it from a run that succeeded.
func TestBatchWritesNothingOnConflict(t *testing.T) {
	dir := t.TempDir()

	// Seed the conflict on the last member, so a naive implementation has
	// already written the first two by the time it notices.
	blocker := filepath.Join(dir, "character-102.json")
	if err := os.WriteFile(blocker, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"batch", "--count", "3", "--auto", "--seed", "100", "-o", dir},
		noSeed(t), noInput(), &stdout, &stderr); code != exitError {
		t.Fatalf("batch over an existing file: exit %d, want %d", code, exitError)
	}

	for _, seed := range []int{100, 101} {
		path := filepath.Join(dir, fmt.Sprintf("character-%d.json", seed))
		if _, err := os.Stat(path); err == nil {
			t.Errorf("batch wrote %s despite refusing the run", filepath.Base(path))
		}
	}

	if data, err := os.ReadFile(blocker); err != nil || string(data) != "{}" { //nolint:gosec // a temp path the test wrote
		t.Errorf("batch overwrote the existing file without --force")
	}
}

// A batch whose second member fails: seed 2 is young enough for Imperial
// year 40 and seed 3 is not, so the engine refuses member 2 of 2 for the
// caller's --current-year.
var failsOnTheSecondMember = []string{"batch", "--auto", "--count", "2", "--seed", "2", "--current-year", "40"}

// TestABatchThatFailsHalfwayWritesNothing verifies the batch contract for
// the destinations that can be staged: a JSONL file and a directory are
// put in place only once every member has generated, so a run that fails
// on its second member leaves no file, no directory, and nothing staged.
func TestABatchThatFailsHalfwayWritesNothing(t *testing.T) {
	for _, out := range []string{"run.jsonl", filepath.Join("deep", "npcs") + string(filepath.Separator)} {
		t.Run(out, func(t *testing.T) {
			root := t.TempDir()

			var stdout, stderr bytes.Buffer

			// Joined by hand: filepath.Join would clean away the
			// trailing separator that makes npcs/ a directory.
			args := append(slices.Clone(failsOnTheSecondMember), "-o", root+string(filepath.Separator)+out)
			if code := run(args, noSeed(t), noInput(), &stdout, &stderr); code != exitUsage {
				t.Fatalf("exit %d, want %d (stderr: %s)", code, exitUsage, stderr.String())
			}

			if !strings.Contains(stderr.String(), "character 2 of 2") {
				t.Errorf("the failure does not name the member: %s", stderr.String())
			}

			if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
				t.Errorf("a failed batch left %v behind (%v)", entries, err)
			}
		})
	}
}

// TestAStreamThatFailsHalfwayStops pins the one destination that cannot
// be staged. JSONL on stdout streams — holding the run back would hold
// all of it in memory — so the members before the failure are already
// out, as whole lines, and the exit status is what says the stream is
// incomplete.
func TestAStreamThatFailsHalfwayStops(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run(failsOnTheSecondMember, noSeed(t), noInput(), &stdout, &stderr); code != exitUsage {
		t.Fatalf("exit %d, want %d (stderr: %s)", code, exitUsage, stderr.String())
	}

	if members := readJSONL(t, stdout.String()); len(members) != 1 || members[0].RNG.Seed != 2 {
		t.Errorf("stdout should hold member 1 alone, whole; got %d records", len(members))
	}
}

// TestABatchDestinationIsCheckedFirst verifies -o is resolved before
// anything is generated: a conflict is refused without a run, and a
// directory that cannot exist is refused by name.
func TestABatchDestinationIsCheckedFirst(t *testing.T) {
	root := t.TempDir()

	existing := filepath.Join(root, "run.jsonl")
	if err := os.WriteFile(existing, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, out, want string
	}{
		{"an existing JSONL file", existing, "exists"},
		{"a file where the directory would be", existing + string(filepath.Separator), "not a directory"},
		{"a file above the directory", filepath.Join(existing, "npcs") + string(filepath.Separator), "not a directory"},
		{"a JSONL file with no directory", filepath.Join(root, "missing", "run.jsonl"), "no such directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			// --career nosuch would exit 2 if generation were reached.
			args := []string{"batch", "--auto", "--count", "2", "--seed", "1", "--career", "nosuch", "-o", tt.out}
			if code := run(args, noSeed(t), noInput(), &stdout, &stderr); code != exitError {
				t.Fatalf("exit %d, want %d (stderr: %s)", code, exitError, stderr.String())
			}

			if !strings.Contains(stderr.String(), tt.want) {
				t.Errorf("stderr does not say %q: %s", tt.want, stderr.String())
			}
		})
	}
}

// TestARefusedBatchMakesNoDirectory verifies that resolving -o only looks.
// "-o newdir/" declares a directory, but a run refused before it wrote
// anything must not leave an empty one behind.
func TestARefusedBatchMakesNoDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "newdir")

	var stdout, stderr bytes.Buffer

	args := []string{
		"batch", "--auto", "--count", "2", "--seed", "1", "--career", "nosuch",
		"-o", dir + string(filepath.Separator),
	}
	if code := run(args, noSeed(t), noInput(), &stdout, &stderr); code != exitUsage {
		t.Fatalf("exit %d, want %d (stderr: %s)", code, exitUsage, stderr.String())
	}

	if _, err := os.Stat(dir); err == nil {
		t.Error("a refused batch created its directory")
	}
}

// TestInteractiveNewWritesACharacter verifies the default mode. Without
// --auto the player answers, and the record attests that he did.
func TestInteractiveNewWritesACharacter(t *testing.T) {
	record := filepath.Join(t.TempDir(), "character.json")

	var stdout, stderr bytes.Buffer

	script := strings.NewReader(strings.Repeat("1\n", 4000))
	if code := run([]string{"new", "--seed", "1", "-o", record}, noSeed(t), script, &stdout, &stderr); code != exitOK {
		t.Fatalf("interactive new: exit %d, stderr: %s", code, stderr.String())
	}

	data, err := os.ReadFile(record) //nolint:gosec // a temp path this test named
	if err != nil {
		t.Fatal(err)
	}

	var character chargen.Character
	if err := json.Unmarshal(data, &character); err != nil {
		t.Fatal(err)
	}

	if character.PolicyVersion != "none" {
		t.Errorf("policy_version is %q, want %q for a player-decided run", character.PolicyVersion, "none")
	}
}

// TestOutputIsCheckedBeforeTheFirstQuestion verifies that an interactive
// run aimed at a file it may not overwrite is refused before it asks
// anything. The write used to be the check, and the write comes after the
// last answer, so the refusal took the whole lifepath with it.
func TestOutputIsCheckedBeforeTheFirstQuestion(t *testing.T) {
	record := filepath.Join(t.TempDir(), "character.json")
	if err := os.WriteFile(record, []byte("already here\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer

	script := strings.NewReader(strings.Repeat("1\n", 4000))
	if code := run([]string{"new", "--seed", "1", "-o", record}, noSeed(t), script, &stdout, &stderr); code != exitError {
		t.Fatalf("exit %d, want %d (stderr: %s)", code, exitError, stderr.String())
	}

	// One line, the refusal: no rule, no prompt, nothing asked.
	if got := strings.TrimSpace(stderr.String()); strings.Contains(got, "\n") || !strings.Contains(got, "--force") {
		t.Errorf("stderr should be the refusal alone, got:\n%s", stderr.String())
	}

	data, err := os.ReadFile(record) //nolint:gosec // a temp path this test named
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "already here\n" {
		t.Errorf("the existing file was changed: %q", data)
	}
}

// TestNewRefusesAnOutputItCannotWrite verifies that -o naming a directory,
// or a file in a directory that does not exist, is refused by name and
// before generation, and that --force does not change the answer: no
// amount of forcing puts a file where a directory is.
func TestNewRefusesAnOutputItCannotWrite(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "crew")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "crew.json"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		out  string
		args []string
		want string
	}{
		{"an existing directory", dir, nil, "names a directory"},
		{"an existing directory, forced", dir, []string{"--force"}, "names a directory"},
		{"a trailing separator", filepath.Join(root, "npcs") + string(filepath.Separator), nil, "names a directory"},
		{"a missing parent", filepath.Join(root, "missing", "x.json"), nil, "no such directory"},
		{"a file where a directory should be", filepath.Join(dir, "..", "crew.json", "x.json"), nil, "not a directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			args := append([]string{"new", "--auto", "--seed", "1", "-o", tt.out}, tt.args...)
			if code := run(args, noSeed(t), noInput(), &stdout, &stderr); code != exitError {
				t.Fatalf("exit %d, want %d (stderr: %s)", code, exitError, stderr.String())
			}

			if !strings.Contains(stderr.String(), tt.want) {
				t.Errorf("stderr does not say %q: %s", tt.want, stderr.String())
			}

			if strings.Contains(stderr.String(), ".t5chargen-") {
				t.Errorf("stderr leaks a temporary file name: %s", stderr.String())
			}

			if stdout.Len() != 0 {
				t.Errorf("a refused run wrote to stdout: %s", stdout.String())
			}
		})
	}

	// The trailing-separator case is a declaration new does not honour,
	// so it must not have made the directory either.
	if _, err := os.Stat(filepath.Join(root, "npcs")); err == nil {
		t.Error("a refused -o npcs/ created the directory")
	}
}

// TestAFailedWriteNamesNoTemporaryFile verifies that when the final rename
// fails, the reader is told about the path they named and not about the
// temporary file beside it. batch --force onto a member path that is a
// directory is the case that reaches the rename: nothing checks the shape
// of a path --force is allowed to replace.
func TestAFailedWriteNamesNoTemporaryFile(t *testing.T) {
	out := t.TempDir()
	if err := os.Mkdir(filepath.Join(out, "character-7.json"), 0o750); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer

	args := []string{"batch", "--auto", "--count", "1", "--seed", "7", "--force", "-o", out + string(filepath.Separator)}
	if code := run(args, noSeed(t), noInput(), &stdout, &stderr); code != exitError {
		t.Fatalf("exit %d, want %d (stderr: %s)", code, exitError, stderr.String())
	}

	if !strings.Contains(stderr.String(), "character-7.json") || strings.Contains(stderr.String(), ".t5chargen-") {
		t.Errorf("stderr should name the record and not the temporary file: %s", stderr.String())
	}
}

// TestARecordOutputCannotTakeGoesToStdout verifies the last line of
// defence: a record the -o write fails on is written to stdout, and the
// run still exits 1, because it did not do what it was asked. The file
// appears after the session starts — past the pre-flight check, which
// only looks — which is the case that check cannot catch.
func TestARecordOutputCannotTakeGoesToStdout(t *testing.T) {
	record := filepath.Join(t.TempDir(), "character.json")

	var stdout, stderr bytes.Buffer

	// The first answer the player gives, the file is made behind his back.
	script := &onFirstRead{r: strings.NewReader(strings.Repeat("1\n", 4000)), do: func() {
		if err := os.WriteFile(record, []byte("already here\n"), 0o600); err != nil {
			t.Error(err)
		}
	}}

	if code := run([]string{"new", "--seed", "1", "-o", record}, noSeed(t), script, &stdout, &stderr); code != exitError {
		t.Fatalf("exit %d, want %d (stderr: %s)", code, exitError, stderr.String())
	}

	var character chargen.Character
	if err := json.Unmarshal(stdout.Bytes(), &character); err != nil || character.RNG.Seed != 1 {
		t.Errorf("stdout is not the record (seed %d, %v):\n%.200s", character.RNG.Seed, err, stdout.String())
	}

	if !strings.Contains(stderr.String(), "on stdout instead") {
		t.Errorf("stderr does not say where the record went: %s", stderr.String())
	}

	data, err := os.ReadFile(record) //nolint:gosec // a temp path this test named
	if err != nil || string(data) != "already here\n" {
		t.Errorf("the file that appeared was overwritten without --force (%v)", err)
	}
}

// onFirstRead runs do once, before the first read, and then reads from r.
type onFirstRead struct {
	r    io.Reader
	do   func()
	done bool
}

func (o *onFirstRead) Read(p []byte) (int, error) {
	if !o.done {
		o.done = true
		o.do()
	}

	return o.r.Read(p) //nolint:wrapcheck // a pass-through reader
}

// TestAbandonedSessionWritesNothing verifies the PRD's own sentence:
// "Interrupted interactive sessions produce no output file" (CLI sketch).
// The file must not exist — not exist and be empty, and not hold a partial
// record.
func TestAbandonedSessionWritesNothing(t *testing.T) {
	record := filepath.Join(t.TempDir(), "character.json")

	var stdout, stderr bytes.Buffer

	// Two answers and then the player leaves, so the session gets under
	// way before it is abandoned.
	script := strings.NewReader("1\n1\nq\ny\n")
	if code := run([]string{"new", "--seed", "1", "-o", record}, noSeed(t), script, &stdout, &stderr); code != exitError {
		t.Fatalf("abandoned session: exit %d, want %d", code, exitError)
	}

	if _, err := os.Stat(record); err == nil {
		t.Error("an abandoned session wrote an output file")
	}

	if stdout.Len() != 0 {
		t.Errorf("an abandoned session wrote to stdout: %s", stdout.String())
	}

	if !strings.Contains(stderr.String(), "abandoned") {
		t.Errorf("the player was not told the session was abandoned: %s", stderr.String())
	}

	// And is not told what he made, because he did not make one. The six
	// are rolled before the first question, so an abandoned session has a
	// UPP and no character — a summary of it would describe somebody who
	// was never finished.
	if strings.Contains(stderr.String(), "Character complete") {
		t.Errorf("an abandoned session was summarised: %s", stderr.String())
	}
}

// TestBatchOutputIsReadable verifies the tool can read back what it
// writes. batch emits JSONL, render and replay took a single record, and
// the two only met when a run happened to hold exactly one — so a run of
// one worked and a run of two did not, which is the shape of bug that
// survives testing and appears in use.
func TestBatchOutputIsReadable(t *testing.T) {
	runPath := filepath.Join(t.TempDir(), "run.jsonl")

	var stdout, stderr bytes.Buffer
	if code := run3(t, []string{"batch", "--count", "3", "--auto", "--seed", "5", "-o", runPath},
		&stdout, &stderr); code != exitOK {
		t.Fatalf("batch: exit %d, stderr: %s", code, stderr.String())
	}

	for _, tc := range []struct {
		name string
		args []string
		want int
	}{
		{name: "replay", args: []string{"replay", runPath}, want: 3},
		{name: "render", args: []string{"render", runPath}, want: 3},
		{name: "render history", args: []string{"render", "--history", runPath}, want: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run3(t, tc.args, &out, &errOut); code != exitOK {
				t.Fatalf("exit %d, stderr: %s", code, errOut.String())
			}

			if got := strings.Count(out.String(), marker(tc.name)); got != tc.want {
				t.Errorf("%d of %q in the output, want %d — one per record",
					got, marker(tc.name), tc.want)
			}
		})
	}
}

// marker is the line each subcommand emits once per record.
func marker(name string) string {
	switch name {
	case "replay":
		return "reproduced from seed"
	case "render history":
		return "# Generation Record"
	default:
		return "# Character Card"
	}
}

// run3 runs the CLI with no interactive input.
func run3(t *testing.T, args []string, stdout, stderr *bytes.Buffer) int {
	t.Helper()

	return run(args, noSeed(t), noInput(), stdout, stderr)
}

// TestUnreadableInputsSayWhy verifies the two ways a path can hold no
// record say which one it is. Both are things batch's own output invites:
// it writes a directory as readily as a file, and an interrupted run
// leaves an empty one. Neither should be reported against "record 1" of
// something that holds no records.
func TestUnreadableInputsSayWhy(t *testing.T) {
	dir := t.TempDir()

	empty := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ name, path, want string }{
		{name: "a directory", path: dir, want: "is a directory"},
		{name: "an empty file", path: empty, want: "no t5chargen character records"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run3(t, []string{"replay", tc.path}, &out, &errOut); code != exitError {
				t.Fatalf("exit %d, want %d", code, exitError)
			}

			if !strings.Contains(errOut.String(), tc.want) {
				t.Errorf("said %q, want it to mention %q", strings.TrimSpace(errOut.String()), tc.want)
			}

			if strings.Contains(errOut.String(), "record 1") {
				t.Errorf("blamed a record that does not exist: %s", strings.TrimSpace(errOut.String()))
			}
		})
	}
}

// TestAPartlyBrokenRunNamesTheRecord verifies a run that goes wrong says
// which of its records did. "Record 2 of 3" is actionable where a parse
// error about the whole file is not.
func TestAPartlyBrokenRunNamesTheRecord(t *testing.T) {
	runPath := filepath.Join(t.TempDir(), "run.jsonl")

	var stdout, stderr bytes.Buffer
	if code := run3(t, []string{"batch", "--count", "3", "--auto", "--seed", "5", "-o", runPath},
		&stdout, &stderr); code != exitOK {
		t.Fatalf("batch: exit %d", code)
	}

	data, err := os.ReadFile(runPath) //nolint:gosec // a temp path this test wrote
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	lines[1] = `{"upp":"777777"}`

	//nolint:gosec // G703: a path this test built from t.TempDir and wrote once already.
	if err := os.WriteFile(runPath, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if code := run3(t, []string{"replay", runPath}, &out, &errOut); code != exitError {
		t.Fatalf("exit %d, want %d", code, exitError)
	}

	if !strings.Contains(errOut.String(), "record 2") {
		t.Errorf("the failure does not name the record that broke: %s", errOut.String())
	}
}

// TestInteractiveSessionEndsWithItsCharacter verifies a session closes by
// saying what it made and where it went. Hundreds of questions answered
// and nothing said afterwards leaves a player wondering whether it worked.
func TestInteractiveSessionEndsWithItsCharacter(t *testing.T) {
	record := filepath.Join(t.TempDir(), "character.json")

	var stdout, stderr bytes.Buffer

	script := strings.NewReader(strings.Repeat("1\n", 4000))
	if code := run([]string{"new", "--seed", "3", "-o", record}, noSeed(t), script, &stdout, &stderr); code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}

	said := stderr.String()
	for _, want := range []string{"Character complete", "age ", record} {
		if !strings.Contains(said, want) {
			t.Errorf("the session ended without mentioning %q", want)
		}
	}

	// The summary belongs with the prompts. Without -o the record goes to
	// stdout, and a summary mixed into it would make that unparseable.
	if strings.Contains(stdout.String(), "Character complete") {
		t.Error("the summary was written to stdout, where the record goes")
	}
}

// TestAForeignRecordIsToldWhatStillWorks verifies the refusal is not a
// dead end. A record an older build wrote cannot be re-run — replay
// recomputes, and a newer engine does not reproduce an older one — but the
// record is not damaged, and saying only "cannot be re-run" leaves a
// reader with nowhere to go.
func TestAForeignRecordIsToldWhatStillWorks(t *testing.T) {
	record := filepath.Join(t.TempDir(), "old.json")

	var stdout, stderr bytes.Buffer
	if code := run3(t, []string{"new", "--auto", "--seed", "1", "-o", record}, &stdout, &stderr); code != exitOK {
		t.Fatalf("new: exit %d", code)
	}

	aged := readRecordFile(t, record)
	aged["schema_version"] = "0.0.1"
	writeRecordFile(t, record, aged)

	var out, errOut bytes.Buffer
	if code := run3(t, []string{"replay", record}, &out, &errOut); code != exitError {
		t.Fatalf("replay: exit %d, want %d", code, exitError)
	}

	said := errOut.String()
	for _, want := range []string{"different build", "schema_version", "not damaged", "render"} {
		if !strings.Contains(said, want) {
			t.Errorf("the refusal does not mention %q: %s", want, strings.TrimSpace(said))
		}
	}
}

// readRecordFile loads a record for a test to alter.
func readRecordFile(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path) //nolint:gosec // a temp path this test wrote
	if err != nil {
		t.Fatal(err)
	}

	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}

	return record
}

// writeRecordFile puts an altered record back.
func writeRecordFile(t *testing.T, path string, record map[string]any) {
	t.Helper()

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestReplayRefusalNamesTheWayOut verifies the refusal is not a dead end.
// A reader told only that his record was made by a different build has
// nowhere to go, and both places he can go — render, which still reads it,
// and the flag, which re-runs it — are named in the message rather than
// left to be discovered.
func TestReplayRefusalNamesTheWayOut(t *testing.T) {
	record := recordFromAnotherBuild(t)

	var stdout, stderr bytes.Buffer
	if code := run3(t, []string{"replay", record}, &stdout, &stderr); code != exitError {
		t.Fatalf("replay of a record from another build: exit %d, want %d", code, exitError)
	}

	for _, want := range []string{"render", "--ignore-provenance"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("the refusal %q does not mention %s", stderr.String(), want)
		}
	}
}

// TestReplayIgnoreProvenanceRunsItAnyway is the flag end to end. The
// record here differs from this build only in the version it claims, so
// the run reproduces: the flag has to reach a verdict on the generation
// rather than stop at the versions, and it has to say which check it
// waived while doing so.
func TestReplayIgnoreProvenanceRunsItAnyway(t *testing.T) {
	record := recordFromAnotherBuild(t)

	var stdout, stderr bytes.Buffer
	if code := run3(t, []string{"replay", "--ignore-provenance", record}, &stdout, &stderr); code != exitOK {
		t.Fatalf("replay --ignore-provenance: exit %d, want %d, stderr: %s", code, exitOK, stderr.String())
	}

	if !strings.Contains(stdout.String(), "reproduced from seed 1") {
		t.Errorf("replay said %q, which does not report the run it reproduced", stdout.String())
	}

	if !strings.Contains(stderr.String(), "anyway") {
		t.Errorf("stderr %q does not say the provenance check was waived", stderr.String())
	}
}

// recordFromAnotherBuild writes a real record and falsifies the one field
// that decides whether this build may re-run it. Falsifying the version
// rather than the run is deliberate: it is the case where the provenance
// gate and the generation disagree, which is what the flag exists for.
func recordFromAnotherBuild(t *testing.T) string {
	t.Helper()

	record := filepath.Join(t.TempDir(), "character.json")

	var stdout, stderr bytes.Buffer
	if code := run3(t, []string{"new", "--auto", "--seed", "1", "-o", record}, &stdout, &stderr); code != exitOK {
		t.Fatalf("new: exit %d, stderr: %s", code, stderr.String())
	}

	stored := readRecordFile(t, record)
	stored["engine_version"] = "0.2.0"
	writeRecordFile(t, record, stored)

	return record
}

// TestHelpIsAskedForNotBlunderedInto verifies `help` and its flag forms
// print to stdout and exit zero, while a misuse still prints the terse
// usage to stderr. The distinction is the point: usage accompanies every
// error, where prose would bury the line saying what went wrong.
func TestHelpIsAskedForNotBlunderedInto(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			if code := run([]string{arg}, noSeed(t), noInput(), &stdout, &stderr); code != exitOK {
				t.Fatalf("exit %d, stderr: %s", code, stderr.String())
			}

			if stderr.Len() != 0 {
				t.Errorf("help wrote to stderr: %s", stderr.String())
			}

			// The three things §6 of docs/BETA_READINESS.md asked for,
			// and the one place the --auto explanation is allowed to
			// live: a prompt is replay-compared, help text is not.
			for _, want := range []string{
				"examples:",
				"troubleshooting:",
				"report a problem:",
				"https://github.com/philoserf/t5chargen/issues",
				"stability:",
				"every automatic character is a Citizen",
			} {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("help does not mention %q", want)
				}
			}
		})
	}

	// A misuse gets the short form, not the essay.
	var stdout, stderr bytes.Buffer

	if code := run(nil, noSeed(t), noInput(), &stdout, &stderr); code != exitUsage {
		t.Fatalf("no arguments: exit %d, want %d", code, exitUsage)
	}

	if strings.Contains(stderr.String(), "troubleshooting:") {
		t.Error("a usage error printed the full help")
	}
}
