// Command t5chargen generates Traveller5 characters. See docs/PRD.md.
//
// Implemented subcommands: new, batch, render, replay.
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/philoserf/t5chargen/calendar"
	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/interactive"
	"github.com/philoserf/t5chargen/render"
	"github.com/philoserf/t5chargen/world"
)

// Exit codes: 0 success, 1 operational error, 2 usage error (the flag
// package's own convention). A replay divergence is an operational error:
// the command worked and the answer is no (docs/PRD.md, Replay and
// provenance contract: "exits non-zero at the first mismatch").
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

const usage = `usage:
  t5chargen new [--auto] [--seed N] [--name X] [--career citizen] [--homeworld "UWP TC..."|random]
                [--current-year 1105] [-o file] [--force]
                (without --auto the player answers each choice; --auto applies POLICY.md)
  t5chargen batch --count N --auto [--seed N] [--name X] [--career citizen]
                  [--homeworld "UWP TC..."|random] [--current-year 1105] [-o dir/|file.jsonl] [--force]
  t5chargen render [--history] character.json
  t5chargen replay [--ignore-provenance] character.json
`

func main() {
	os.Exit(run(os.Args[1:], randomSeed, os.Stdin, os.Stdout, os.Stderr))
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
// not given and stdin answers interactive prompts; tests inject a
// deterministic seed and a scripted script.
func run(args []string, seedFn func() (uint64, error), stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)

		return exitUsage
	}

	switch args[0] {
	case "new":
		return runNew(args[1:], seedFn, stdin, stdout, stderr)
	case "batch":
		return runBatch(args[1:], seedFn, stdout, stderr)
	case "render":
		return runRender(args[1:], stdout, stderr)
	case "replay":
		return runReplay(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "t5chargen: unknown subcommand %q\n%s", args[0], usage)

		return exitUsage
	}
}

// isUsageError reports whether a generation failure is the caller's
// fault rather than the engine's. The engine is the single validator for
// careers, UWPs, trade classifications, and the current year.
//
// checkCurrentYear catches the years that are not years at all, but a year
// the character has outlived is only knowable once the age is (birthdate.go
// runs last), so that one comes back from the engine — and it is still the
// caller's --current-year that is wrong.
func isUsageError(err error) bool {
	return errors.Is(err, chargen.ErrUnknownCareer) ||
		errors.Is(err, chargen.ErrCareerUnavailable) ||
		errors.Is(err, chargen.ErrCurrentYear) ||
		errors.Is(err, world.ErrInvalidUWP) ||
		errors.Is(err, world.ErrUnknownTC) ||
		errors.Is(err, world.ErrDuplicateTC)
}

// runNew generates a character and writes its JSON record to stdout, or to
// -o file (docs/PRD.md, CLI sketch: "new writes JSON to stdout unless -o").
func runNew(args []string, seedFn func() (uint64, error), stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seed := flags.Uint64("seed", 0, "RNG seed (default: drawn from OS entropy)")
	name := flags.String("name", "", "character name (blank by default)")
	careerFlag := flags.String("career", "", "force the first career")
	homeworldFlag := flags.String("homeworld", "", homeworldUsage)
	currentYear := flags.Int("current-year", calendar.DefaultYear,
		"Imperial year adventuring begins in, which fixes the birth year (Book 1 p. 58)")
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

	if code := checkFlags("new", *currentYear, *name, stderr); code != exitOK {
		return code
	}

	if err := resolveSeed(flags, seed, seedFn); err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	options := generateOptions(*seed, *name, *careerFlag, *homeworldFlag, *currentYear)

	player := openSession(&options, *auto, stdin, stderr)

	character, err := chargen.Generate(options)
	if err != nil {
		// An abandoned session is not a failure to report as one, and
		// it must leave nothing behind: "Interrupted interactive
		// sessions produce no output file" (docs/PRD.md, CLI sketch).
		if errors.Is(err, interactive.ErrAbandoned) {
			fmt.Fprintln(stderr, "t5chargen new: abandoned; no character written")

			return exitError
		}

		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		if isUsageError(err) {
			return exitUsage
		}

		return exitError
	}

	code := emitRecord(character, *out, *force, stdout, stderr)
	if code == exitOK {
		closeSession(player, character, *out, stderr)
	}

	return code
}

