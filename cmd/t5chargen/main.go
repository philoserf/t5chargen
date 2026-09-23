// Command t5chargen generates Traveller5 characters. See docs/PRD.md.
//
// Implemented subcommands: new, batch, render, replay.
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
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
// package's own convention). docs/COMPATIBILITY.md promises them. A replay divergence is an operational error:
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
  t5chargen version                (also --version; the build, and the versions a record stamps)
  t5chargen help                   (examples, troubleshooting, and how to report a problem)
`

// help is what `t5chargen help` prints: usage, plus the things a person
// meeting this tool needs and cannot get from a flag list.
//
// It is deliberately separate from usage. usage is printed on every
// misuse, where a wall of prose buries the one line saying what went
// wrong; this is printed only when someone asked for it.
//
// The paragraph on --auto is here rather than in a prompt on purpose. A
// choice's Prompt is recorded in the event log and compared byte for byte
// on replay (docs/PRD.md FR10), so explanatory text inside one would
// invalidate every record already written. Help text costs nothing and is
// where an explanation belongs.
const help = usage + `
examples:
  t5chargen new --auto --seed 7                  a character, decided by policy
  t5chargen new                                  a character, decided by you
  t5chargen new --auto --seed 7 -o char.json     write the record
  t5chargen render char.json                     the character sheet
  t5chargen render --history char.json           every throw and choice that made him
  t5chargen replay char.json                     regenerate the record and compare
  t5chargen batch --auto --count 20 -o crew/     twenty characters, one file each

--auto and interactive runs differ, and the difference is not a bug:
  --auto answers every choice from the default policy (docs/POLICY.md).
  The policy declines career changes, so careers reachable only by
  changing into one — Craftsman and Functionary — never appear in an
  automatic run. Answer the choices yourself and they do. A record says
  which decided it: policy_version is "none" when a player did.

troubleshooting:
  replay refuses
      A record replays only under the engine version that wrote it; that
      is the provenance contract, not a defect. ` + "`t5chargen version`" + `
      prints what this build is, and the record names what wrote it.
      --ignore-provenance runs it anyway, and says so.
  version reports (devel) or a long pseudo-version
      The binary was built from a work tree rather than installed from a
      released version. Install with
      ` + "`go install github.com/philoserf/t5chargen/cmd/t5chargen@<tag>`" + `.
  a file will not be overwritten
      Nothing is overwritten without --force, and -o is checked before
      the first question, so a mistyped -o is refused before it can cost
      you a character. A record -o cannot take after all goes to stdout.

report a problem:
  https://github.com/philoserf/t5chargen/issues

  A rules report is worth far more with the record attached. Include the
  output of ` + "`t5chargen version`" + `, the record JSON, the rule you
  expected with its Book 1 page, and what happened instead.

stability:
  Prerelease. docs/COMPATIBILITY.md says what a release promises: which
  records render and replay, and which parts of the command line a
  script may rely on.
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
	// Accepted as a subcommand and as a flag. The flag form is what a
	// reporter reaches for first, and this CLI takes flags after the
	// subcommand everywhere else, so there is no subcommand for it to be
	// ambiguous with.
	case "version", "--version", "-version":
		writeVersion(stdout)

		return exitOK
	// Asked for rather than blundered into, so it goes to stdout and
	// exits zero: `t5chargen help | less` should work, and a script that
	// checks the exit status should not read a request for help as a
	// failure.
	case "help", "--help", "-help", "-h":
		fmt.Fprint(stdout, help)

		return exitOK
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
// careers, UWPs, trade classifications, and the current year, and it marks
// every error its caller caused with chargen.ErrInput.
//
// checkCurrentYear catches the years that are not years at all, but a year
// the character has outlived is only knowable once the age is (birthdate.go
// runs last), so that one comes back from the engine — and it is still the
// caller's --current-year that is wrong.
func isUsageError(err error) bool {
	return errors.Is(err, chargen.ErrInput)
}

// runNew generates a character and writes its JSON record to stdout, or to
// -o file (docs/PRD.md, CLI sketch: "new writes JSON to stdout unless -o").
func runNew(args []string, seedFn func() (uint64, error), stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(stderr)
	common := registerCommon(flags,
		"RNG seed (default: drawn from OS entropy)",
		"apply the fixed default policy (POLICY.md) to every choice",
		"output file (default: stdout)")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if code := common.check("new", flags, stderr); code != exitOK {
		return code
	}

	if err := checkOutput(*common.out, *common.force); err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	if err := resolveSeed(flags, common.seed, seedFn); err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	options := common.options()

	player := openSession(&options, *common.auto, stdin, stderr)

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

	return deliver(player, character, *common.out, *common.force, stdout, stderr)
}

