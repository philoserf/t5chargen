// Command t5chargen generates Traveller5 characters. See docs/PRD.md.
//
// Implemented subcommands: new, render. Planned: batch, replay.
package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/render"
	"github.com/philoserf/t5chargen/world"
)

// Exit codes: 0 success, 1 operational error, 2 usage error (the flag
// package's own convention). Replay will later use non-zero for the first
// event mismatch (docs/PRD.md, Replay and provenance contract).
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

const usage = `usage:
  t5chargen new --auto [--seed N] [--name X] [--career citizen] [--homeworld "UWP TC..."] [-o file] [--force]
  t5chargen render character.json [--format md] [--history]
`

func main() {
	os.Exit(run(os.Args[1:], randomSeed, os.Stdout, os.Stderr))
}

// randomSeed draws a seed from the OS entropy source. This is the one
// deliberate exception to the repo's no-unseeded-randomness rule, which is
// engine-scoped: the CLI may pick the seed, and the chosen seed is recorded
// in the character's rng provenance so replay stays exact.
func randomSeed() (uint64, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, fmt.Errorf("drawing random seed: %w", err)
	}

	return binary.LittleEndian.Uint64(buf[:]), nil
}

// run dispatches the subcommand. seedFn supplies the seed when --seed is
// not given; tests inject a deterministic one.
func run(args []string, seedFn func() (uint64, error), stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)

		return exitUsage
	}

	switch args[0] {
	case "new":
		return runNew(args[1:], seedFn, stdout, stderr)
	case "render":
		return runRender(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "t5chargen: unknown subcommand %q\n%s", args[0], usage)

		return exitUsage
	}
}

// runNew generates a character and writes its JSON record to stdout, or to
// -o file (docs/PRD.md, CLI sketch: "new writes JSON to stdout unless -o").
// isUsageError reports whether a generation failure is the caller's
// fault rather than the engine's. The engine is the single validator for
// careers, UWPs, and trade classifications.
func isUsageError(err error) bool {
	return errors.Is(err, chargen.ErrUnknownCareer) ||
		errors.Is(err, chargen.ErrCareerUnavailable) ||
		errors.Is(err, world.ErrInvalidUWP) ||
		errors.Is(err, world.ErrUnknownTC) ||
		errors.Is(err, world.ErrDuplicateTC)
}

func runNew(args []string, seedFn func() (uint64, error), stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seed := flags.Uint64("seed", 0, "RNG seed (default: drawn from OS entropy)")
	name := flags.String("name", "", "character name (blank by default)")
	careerFlag := flags.String("career", "", "force the first career")
	homeworldFlag := flags.String("homeworld", "",
		`homeworld as "UWP" or "UWP TC TC..." (for example "A788899-C Ph Pa Ri"); `+
			`skills come from the trade classifications, so a bare UWP grants none (default: Regina)`)
	auto := flags.Bool("auto", false, "apply the fixed default policy (POLICY.md) to every choice")
	out := flags.String("o", "", "output file (default: stdout)")
	force := flags.Bool("force", false, "overwrite an existing output file")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "t5chargen new: unexpected arguments %q (use -o for an output file)\n%s", flags.Args(), usage)

		return exitUsage
	}

	// Interactive mode is the PRD's default; it lands with milestone 5.
	// Refusing is honest — silently substituting the auto policy would
	// misrepresent who decided.
	if !*auto {
		fmt.Fprintln(stderr, "t5chargen new: interactive mode is not yet implemented (milestone 5); use --auto")

		return exitUsage
	}

	if err := resolveSeed(flags, seed, seedFn); err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	character, err := chargen.Generate(chargen.Options{
		Seed:      *seed,
		Name:      *name,
		Career:    canonicalCareer(*careerFlag),
		Homeworld: parseHomeworldFlag(*homeworldFlag),
		Decider:   chargen.DefaultPolicy{},
	})
	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		if isUsageError(err) {
			return exitUsage
		}

		return exitError
	}

	return emitRecord(character, *out, *force, stdout, stderr)
}