// runBatch generates a run of characters for NPC use: "batch emits JSONL
// (or one file per character with -o dir), requires --auto, and derives
// each member's seed from the base seed + index, recorded in each record"
// (docs/PRD.md, CLI sketch).
func runBatch(args []string, seedFn func() (uint64, error), stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("batch", flag.ContinueOnError)
	flags.SetOutput(stderr)
	count := flags.Int("count", 0, "how many characters to generate")
	seed := flags.Uint64("seed", 0, "base RNG seed; member i uses base+i (default: drawn from OS entropy)")
	name := flags.String("name", "", "character name, applied to every member (blank by default)")
	careerFlag := flags.String("career", "", "force the first career")
	homeworldFlag := flags.String("homeworld", "", homeworldUsage)
	currentYear := flags.Int("current-year", calendar.DefaultYear,
		"Imperial year adventuring begins in, which fixes the birth year (Book 1 p. 58)")
	auto := flags.Bool("auto", false, "required: batch has no interactive mode")
	out := flags.String("o", "",
		"output directory, named with a trailing / and created if missing (one file per character), "+
			"or a .jsonl file (default: JSONL on stdout)")
	force := flags.Bool("force", false, "overwrite existing output files")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if code := checkBatchFlags(flags, *count, *auto, *currentYear, stderr); code != exitOK {
		return code
	}

	if err := resolveSeed(flags, seed, seedFn); err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	characters, err := generateBatch(*count, *seed,
		generateOptions(*seed, *name, *careerFlag, *homeworldFlag, *currentYear))
	if err != nil {
		fmt.Fprintf(stderr, "t5chargen batch: %v\n", err)

		if isUsageError(err) {
			return exitUsage
		}

		return exitError
	}

	return emitBatch(characters, *out, *force, stdout, stderr)
}

// checkBatchFlags validates the flags batch does not share with new.
func checkBatchFlags(flags *flag.FlagSet, count int, auto bool, currentYear int, stderr io.Writer) int {
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "t5chargen batch: unexpected arguments %q (use -o for output)\n%s", flags.Args(), usage)

		return exitUsage
	}

	// "batch ... requires --auto" (docs/PRD.md, CLI sketch). Unlike new,
	// this is not a milestone deferral: a run of characters has nobody to
	// ask, so the flag is the caller acknowledging the policy decides.
	if !auto {
		fmt.Fprintln(stderr, "t5chargen batch: --auto is required (a batch has nobody to ask)")

		return exitUsage
	}

	if count < 1 {
		fmt.Fprintf(stderr, "t5chargen batch: --count %d is not a number of characters\n", count)

		return exitUsage
	}

	return checkCurrentYear("batch", currentYear, stderr)
}

// batchAllocHint bounds generateBatch's initial allocation. It is not a
// limit on --count: the slice grows past it.
const batchAllocHint = 1024

// generateBatch derives each member's seed from the base seed plus its
// index, so every member is independently replayable from the record it
// lands in — the seed it was generated from is the seed it reports.
//
// Nothing is written until every member has generated. A batch that fails
// halfway is a batch that did not happen, rather than a directory holding
// some of what was asked for.
func generateBatch(count int, base uint64, opts chargen.Options) ([]chargen.Character, error) {
	// The capacity hint is bounded, not --count itself: a cap on the count
	// would be a limit docs/PRD.md does not state, and this repo does not
	// invent those. What is bounded is the request — a mistyped
	// --count 100000000000 asks for a 54 TB reservation, which macOS
	// happily grants lazily and a system without overcommit refuses
	// outright. Neither is a thing to ask for, and the slice grows to
	// whatever a real run needs.
	characters := make([]chargen.Character, 0, min(count, batchAllocHint))

	for i := range count {
		opts.Seed = base + uint64(i)

		character, err := chargen.Generate(opts)
		if err != nil {
			return nil, fmt.Errorf("character %d of %d (seed %d): %w", i+1, count, opts.Seed, err)
		}

		characters = append(characters, character)
	}

	return characters, nil
}