// deliver writes the record, tells a player where it went, and returns the
// exit status. A record that went to stdout because -o could not take it
// was saved, but not where it was asked to be, and a script has to know.
func deliver(
	player *interactive.Decider, character chargen.Character, out string, force bool, stdout, stderr io.Writer,
) int {
	where, written := emitRecord(character, out, force, stdout, stderr)
	if written {
		closeSession(player, character, where, stderr)
	}

	if !written || where != out {
		return exitError
	}

	return exitOK
}

// runBatch generates a run of characters for NPC use: "batch emits JSONL
// (or one file per character with -o dir), requires --auto, and derives
// each member's seed from the base seed + index, recorded in each record"
// (docs/PRD.md, CLI sketch).
func runBatch(args []string, seedFn func() (uint64, error), stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("batch", flag.ContinueOnError)
	flags.SetOutput(stderr)
	count := flags.Int("count", 0, "how many characters to generate")
	common := registerCommon(flags,
		"base RNG seed; member i uses base+i (default: drawn from OS entropy)",
		"required: batch has no interactive mode",
		"output directory, named with a trailing / and created if missing (one file per character), "+
			"or a .jsonl file (default: JSONL on stdout)")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if code := common.check("batch", flags, stderr); code != exitOK {
		return code
	}

	if code := checkBatchFlags(*count, *common.auto, stderr); code != exitOK {
		return code
	}

	if err := resolveSeed(flags, common.seed, seedFn); err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	dest, err := resolveBatchDest(*common.out, *count, *common.seed, *common.force)
	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return exitError
	}

	if err := writeBatch(dest, *count, common.options(), stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "t5chargen batch: %v\n", err)

		if isUsageError(err) {
			return exitUsage
		}

		return exitError
	}

	return exitOK
}

// checkBatchFlags validates the flags batch does not share with new;
// commonFlags.check has already validated the ones it does.
func checkBatchFlags(count int, auto bool, stderr io.Writer) int {
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

	return exitOK
}

// batchDest is where a batch goes: a directory taking one file per member,
// a JSONL file, or — both empty — JSONL on stdout.
type batchDest struct {
	dir, file string
	force     bool
}

// resolveBatchDest reads -o the way docs/PRD.md spells it, "-o
// dir|file.jsonl", and refuses what cannot be written before the first
// member is generated. The path itself says which was meant: an existing
// directory is one, and so is any path written with a trailing separator —
// "-o npcs/" declares a directory, and a declaration is honoured by
// creating it, where "-o npcs" (nothing there) is a JSONL file.
//
// Resolving only looks. The declared directory is made when the run is
// written, so a batch refused for a bad --career leaves nothing behind.
// Every member's path is known before any is generated — member i is
// seed base+i — so a conflict is found here rather than after the run.
func resolveBatchDest(out string, count int, base uint64, force bool) (batchDest, error) {
	if out == "" {
		return batchDest{}, nil
	}

	isDir := false
	if info, err := os.Stat(out); err == nil {
		isDir = info.IsDir()
	}

	if !isDir && !os.IsPathSeparator(out[len(out)-1]) {
		if err := checkOutput(out, force); err != nil {
			return batchDest{}, err
		}

		return batchDest{file: out, force: force}, nil
	}

	dir := filepath.Clean(out)
	if _, err := existingAncestor(dir); err != nil {
		return batchDest{}, err
	}

	if isDir && !force {
		if err := memberConflict(dir, count, base); err != nil {
			return batchDest{}, err
		}
	}

	return batchDest{dir: dir, force: force}, nil
}

// memberConflict refuses a run with a member whose file is already in dir.
func memberConflict(dir string, count int, base uint64) error {
	for i := range count {
		path := batchPath(dir, base+uint64(i))
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("%s: %w", path, errExists)
		}
	}

	return nil
}

// existingAncestor returns the nearest directory at or above path that
// exists. A batch directory's files are staged there, which keeps the
// final renames on one filesystem.
func existingAncestor(path string) (string, error) {
	for p := path; ; p = filepath.Dir(p) {
		info, err := os.Stat(p)

		switch {
		case err == nil && info.IsDir():
			return p, nil
		case err == nil:
			return "", fmt.Errorf("%s: %w", p, errNotDirectory)
		case !errors.Is(err, fs.ErrNotExist) || filepath.Dir(p) == p:
			return "", fmt.Errorf("writing %s: %w", path, err)
		}
	}
}

// batchProgress is how often a long batch says how far it has got. A run
// that prints nothing until it finishes looks exactly like a hang.
const batchProgress = 1000

