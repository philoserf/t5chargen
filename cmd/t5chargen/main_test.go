package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	if code := run([]string{"new", "--auto", "--seed", "1"}, noSeed(t), &stdout, &stderr); code != exitOK {
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

// TestNewSeedZero verifies --seed 0 is honored as an explicit seed rather
// than triggering the default seed source.
func TestNewSeedZero(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run([]string{"new", "--auto", "--seed", "0"}, noSeed(t), &stdout, &stderr); code != exitOK {
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

	if code := run([]string{"new", "--auto"}, fixedSeed(7), &stdout, &stderr); code != exitOK {
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
	if code := run(args, noSeed(t), &stdout, &stderr); code != exitOK {
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

	if code := run([]string{"new", "--auto", "--seed", "1", "-o", path}, noSeed(t), &stdout, &stderr); code != exitOK {
		t.Fatalf("first write: exit %d, stderr: %s", code, stderr.String())
	}

	if stdout.Len() != 0 {
		t.Error("-o also wrote to stdout")
	}

	stderr.Reset()

	if code := run([]string{"new", "--auto", "--seed", "2", "-o", path}, noSeed(t), &stdout, &stderr); code != exitError {
		t.Fatalf("overwrite without --force: exit %d, want %d", code, exitError)
	}

	if !strings.Contains(stderr.String(), "--force") {
		t.Errorf("overwrite refusal does not mention --force: %s", stderr.String())
	}

	forceArgs := []string{"new", "--auto", "--seed", "2", "-o", path, "--force"}
	if code := run(forceArgs, noSeed(t), &stdout, &stderr); code != exitOK {
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
	if code := run(args, noSeed(t), &stdout, &stderr); code != exitOK {
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
	if code := run(args, noSeed(t), &stdout, &stderr); code != exitOK {
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

	if code := run([]string{"new", "--auto", "--seed", "1", "-o", path}, noSeed(t), &stdout, &stderr); code != exitOK {
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

			if code := run(tt.args, noSeed(t), &out, &errOut); code != exitOK {
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
func TestErrors(t *testing.T) {
	garbage := filepath.Join(t.TempDir(), "garbage.json")
	if err := os.WriteFile(garbage, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	noSchema := filepath.Join(t.TempDir(), "noschema.json")
	if err := os.WriteFile(noSchema, []byte(`{"upp":"777777"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args []string
		code int
	}{
		{"no arguments", nil, exitUsage},
		{"unknown subcommand", []string{"bogus"}, exitUsage},
		{"unknown flag", []string{"new", "--bogus"}, exitUsage},
		{"new without --auto", []string{"new", "--seed", "1"}, exitUsage},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			if code := run(tt.args, fixedSeed(1), &stdout, &stderr); code != tt.code {
				t.Errorf("exit %d, want %d (stderr: %s)", code, tt.code, stderr.String())
			}
		})
	}
}