// emitBatch writes the run as JSONL, or as one file per character when -o
// names a directory. The PRD spells the flag "-o dir|file.jsonl", so the
// path itself says which was meant: see batchDir.
func emitBatch(characters []chargen.Character, out string, force bool, stdout, stderr io.Writer) int {
	dir, isDir, err := batchDir(out)
	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	if isDir {
		return emitBatchFiles(characters, dir, force, stderr)
	}

	var buf bytes.Buffer

	for _, character := range characters {
		// One record per line, so the stream stays greppable and a
		// consumer can read it a character at a time.
		line, err := json.Marshal(character)
		if err != nil {
			fmt.Fprintf(stderr, "t5chargen batch: encoding seed %d: %v\n", character.RNG.Seed, err)

			return exitError
		}

		buf.Write(line)
		buf.WriteByte('\n')
	}

	if out == "" {
		_, err = stdout.Write(buf.Bytes())
	} else {
		err = writeFile(out, buf.Bytes(), force)
	}

	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	return exitOK
}

// emitBatchFiles writes one indented record per character, named for the
// seed that produced it so the file says how to reproduce itself.
//
// Every path is checked before any is written. Writing them one at a time
// would leave a directory holding the first half of a run that failed on
// the tenth file — the same reason generateBatch finishes before anything
// is emitted. The check races anything else writing the directory
// meanwhile, which is why writeFile still refuses on its own with O_EXCL;
// this pass is so the common case fails before it has made a mess.
func emitBatchFiles(characters []chargen.Character, dir string, force bool, stderr io.Writer) int {
	if !force {
		for _, character := range characters {
			path := batchPath(dir, character)
			if _, err := os.Stat(path); err == nil {
				fmt.Fprintf(stderr, "t5chargen: %s: %v\n", path, errExists)

				return exitError
			}
		}
	}

	for _, character := range characters {
		data, err := json.MarshalIndent(character, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "t5chargen batch: encoding seed %d: %v\n", character.RNG.Seed, err)

			return exitError
		}

		if err := writeFile(batchPath(dir, character), append(data, '\n'), force); err != nil {
			fmt.Fprintf(stderr, "t5chargen: %v\n", err)

			return exitError
		}
	}

	return exitOK
}

// batchDir decides whether -o named the "dir" or the "file.jsonl" half of
// the PRD's "-o dir|file.jsonl", and reports the directory when it is the
// former.
//
// An existing directory is one. So is any path written with a trailing
// separator: "-o npcs/" is the caller declaring a directory, and a
// declaration is honoured by creating it rather than refused for not
// existing yet — otherwise the flag would only work for a directory the
// caller had already made, and "-o npcs" (no slash, nothing there) would
// quietly deposit the whole run in a single file named npcs.
func batchDir(out string) (string, bool, error) {
	if out == "" {
		return "", false, nil
	}

	if os.IsPathSeparator(out[len(out)-1]) {
		dir := filepath.Clean(out)
		//nolint:gosec // G301: an output directory the caller named; 0755 matches mkdir(1).
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", false, fmt.Errorf("creating %s: %w", dir, err)
		}

		return dir, true, nil
	}

	if info, err := os.Stat(out); err == nil && info.IsDir() {
		return out, true, nil
	}

	return out, false, nil
}

// batchPath names a batch member's file for the seed that produced it.
func batchPath(dir string, character chargen.Character) string {
	return filepath.Join(dir, fmt.Sprintf("character-%d.json", character.RNG.Seed))
}

// openSession puts a player behind the engine unless --auto asked for the
// policy instead. Whichever answers, the record says which (FR10).
func openSession(options *chargen.Options, auto bool, stdin io.Reader, prompts io.Writer) *interactive.Decider {
	if auto {
		return nil
	}

	player := interactive.New(stdin, prompts)
	options.Decider = player

	return player
}