// writeBatch generates the run and writes it member by member, so memory
// holds one record rather than the whole run and its serialization — which
// at --count 20000 was four gigabytes.
//
// A batch that fails halfway is still a batch that did not happen, where
// that can be arranged: a JSONL file and a directory are staged, and put
// in place only once every member has generated. stdout cannot be staged.
// A stream that fails partway stops there, and the exit status is what
// says it is incomplete.
func writeBatch(dest batchDest, count int, opts chargen.Options, stdout, stderr io.Writer) error {
	if dest.dir != "" {
		return writeBatchDir(dest, count, opts, stderr)
	}

	return writeBatchJSONL(dest, count, opts, stdout, stderr)
}

// generateEach generates the members in order and hands each to emit. It
// derives each member's seed from the base seed plus its index, so every
// member is independently replayable from the record it lands in — the
// seed it was generated from is the seed it reports.
func generateEach(count int, opts chargen.Options, stderr io.Writer, emit func(chargen.Character) error) error {
	base := opts.Seed

	for i := range count {
		opts.Seed = base + uint64(i)

		character, err := chargen.Generate(opts)
		if err != nil {
			return fmt.Errorf("character %d of %d (seed %d): %w", i+1, count, opts.Seed, err)
		}

		if err := emit(character); err != nil {
			return err
		}

		if (i+1)%batchProgress == 0 {
			fmt.Fprintf(stderr, "t5chargen batch: %d of %d\n", i+1, count)
		}
	}

	return nil
}

// writeBatchJSONL writes one record per line, so the stream stays
// greppable and a consumer can read it a character at a time.
func writeBatchJSONL(dest batchDest, count int, opts chargen.Options, stdout, stderr io.Writer) error {
	out := stdout

	var tmp *os.File

	if dest.file != "" {
		var err error
		if tmp, err = stageFile(dest.file); err != nil {
			return err
		}

		defer discard(tmp)

		out = tmp
	}

	// An Encoder writes what json.Marshal does, and a line break.
	buffered := bufio.NewWriter(out)
	encoder := json.NewEncoder(buffered)

	err := generateEach(count, opts, stderr, func(character chargen.Character) error {
		if err := encoder.Encode(character); err != nil {
			return fmt.Errorf("encoding seed %d: %w", character.RNG.Seed, err)
		}

		return nil
	})

	// Flushed even on failure: on stdout, the members already generated
	// are whole lines a consumer can use; in a staged file they are
	// discarded with it.
	if flushErr := buffered.Flush(); err == nil && flushErr != nil {
		err = fmt.Errorf("writing: %w", flushErr)
	}

	if err != nil || dest.file == "" {
		return err
	}

	return commitFile(tmp, dest.file, dest.force)
}

// writeBatchDir writes one indented record per member into a staging
// directory beside the target, and moves the run into place only once
// every member exists.
func writeBatchDir(dest batchDest, count int, opts chargen.Options, stderr io.Writer) error {
	root, err := existingAncestor(dest.dir)
	if err != nil {
		return err
	}

	stage, err := os.MkdirTemp(root, ".t5chargen-*")
	if err != nil {
		return fmt.Errorf("writing %s: %w", dest.dir, err)
	}

	defer os.RemoveAll(stage) //nolint:errcheck // best effort; empty once the run is committed

	var names []string

	err = generateEach(count, opts, stderr, func(character chargen.Character) error {
		data, encodeErr := json.MarshalIndent(character, "", "  ")
		if encodeErr != nil {
			return fmt.Errorf("encoding seed %d: %w", character.RNG.Seed, encodeErr)
		}

		name := filepath.Base(batchPath("", character.RNG.Seed))
		names = append(names, name)

		//nolint:gosec // G306: a character record is not a secret; 0o644 matches writeFile.
		return os.WriteFile(filepath.Join(stage, name), append(data, '\n'), 0o644)
	})
	if err != nil {
		return err
	}

	return commitDir(stage, dest, root, names)
}

