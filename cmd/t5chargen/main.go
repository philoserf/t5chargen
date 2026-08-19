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

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/render"
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
  t5chargen new [--seed N] [--name X] [-o file] [--force]
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
func runNew(args []string, seedFn func() (uint64, error), stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seed := flags.Uint64("seed", 0, "RNG seed (default: drawn from OS entropy)")
	name := flags.String("name", "", "character name (blank by default)")
	out := flags.String("o", "", "output file (default: stdout)")
	force := flags.Bool("force", false, "overwrite an existing output file")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	// --seed 0 is a valid explicit seed; only fall back to seedFn when the
	// flag was not given at all.
	seedSet := false

	flags.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			seedSet = true
		}
	})

	if !seedSet {
		drawn, err := seedFn()
		if err != nil {
			fmt.Fprintf(stderr, "t5chargen: %v\n", err)

			return exitError
		}

		*seed = drawn
	}

	character := chargen.Generate(*seed, *name)

	data, err := json.MarshalIndent(character, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: encoding character: %v\n", err)

		return exitError
	}

	data = append(data, '\n')

	if *out == "" {
		_, err = stdout.Write(data)
	} else {
		err = writeFile(*out, data, *force)
	}

	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	return exitOK
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