// closeSession says what an interactive run made and where it went.
// Hundreds of questions answered and nothing said afterwards leaves a
// player wondering whether it worked.
//
// Called only once the record is written, because "written to" is a claim
// about a file that exists: announcing it first would tell a player his
// lifepath was saved and then print the error saying it was not.
//
// On stderr with the prompts, never on stdout: without -o the record
// itself goes there, and a summary mixed into it would make the output
// unparseable.
func closeSession(player *interactive.Decider, character chargen.Character, out string, stderr io.Writer) {
	if player == nil {
		return
	}

	// Read off the record rather than off the session's own running
	// count. The session keeps a shadow of the character because there is
	// no character to read while it is being made — but by now there is,
	// and it is the one that was written. Summarising from the shadow is
	// how the closing line came to show a Str of 0 for a character whose
	// Str is 1: the shadow missed the reset that follows aging (p. 89),
	// and nothing compared the two.
	summary := fmt.Sprintf("%s · age %d · %d %s",
		character.UPP, character.Age, len(character.Skills), plural(len(character.Skills), "skill"))

	// "the Character is dead (and all efforts in this particular
	// character creation process are lost)" (p. 69). The record is kept,
	// but a summary that read like any other would not say what happened.
	if character.Dead {
		summary += " · dead"
	}

	fmt.Fprintf(stderr, "\n%s\n  %s\n", interactive.Rule("Character complete"), summary)

	if out != "" {
		fmt.Fprintf(stderr, "  written to %s\n", out)
	}
}

// plural is the crudest possible pluralisation, and enough here.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}

	return word + "s"
}