// emitRecord marshals the record and writes it to stdout or the output
// file.
func emitRecord(character chargen.Character, out string, force bool, stdout, stderr io.Writer) int {
	data, err := json.MarshalIndent(character, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: encoding character: %v\n", err)

		return exitError
	}

	data = append(data, '\n')

	if out == "" {
		_, err = stdout.Write(data)
	} else {
		err = writeFile(out, data, force)
	}

	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	return exitOK
}

// resolveSeed draws a seed from seedFn when --seed was not given. --seed 0
// is a valid explicit seed; only an absent flag falls back to seedFn.
func resolveSeed(flags *flag.FlagSet, seed *uint64, seedFn func() (uint64, error)) error {
	seedSet := false

	flags.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			seedSet = true
		}
	})

	if seedSet {
		return nil
	}

	drawn, err := seedFn()
	if err != nil {
		return err
	}

	*seed = drawn

	return nil
}

// parseHomeworldFlag splits a --homeworld value into a Homeworld: the
// first field is the UWP, the rest are trade classifications. Validation
// is the engine's; an empty flag leaves the zero value for the default.
func parseHomeworldFlag(value string) world.Homeworld {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return world.Homeworld{}
	}

	return world.Homeworld{UWP: fields[0], TradeClassifications: fields[1:]}
}

// canonicalCareer maps a case-insensitive --career value to its canonical
// Book 1 name; unknown names pass through unchanged for the engine — the
// single validator — to reject.
func canonicalCareer(name string) string {
	for _, available := range career.Available() {
		if strings.EqualFold(available, name) {
			return available
		}
	}

	return name
}

// writeFile writes the record to path. "Existing files are never
// overwritten without --force." (docs/PRD.md, CLI sketch) — creation is
// exclusive unless force allows truncation.
func writeFile(path string, data []byte, force bool) error {
	mode := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		mode = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	file, err := os.OpenFile(path, mode, 0o644) //nolint:gosec // G304: user-supplied output path is the CLI contract.
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%s: %w", path, errExists)
		}

		return fmt.Errorf("writing %s: %w", path, err)
	}

	if _, err := file.Write(data); err != nil {
		file.Close() //nolint:errcheck,gosec // write error takes precedence

		return fmt.Errorf("writing %s: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// runRender renders a character JSON record as a Markdown sheet, or as the
// history transcript with --history (docs/PRD.md goal 4, goal 5).
func runRender(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", "md", "output format: md (txt planned)")
	history := flags.Bool("history", false, "render the generation-record transcript instead of the sheet")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if flags.NArg() != 1 {
		fmt.Fprintf(stderr, "t5chargen render: want exactly one character.json argument\n%s", usage)

		return exitUsage
	}

	switch *format {
	case "md":
	case "txt":
		fmt.Fprintln(stderr, "t5chargen render: format txt is not yet implemented")

		return exitError
	default:
		fmt.Fprintf(stderr, "t5chargen render: unknown format %q\n", *format)

		return exitUsage
	}

	character, err := readCharacter(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	if *history {
		fmt.Fprint(stdout, render.History(character))
	} else {
		fmt.Fprint(stdout, render.Sheet(character))
	}

	return exitOK
}

// errExists reports an output file that already exists ("Existing files are
// never overwritten without --force", docs/PRD.md CLI sketch).
var errExists = errors.New("exists; use --force to overwrite")

// errNotCharacter reports JSON that parsed but is not a character record.
var errNotCharacter = errors.New("not a t5chargen character record (no schema_version)")

// readCharacter loads and minimally validates a character record.
func readCharacter(path string) (chargen.Character, error) {
	var character chargen.Character

	data, err := os.ReadFile(path) //nolint:gosec // G304: user-supplied input path is the CLI contract.
	if err != nil {
		return character, fmt.Errorf("reading %s: %w", path, err)
	}

	if err := json.Unmarshal(data, &character); err != nil {
		return character, fmt.Errorf("parsing %s: %w", path, err)
	}

	if character.SchemaVersion == "" {
		return character, fmt.Errorf("parsing %s: %w", path, errNotCharacter)
	}

	return character, nil
}
