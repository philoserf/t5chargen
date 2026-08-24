package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		{"render txt deferred", []string{"render", "--format", "txt", noSchema}, exitError},
		{"render unknown format", []string{"render", "--format", "html", noSchema}, exitUsage},
		{"batch without --auto", []string{"batch", "--count", "2"}, exitUsage},
		{"batch without --count", []string{"batch", "--auto"}, exitUsage},
		{"batch negative count", []string{"batch", "--auto", "--count", "-2"}, exitUsage},
		{"batch stray arguments", []string{"batch", "--auto", "--count", "2", "out.jsonl"}, exitUsage},
		{"batch unknown career", []string{"batch", "--auto", "--count", "2", "--career", "bogus"}, exitUsage},
		{"batch career unavailable", []string{"batch", "--auto", "--count", "2", "--career", "Craftsman"}, exitUsage},
		{"batch outlived current year", []string{"batch", "--auto", "--count", "2", "--current-year", "30"}, exitUsage},
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