// generateOptions assembles the engine options the flags describe, shared
// by new and batch so the two cannot drift apart in how they read them.
// batch passes the base seed and generateBatch replaces it per member.
func generateOptions(seed uint64, name, careerFlag, homeworldFlag string, currentYear int) chargen.Options {
	return chargen.Options{
		Seed:          seed,
		Name:          name,
		Career:        canonicalCareer(careerFlag),
		Homeworld:     parseHomeworldFlag(homeworldFlag),
		RollHomeworld: isRandomHomeworld(homeworldFlag),
		CurrentYear:   currentYear,
		Decider:       chargen.DefaultPolicy{},
	}
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

// checkCurrentYear rejects a year that is not one. The engine reads a zero
// CurrentYear as "not provided" and takes p. 58's default, the way an
// all-zero Homeworld takes the tool-owned one; a referee who typed a year
// meant it, so an explicit 0 is a usage error here rather than a silent
// fallback to 1105.
//
// cmd names the subcommand that read the flag, because new and batch both
// take --current-year and the diagnostic has to say which one refused.
func checkCurrentYear(cmd string, year int, stderr io.Writer) int {
	if year >= 1 {
		return exitOK
	}

	fmt.Fprintf(stderr, "t5chargen %s: --current-year %d is not an Imperial year\n%s", cmd, year, usage)

	return exitUsage
}

// randomHomeworld is the --homeworld value that determines the world on
// chart B rather than naming one: "Select or determine a Homeworld"
// (p. 56).
const randomHomeworld = "random"

// homeworldUsage documents --homeworld for both new and batch. Shared so
// the flag cannot be described one way and accepted another: batch reads
// it through the same generateOptions.
const homeworldUsage = `homeworld as "UWP" or "UWP TC TC..." ` +
	`(for example "A788899-C Ph Pa Ri"), or "` + randomHomeworld +
	`" to determine it on chart B; skills come from the trade ` +
	`classifications, so a bare UWP grants none (default: Regina)`

// isRandomHomeworld reports whether the flag asks for a chart B roll.
func isRandomHomeworld(flag string) bool {
	return strings.EqualFold(strings.TrimSpace(flag), randomHomeworld)
}

// parseHomeworldFlag splits a --homeworld value into a Homeworld: the
// first field is the UWP, the rest are trade classifications. Validation
// is the engine's; an empty flag leaves the zero value for the default.
func parseHomeworldFlag(value string) world.Homeworld {
	if isRandomHomeworld(value) {
		return world.Homeworld{} // determined on chart B, not supplied
	}

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
// The record is written whole or not at all: a truncating write that
// fails partway leaves a half-written file where a valid record was, and
// a record is the one artifact this tool exists to produce. Writing beside
// the target and renaming over it makes the replacement atomic.
func writeFile(path string, data []byte, force bool) error {
	if !force {
		if err := claimPath(path); err != nil {
			return err
		}
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".t5chargen-*")
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	defer os.Remove(tmp.Name()) //nolint:errcheck // best effort; the rename below usually beats it

	if _, err := tmp.Write(data); err != nil {
		tmp.Close() //nolint:errcheck,gosec // write error takes precedence

		return fmt.Errorf("writing %s: %w", path, err)
	}

	// Synced before the rename: without it the rename can be durable
	// while the bytes it points at are not.
	if err := tmp.Sync(); err != nil {
		tmp.Close() //nolint:errcheck,gosec // sync error takes precedence

		return fmt.Errorf("writing %s: %w", path, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	// CreateTemp makes the file 0o600; the record is not a secret and the
	// old path wrote 0o644.
	//nolint:gosec // G302: a character record is not a secret, and the
	// non-atomic write this replaces created 0o644.
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// claimPath refuses a path that already holds a file, and reserves it if
// it does not. "Existing files are never overwritten without --force."
// (docs/PRD.md, CLI sketch) — the exclusive create is the existence check,
// not the write, so writeFile can still replace the file atomically.
func claimPath(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644) //nolint:gosec // G304: the CLI contract.
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%s: %w", path, errExists)
		}

		return fmt.Errorf("writing %s: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// runRender renders a character JSON record as a Markdown sheet, or as the
// history transcript with --history (docs/PRD.md goal 4, goal 5).
//
// Markdown is the only format. A --format flag offering "md" and
// refusing "txt" was carried for a while against the PRD's CLI sketch;
// the sketch dropped txt rather than grow a second set of golden sheets
// for output with the emphasis markers stripped.
func runRender(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	flags.SetOutput(stderr)
	history := flags.Bool("history", false, "render the generation-record transcript instead of the sheet")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if flags.NArg() != 1 {
		fmt.Fprintf(stderr, "t5chargen render: want exactly one character.json argument\n%s", usage)

		return exitUsage
	}

	characters, err := readCharacters(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	for i, character := range characters {
		// A run of sheets needs something between them; one does not,
		// because nothing follows it.
		if i > 0 {
			fmt.Fprint(stdout, "\n---\n\n")
		}

		if *history {
			fmt.Fprint(stdout, render.History(character))
		} else {
			fmt.Fprint(stdout, render.Sheet(character))
		}
	}

	return exitOK
}

// runReplay re-runs a record and verifies it reproduces itself:
// "t5chargen replay character.json exits non-zero at the first mismatch,
// reporting the diverging event's sequence number" (docs/PRD.md, Replay
// and provenance contract).
func runReplay(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("replay", flag.ContinueOnError)
	flags.SetOutput(stderr)
	// Deliberately not --force, which in this CLI means "overwrite the
	// file I am about to write". Replay writes nothing; what this waives
	// is a check, and the name says which one.
	ignoreProvenance := flags.Bool("ignore-provenance", false,
		"re-run a record made by a different build, and report where it disagrees")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if flags.NArg() != 1 {
		fmt.Fprintf(stderr, "t5chargen replay: want exactly one character.json argument\n%s", usage)

		return exitUsage
	}

	characters, err := readCharacters(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	for i, character := range characters {
		name := recordName(flags.Arg(0), i, len(characters))

		if code := replayOne(character, name, flags.Arg(0), *ignoreProvenance, stdout, stderr); code != exitOK {
			return code
		}
	}

	return exitOK
}

// replayOne verifies a single record and reports the outcome.
//
// With --ignore-provenance the mismatch is announced rather than fatal, so
// the run happens and the reader is told what he is looking at. A record
// that then reproduces exactly is reported as reproducing: the versions
// disagreeing while the generation does not is a true and useful answer,
// not a qualified success.
func replayOne(
	character chargen.Character, name, path string, ignoreProvenance bool, stdout, stderr io.Writer,
) int {
	_, err := chargen.Replay(character)

	// Replay stops at the provenance gate before rolling anything, so
	// asking it first costs nothing and keeps the check itself in the
	// engine rather than duplicated here.
	if err != nil && ignoreProvenance && errors.Is(err, chargen.ErrReplayProvenance) {
		fmt.Fprintf(stderr, "t5chargen replay: %s: %v\n", name, err)
		fmt.Fprintf(stderr, "  Re-running it anyway, because --ignore-provenance was given.\n")

		_, err = chargen.ReplayIgnoringProvenance(character)
	}

	if err != nil {
		fmt.Fprintf(stderr, "t5chargen replay: %s: %v\n", name, err)

		// A record from another build is not a damaged one, and saying
		// only that it cannot be re-run leaves a reader with nowhere to
		// go. Replay re-runs the engine, so a record an older one wrote
		// cannot be reproduced by a newer one — but the record itself is
		// untouched and still reads, and --ignore-provenance will say
		// where the two builds part company.
		if errors.Is(err, chargen.ErrReplayProvenance) {
			fmt.Fprintf(stderr,
				"  The record is not damaged: t5chargen render %s still reads it,\n"+
					"  and t5chargen replay --ignore-provenance %s re-runs it anyway.\n", path, path)
		}

		return exitError
	}

	fmt.Fprintf(stdout, "replayed %s: %d events reproduced from seed %d\n",
		name, len(character.Events), character.RNG.Seed)

	return exitOK
}

// recordName names one record for a message: the file where it holds a
// single character, and the file and position where it holds a run.
func recordName(path string, i, count int) string {
	if count == 1 {
		return path
	}

	return fmt.Sprintf("%s record %d of %d", path, i+1, count)
}

// errExists reports an output file that already exists ("Existing files are
// never overwritten without --force", docs/PRD.md CLI sketch).
var errExists = errors.New("exists; use --force to overwrite")

// errIsDirectory reports a directory where a record was wanted — the
// other shape batch writes, named one file at a time.
var errIsDirectory = errors.New("is a directory; name one of the records inside it")

// errNotCharacter reports JSON that parsed but is not a character record.
var errNotCharacter = errors.New("not a t5chargen character record (no schema_version)")

// errNoRecords reports a file that held no JSON at all, which is a
// different mistake from a record that parsed and was the wrong shape.
var errNoRecords = errors.New("no t5chargen character records in file")

// readCharacters loads a file as a run of character records.
//
// One record and a run of them are the same thing here, a run of one,
// because batch writes JSONL and a JSONL file of a single record is also
// a single record — so a tool that read only one would work while a run
// was being tested and fail once it held two. render and replay both take
// whatever batch wrote, in either of the forms it writes.
func readCharacters(path string) ([]chargen.Character, error) {
	file, err := os.Open(path) //nolint:gosec // G304: user-supplied input path is the CLI contract.
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	// batch writes a directory of records as readily as a file of them,
	// so one will be handed to these commands. Saying that plainly beats
	// letting the decoder report a read failure against "record 1" of
	// something that holds no records at all.
	if info, err := file.Stat(); err == nil && info.IsDir() {
		_ = file.Close()

		return nil, fmt.Errorf("%s: %w", path, errIsDirectory)
	}

	defer func() { _ = file.Close() }()

	var characters []chargen.Character

	decoder := json.NewDecoder(file)

	for {
		var character chargen.Character

		switch err := decoder.Decode(&character); {
		case errors.Is(err, io.EOF):
			if len(characters) == 0 {
				return nil, fmt.Errorf("parsing %s: %w", path, errNoRecords)
			}

			return characters, nil
		case err != nil:
			return nil, fmt.Errorf("parsing %s (record %d): %w", path, len(characters)+1, err)
		}

		if character.SchemaVersion == "" {
			return nil, fmt.Errorf("parsing %s (record %d): %w", path, len(characters)+1, errNotCharacter)
		}

		characters = append(characters, character)
	}
}

// checkFlags validates the flags new and batch share.
//
// --name is refused here rather than escaped at each output. The name
// reaches a Markdown sheet, a Markdown transcript and a JSON record, so it
// has one entry point and three exits; a line break is the character that
// breaks all three, and no Traveller name needs one.
func checkFlags(cmd string, currentYear int, name string, stderr io.Writer) int {
	if code := checkCurrentYear(cmd, currentYear, stderr); code != exitOK {
		return code
	}

	if strings.ContainsAny(name, "\r\n") {
		fmt.Fprintf(stderr, "t5chargen %s: --name may not contain a line break\n", cmd)

		return exitUsage
	}

	return exitOK
}