// commitDir moves a staged run into dest.dir, creating it and any missing
// parents. If a file cannot be placed — something else wrote one of the
// names while the run was generating — the files this run placed and the
// directories it made are removed again, so the run is all there or not
// there. Under --force it cannot be: an overwritten record is gone, so
// what was placed stays and the error says where it stopped.
func commitDir(stage string, dest batchDest, root string, names []string) error {
	var made []string
	for p := dest.dir; p != root && p != filepath.Dir(p); p = filepath.Dir(p) {
		made = append(made, p)
	}

	//nolint:gosec // G301: an output directory the caller named; 0755 matches mkdir(1).
	if err := os.MkdirAll(dest.dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dest.dir, err)
	}

	for i, name := range names {
		err := commitPath(filepath.Join(stage, name), filepath.Join(dest.dir, name), dest.force)
		if err == nil {
			continue
		}

		if !dest.force {
			for _, placed := range names[:i] {
				os.Remove(filepath.Join(dest.dir, placed)) //nolint:errcheck,gosec // best effort unwind
			}

			// Deepest first, and os.Remove takes only an empty directory,
			// which is exactly the rule wanted.
			for _, dir := range made {
				os.Remove(dir) //nolint:errcheck,gosec // best effort unwind
			}
		}

		return err
	}

	return nil
}

// batchPath names a batch member's file for the seed that produced it, so
// the file says how to reproduce itself.
func batchPath(dir string, seed uint64) string {
	return filepath.Join(dir, fmt.Sprintf("character-%d.json", seed))
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

// commonFlags are the flags new and batch share. They are registered once
// and checked once: declared twice, the two subcommands' validation forked,
// and batch stopped refusing the --name that new refuses.
type commonFlags struct {
	seed        *uint64
	name        *string
	career      *string
	homeworld   *string
	currentYear *int
	auto        *bool
	out         *string
	force       *bool
}

// registerCommon registers the shared flags on flags. The three whose
// meaning genuinely differs between the subcommands take their usage text
// as arguments; the rest are described once.
func registerCommon(flags *flag.FlagSet, seedUsage, autoUsage, outUsage string) commonFlags {
	return commonFlags{
		seed:      flags.Uint64("seed", 0, seedUsage),
		name:      flags.String("name", "", "character name (blank by default)"),
		career:    flags.String("career", "", "force the first career"),
		homeworld: flags.String("homeworld", "", homeworldUsage),
		currentYear: flags.Int("current-year", calendar.DefaultYear,
			"Imperial year adventuring begins in, which fixes the birth year (Book 1 p. 58)"),
		auto:  flags.Bool("auto", false, autoUsage),
		out:   flags.String("o", "", outUsage),
		force: flags.Bool("force", false, "overwrite existing output"),
	}
}

// check validates the shared flags, and refuses stray arguments, which
// neither subcommand takes. cmd names the subcommand that read them,
// because the diagnostic has to say which one refused.
//
// --name is refused here rather than escaped at each output. The name
// reaches a Markdown sheet, a Markdown transcript and a JSON record, so it
// has one entry point and three exits; a line break is the character that
// breaks all three, and no Traveller name needs one.
func (c commonFlags) check(cmd string, flags *flag.FlagSet, stderr io.Writer) int {
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "t5chargen %s: unexpected arguments %q (use -o for output)\n%s", cmd, flags.Args(), usage)

		return exitUsage
	}

	if code := checkCurrentYear(cmd, *c.currentYear, stderr); code != exitOK {
		return code
	}

	if strings.ContainsAny(*c.name, "\r\n") {
		fmt.Fprintf(stderr, "t5chargen %s: --name may not contain a line break\n", cmd)

		return exitUsage
	}

	return exitOK
}

// options assembles the engine options the flags describe. batch passes
// the base seed and generateBatch replaces it per member.
func (c commonFlags) options() chargen.Options {
	return chargen.Options{
		Seed:          *c.seed,
		Name:          *c.name,
		Career:        canonicalCareer(*c.career),
		Homeworld:     parseHomeworldFlag(*c.homeworld),
		RollHomeworld: isRandomHomeworld(*c.homeworld),
		CurrentYear:   *c.currentYear,
		Decider:       chargen.DefaultPolicy{},
	}
}

// emitRecord marshals the record and writes it to stdout or the output
// file, and reports where it went: out, or "" for stdout, and whether it
// was written at all.
//
// A record -o could not take goes to stdout instead. checkOutput refuses the paths that can be refused before a
// session starts, but not a file that appears while it runs, a full disk,
// or a directory whose permissions changed; and after an interactive run
// the record is the only copy of a player's answers, which no seed
// reproduces. A record on stdout can be saved; a record nowhere cannot.
func emitRecord(character chargen.Character, out string, force bool, stdout, stderr io.Writer) (string, bool) {
	data, err := json.MarshalIndent(character, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "t5chargen: encoding character: %v\n", err)

		return "", false
	}

	data = append(data, '\n')

	if out != "" {
		err := writeFile(out, data, force)
		if err == nil {
			return out, true
		}

		fmt.Fprintf(stderr, "t5chargen: %v\nt5chargen new: the record is on stdout instead\n", err)
	}

	if _, err := stdout.Write(data); err != nil {
		fmt.Fprintf(stderr, "t5chargen: %v\n", err)

		return "", false
	}

	return "", true
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

// homeworldUsage documents --homeworld for both new and batch, through
// registerCommon.
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
// overwritten without --force." (docs/PRD.md, CLI sketch) — the path is
// claimed exclusively unless force allows replacing it.
// The record is written whole or not at all: a truncating write that
// fails partway leaves a half-written file where a valid record was, and
// a record is the one artifact this tool exists to produce. Writing beside
// the target and renaming over it makes the replacement atomic.
func writeFile(path string, data []byte, force bool) error {
	tmp, err := stageFile(path)
	if err != nil {
		return err
	}

	defer discard(tmp)

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return commitFile(tmp, path, force)
}

// stageFile creates the temporary file a record for path is written into,
// beside it, so the rename that puts it in place stays on one filesystem.
func stageFile(path string) (*os.File, error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".t5chargen-*")
	if err != nil {
		return nil, fmt.Errorf("writing %s: %w", path, err)
	}

	return tmp, nil
}

// discard closes and removes a staged file. After commitFile it finds
// nothing to remove, which is why its errors are not reported.
func discard(tmp *os.File) {
	tmp.Close()           //nolint:errcheck,gosec // already closed on the committed path
	os.Remove(tmp.Name()) //nolint:errcheck,gosec // already renamed on the committed path
}

// commitFile puts a staged file in place at path.
func commitFile(tmp *os.File, path string, force bool) error {
	// Synced before the rename: without it the rename can be durable
	// while the bytes it points at are not.
	if err := tmp.Sync(); err != nil {
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

	return commitPath(tmp.Name(), path, force)
}

// commitPath renames a staged file onto path, claiming path first unless
// force allows replacing what is there. A claim the rename then fails to
// fill is removed: it is an empty file this run made.
func commitPath(staged, path string, force bool) error {
	if !force {
		if err := claimPath(path); err != nil {
			return err
		}
	}

	if err := os.Rename(staged, path); err != nil {
		if !force {
			os.Remove(path) //nolint:errcheck,gosec // best effort; the claim is this run's own
		}

		// A LinkError prints both of its paths, and the first is a
		// temporary name the reader never chose and cannot act on.
		if linkErr, ok := errors.AsType[*os.LinkError](err); ok {
			err = linkErr.Err
		}

		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// checkOutput refuses an -o that new cannot write, before anything is
// generated. Until it existed the write was the check, and the write comes
// after the last question: an interactive run aimed at a file that already
// existed was refused only once the player had answered everything, and
// the lifepath went with the refusal.
//
// It only looks. Claiming the path here, the way writeFile does, would
// leave an empty file behind an abandoned session ("Interrupted interactive
// sessions produce no output file", docs/PRD.md CLI sketch) and make
// writeFile's own claim refuse the file this one had made. writeFile's
// exclusive create is still what holds against a file that appears while
// the session runs.
func checkOutput(out string, force bool) error {
	if out == "" {
		return nil
	}

	// A trailing separator declares a directory, which is how batch's -o
	// reads it. new has no directory form, and saying so beats writing a
	// file named for one.
	if os.IsPathSeparator(out[len(out)-1]) {
		return fmt.Errorf("%s: %w", out, errDestIsDirectory)
	}

	// Lstat, so a dangling symlink counts as the thing writeFile's
	// exclusive create will find there.
	info, err := os.Lstat(out)

	switch {
	case errors.Is(err, fs.ErrNotExist):
		if parent, statErr := os.Stat(filepath.Dir(out)); statErr != nil || !parent.IsDir() {
			return fmt.Errorf("%s: %w", filepath.Dir(out), errNoParent)
		}
	case err != nil:
		return fmt.Errorf("writing %s: %w", out, err)
	case info.IsDir():
		return fmt.Errorf("%s: %w", out, errDestIsDirectory)
	case !force:
		return fmt.Errorf("%s: %w", out, errExists)
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

// errDestIsDirectory reports a directory where new was told to write its
// record. It is the write side of errIsDirectory, with the advice turned
// round: new writes one file, and filling a directory is batch's job.
var errDestIsDirectory = errors.New(
	"names a directory; new writes one file, so name the file (batch fills a directory)")

// errNoParent reports an output path whose directory does not exist. new
// does not create one: a directory nobody asked for, made because of a
// typo, is a mess of its own.
var errNoParent = errors.New("no such directory")

// errNotDirectory reports a file where a batch's directory, or one of its
// parents, would have to be.
var errNotDirectory = errors.New("not a directory")

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
