# t5chargen Walkthrough

*2026-09-10T12:42:17Z by Showboat 0.6.1*
<!-- showboat-id: 5a1d87ac-6f82-4ed9-b1e0-c4ad24d04d56 -->

## Overview

`t5chargen` is a command-line character generator for **Traveller5**, the
2013 edition of the tabletop RPG. It walks a character through the game's
*lifepath*: roll six characteristics, settle a homeworld, optionally go to
school, serve one or more four-year career terms, age, and muster out.

The thing to hold in mind before reading a line of it: the product is not
the character, it is **the record**. Every throw, every choice and every
consequence is logged as a numbered event, and the JSON record embeds that
whole log. `render` derives a character sheet or a generation transcript
from the record; `replay` re-runs the engine from the recorded seed and the
recorded choices and checks that the walk reproduces itself event for
event. `THEORY.md` argues why that shape is forced; this document follows
the code that implements it.

The module is standard-library only, on Go 1.27. Four subcommands: `new`,
`batch`, `render`, `replay` (plus `version` and `help`).

```bash
cat go.mod
```

```output
module github.com/philoserf/t5chargen

go 1.27
```

## Architecture

Seventeen packages, and the split is by *kind of thing* rather than by
layer. `cmd/t5chargen` is the CLI. `chargen` is the engine. `dice` is the
seeded random stream. `render` turns a record into Markdown. `interactive`
is the line-based front end. `audit` is test-only and contains no rules at
all — it is the machinery that keeps the documents in `docs/` honest.

Everything else is one embedded chart or vocabulary each, loaded with the
same `go:embed` plus `sync.OnceValues` pattern and validated at load time.

```bash
cat <<'TREE'
t5chargen/
  cmd/t5chargen/    CLI: new, batch, render, replay, version, help
  chargen/          the engine: event log, Decider seam, term loop, replay
  dice/             xD rolls, Flux, target-number throws (one PCG stream)
  render/           character sheet and generation transcript
  interactive/      line-based front end (one Decider implementation)
  career/           charts 01-13 as JSON, plus the loader
  world/            chart B: homeworlds, UWP and trade classifications
  education/        chart C: pre-career programmes
  benefit/ medal/ ship/ fame/ lifestage/ skill/ ehex/ calendar/
                    one chart or vocabulary each
  audit/            test-only gates over the documents in docs/
  docs/             PRD, ERRATA, COVERAGE, POLICY, COMPATIBILITY, schema
TREE
```

```output
t5chargen/
  cmd/t5chargen/    CLI: new, batch, render, replay, version, help
  chargen/          the engine: event log, Decider seam, term loop, replay
  dice/             xD rolls, Flux, target-number throws (one PCG stream)
  render/           character sheet and generation transcript
  interactive/      line-based front end (one Decider implementation)
  career/           charts 01-13 as JSON, plus the loader
  world/            chart B: homeworlds, UWP and trade classifications
  education/        chart C: pre-career programmes
  benefit/ medal/ ship/ fame/ lifestage/ skill/ ehex/ calendar/
                    one chart or vocabulary each
  audit/            test-only gates over the documents in docs/
  docs/             PRD, ERRATA, COVERAGE, POLICY, COMPATIBILITY, schema
```

## 1. The entry point

`main` does one thing: it calls `run` with everything injectable — the
argument list, a seed source, and the three streams — and exits with what
it returns. Tests drive the whole CLI through `run` with a fixed seed and a
scripted stdin, so there is no code path that only exists in production.

`randomSeed` is the single deliberate exception to the repository's
no-unseeded-randomness rule. The rule is engine-scoped: the CLI may pick a
seed, and the seed it picks is written into the record, so replay stays
exact.

```bash
sed -n '106,121p' cmd/t5chargen/main.go
```

```output
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
```

### Dispatch

`run` is a flat switch. Note the two conventions it encodes: `version` and
`help` are accepted both as subcommands and as flags, and `help` goes to
stdout with exit 0 because a request for help is not a failure. Everything
else that is the caller's fault exits 2; an operation that ran and failed
exits 1.

```bash
sed -n '126,163p' cmd/t5chargen/main.go
```

```output
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
```

### `new`, in the middle

`runNew` parses flags, validates what it can validate itself, resolves the
seed, builds `chargen.Options`, and opens a session. `openSession` is where
the two decider implementations part: `--auto` installs
`chargen.DefaultPolicy{}`, and without it an `interactive.Decider` reading
stdin. The engine is handed a `Decider` either way and cannot tell which.

The `interactive.ErrAbandoned` branch is the contract that an interrupted
session leaves no file behind: it reports and returns before anything is
written.

```bash
sed -n '196,221p' cmd/t5chargen/main.go
```

```output
	if code := common.check("new", flags, stderr); code != exitOK {
		return code
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
```

## 2. The engine: `chargen.Generate`

This is the spine of the whole system, and it reads as the Master Chargen
Checklist reads (Book 1 chart E1, p. 72): step A characteristics, step B
homeworld, step C education, step D career, then what follows careers —
Fame, muster out, birthdate.

Three things to notice:

- **One roller.** `dice.New(opts.Seed)` creates the single PCG stream and
  it is threaded by pointer through everything below. The *order* in which
  that stream is consumed is part of `EngineVersion`, which is why adding a
  roll anywhere is a version bump even when no rule changed meaning.
- **`policy_version` is decided by a type assertion.** Only
  `DefaultPolicy` stamps the policy version; every other `Decider` — the
  interactive player, a test decider, the replay decider — stamps `"none"`.
- **`careerStartAge` is captured between education and careers**, because
  after this point nothing can tell how much of the character's age was
  spent at home, and a World Knowledge counts exactly that.

```bash
sed -n '652,704p' chargen/character.go
```

```output
func Generate(opts Options) (Character, error) {
	if opts.Decider == nil {
		return Character{}, errNoDecider
	}

	if err := checkSharedData(); err != nil {
		return Character{}, err
	}

	roller := dice.New(opts.Seed)

	log := newLog(opts.Decider)

	// policy_version attests which decision table governed the run's
	// choices: the POLICY.md version only when the default policy itself
	// decided, "none" for any other Decider.
	policyVersion := "none"
	if _, isDefault := opts.Decider.(DefaultPolicy); isDefault {
		policyVersion = PolicyVersion
	}

	// A homeworld the caller named is assigned; one it did not is the
	// character's to select from chart B (p. 58).
	assigned := supplied(opts.Homeworld)

	character := newCharacter(opts, policyVersion, assigned)

	log.Step("Generate Characteristics", "Book 1 p. 72 chart E1 step A")

	character.Characteristics = RollCharacteristics(roller, &log)

	homeworld, err := homeworldOrDefault(opts.Homeworld)
	if err != nil {
		return Character{}, err
	}

	if err := runHomeworld(homeworld, assigned, opts.RollHomeworld, roller, &log, opts.Decider, &character); err != nil {
		return Character{}, err
	}

	if err := runEducation(roller, &log, opts.Decider, &character); err != nil {
		return Character{}, err
	}

	// The age careers begin at, which is what a World Knowledge counts
	// the terms of (p. 134, interpretation I-112). Captured here because
	// education has just finished moving it and nothing afterwards can
	// tell how much of the character's age was spent at home.
	careerStartAge := character.Age

	if err := runCareer(opts.Career, roller, &log, opts.Decider, &character); err != nil {
		return Character{}, err
	}
```

## 3. The seeded stream

`dice` is the only source of randomness in the engine. A `Roller` wraps one
`math/rand/v2` PCG, seeded by expanding the single user-facing seed into
both PCG state words. Every roll records the individual die faces in roll
order, because the event log needs them.

Read the doc comment on `New` carefully — it is where the strongest
constraint in the repository is stated. The seed expansion, the algorithm,
*and the face-consumption order of every roll method* are version-locked.
A change to any of them is an engine version bump, whether or not any rule
changed.

```bash
sed -n '23,42p' dice/dice.go
```

```output
// Roller draws die faces from a seeded PCG stream. It is the only source
// of randomness in the engine (docs/PRD.md FR9): every roll consumes faces
// from the stream in a fixed, documented order.
//
// A Roller is not safe for concurrent use; the engine consumes it
// sequentially so that replay is deterministic.
type Roller struct {
	rng *rand.Rand
}

// New returns a Roller seeded with seed. The single user-facing seed is
// expanded to the two PCG state words as NewPCG(seed, seed).
//
// This expansion, the PCG algorithm, and the face-consumption order of each
// roll method are version-locked by the replay and provenance contract
// (docs/PRD.md): changing any of them is an engine version bump.
func New(seed uint64) *Roller {
	//nolint:gosec // G404: a deterministic, seeded, non-cryptographic stream is the requirement (docs/PRD.md FR9).
	return &Roller{rng: rand.New(rand.NewPCG(seed, seed))}
}
```

## 4. The event log

Four event kinds — `step`, `throw`, `choice`, `consequence` — and exactly
one payload field is non-nil on each. A consequence carries the sequence
number of the throw or choice that caused it, which is what lets the
transcript say *why* a value moved.

```bash
sed -n '31,41p' chargen/event.go
```

```output
// Event is one entry in the generation record. Exactly one payload field is
// non-nil, matching Kind.
type Event struct {
	Seq  int       `json:"seq"`
	Kind EventKind `json:"kind"`

	Step        *StepEvent        `json:"step,omitempty"`
	Throw       *ThrowEvent       `json:"throw,omitempty"`
	Choice      *ChoiceEvent      `json:"choice,omitempty"`
	Consequence *ConsequenceEvent `json:"consequence,omitempty"`
}
```

`Log` accumulates them. Sequence numbers are assigned on append, monotonic
from 1. The `watch` hook is the observation channel a front end uses to
follow a generation it is driving — and the doc comment on `Watcher` is
emphatic that it is observation only: a watcher is handed copies, returns
nothing, and is consulted after the event is already recorded, so nothing
about a character can depend on whether one is present.

```bash
sed -n '387,397p' chargen/event.go
```

```output
// Watcher is shown each event as the engine records it, for a Decider
// that wants to follow the generation as well as answer it.
//
// Watching is not deciding. A Watcher cannot change anything: it is given
// copies, it returns nothing, and it is consulted after the event is
// already recorded. Nothing about a character depends on whether one is
// present — DefaultPolicy does not implement this and neither does the
// replay decider, so no fixture moves and replay is unaffected.
type Watcher interface {
	Watch(event Event)
}
```

```bash
sed -n '513,527p' chargen/event.go
```

```output
// append assigns the next sequence number (monotonic from 1) and stores the
// event.
func (l *Log) append(event Event) int {
	event.Seq = len(l.events) + 1
	l.events = append(l.events, event)

	// Handed out as a copy, the same discipline Events keeps: a watcher
	// that mutated a payload would corrupt the record it is watching, and
	// the record is what replay verifies against.
	if l.watch != nil {
		l.watch(event.clone())
	}

	return event.Seq
}
```

## 5. The `Decider` seam

This is the narrowest waist in the system and the reason replay can be as
strong as it is. Every point where the printed rules ask a person something
goes through one interface. There are about forty such points, each a
`ChoiceID` constant carrying the sentence from Book 1 that creates it.

A `Choice` has two halves, and the split matters. `Prompt`, `Options` and
`Cite` are **recorded**: they go into the choice event and replay compares
them. `Scores`, `ScoreLabel`, `Nth` and `Of` are engine-computed decision
aids that are **not** recorded — which is what lets a front end get better
at helping without invalidating a single existing record.

```bash
sed -n '206,245p' chargen/decider.go
```

```output

// Choice is one choice point presented to a Decider. Options are listed in
// the order the rule presents them (first-listed order in Book 1). Scores,
// when non-nil, are engine-provided decision aids parallel to Options — the
// current characteristic values behind a controlling-characteristic choice,
// or what refusing a waiver would cost (POLICY.md). They are not part of
// the printed rule and are not recorded in the event log, so a policy can
// weigh a stake without reading the prompt text and rewording a prompt
// cannot change a generated character.
type Choice struct {
	ID      ChoiceID
	Prompt  string
	Options []string
	Scores  []int
	Cite    string

	// ScoreLabel names what Scores mean, for a decider that shows them to
	// a person. "1" against a program means "you qualify", and nobody
	// could guess that from the digit; unlabelled scores stay between the
	// engine and the policy.
	//
	// A label applies to every option in the list, so only label a Score
	// that means the same thing for each of them. Where the array is
	// really one flag about the choice, padded to length — the waiver
	// stake is — labelling it reads as a claim about each option in turn,
	// and the second option's padding reads as the opposite of the truth.
	ScoreLabel string

	// Nth and Of place a choice in a run of identical ones: the term's
	// skill selections are the same question asked several times, and a
	// player answering the fifth cannot otherwise tell it from the first.
	// Engine-provided decision data like Scores — not part of the printed
	// rule, and not recorded, so a front end may show them and replay
	// never sees them.
	Nth, Of int
}

// Decider resolves choice points. Interactive play and the auto-mode
// policy are its two implementations (docs/PRD.md, Decisions): every
// choice in the engine goes through this interface so replay can reapply
```

`choose` is the single call site every choice point funnels through. It
validates the answer, logs the resolved choice event, and hands back both
the index and the event's sequence number so that any consequence can name
what caused it.

The line to stare at is `Options: c.Options`. **The recorded answer is an
index into that list.** Reorder the options a choice point presents and
every record already written silently means something different. This is
also why `Prompt` is recorded and why explanatory text belongs in
`t5chargen help` rather than in a prompt.

```bash
sed -n '983,1006p' chargen/character.go
```

```output
func choose(log *Log, decider Decider, c Choice) (int, int, error) {
	if len(c.Options) == 0 {
		return 0, 0, fmt.Errorf("%w: %q presented no options", errBadChoice, c.ID)
	}

	chosen, err := decider.Choose(c)
	if err != nil {
		return 0, 0, fmt.Errorf("%q: %w", c.ID, err)
	}

	if chosen < 0 || chosen >= len(c.Options) {
		return 0, 0, fmt.Errorf("%w: %q answer %d outside 0-%d", errBadChoice, c.ID, chosen, len(c.Options)-1)
	}

	seq := log.Choice(ChoiceEvent{
		Decider: decider.Kind(),
		Prompt:  c.Prompt,
		Options: c.Options,
		Chosen:  chosen,
		Cite:    c.Cite,
	})

	return chosen, seq, nil
}
```

### Decider 1 — the auto policy

`DefaultPolicy` is the fixed decision table behind `--auto`. It is *total*
— every choice point has a rule, and a gate in `audit/` fails the build if
one does not — and it never refuses, so its error is always nil.

The rules are worth reading for how they select. Note `ChooseCareer`: it
picks Citizen **by name**, not by index. The options are listed in chart
order, so first-listed would hand the default career to whichever chart
number happens to be lowest among those implemented — which would change
every generated character each time an earlier chart landed.

```bash
sed -n '5,15p' chargen/policy.go
```

```output
// DefaultPolicy is the fixed auto-mode decision table, version
// PolicyVersion; the rules and their rationale live in POLICY.md
// (docs/PRD.md, CLI sketch: the policy is total, deterministic, and
// tie-breaks by first-listed order in Book 1).
type DefaultPolicy struct{}

// Choose applies the POLICY.md rule for the choice point. The policy is
// total, so it never refuses: the error is always nil.
func (p DefaultPolicy) Choose(c Choice) (int, error) {
	return p.decide(c), nil
}
```

```bash
sed -n '71,96p' chargen/policy.go
```

```output
	switch c.ID {
	case ChooseSkillColumn:
		// POLICY.md: the first present of General, then Exploration, then
		// Business — the all-plain-skills columns of the shipped careers;
		// first-listed otherwise.
		return preferredIndex(c.Options,
			[]string{"General", "Exploration", "Business", "Combat", "Peacekeeper", "Mission"}, 0), true
	case ChooseDuty:
		// POLICY.md: Explorer Duty — the career's point, and the larger
		// skill eligibility (chart 05 table B).
		return indexOrFirst(c.Options, "Explorer Duty"), true
	case ChooseRiskMod:
		// POLICY.md: No Mod.
		return indexOrFirst(c.Options, "No Mod"), true
	case ChooseEducation:
		return chooseEducationProgram(c), true
	case ChooseCareer:
		// POLICY.md: Citizen by name. The alternatives are listed in
		// chart order, so first-listed would hand the default career to
		// whichever chart number is lowest among those implemented —
		// changing every generated character each time an earlier chart
		// lands.
		return indexOrFirst(c.Options, "Citizen"), true
	default:
		return 0, false
	}
```

### Decider 2 — the interactive player

`interactive.Decider` writes numbered prompts to a `Writer` and reads
answers from a `Reader`, so the whole front end is testable with a scripted
script and no terminal. A choice with a single option is answered without
asking: the engine presents some choices that way as a seam rather than a
decision, and asking a player to confirm the only thing he may do is not a
question.

```bash
sed -n '69,95p' interactive/interactive.go
```

```output
// Choose puts the choice to the player and returns the index answered.
//
// A choice with one option is answered without asking. The engine presents
// some choices that way as a seam rather than a decision — an assigned
// homeworld is the standing example — and asking a player to confirm the
// only thing he may do is not a question.
func (d *Decider) Choose(c chargen.Choice) (int, error) {
	if len(c.Options) == 1 {
		fmt.Fprintf(d.out, "\n%s\n  %s\n", c.Prompt, c.Options[0])

		return 0, nil
	}

	d.present(c, "")

	for {
		answer, err := d.read()
		if err != nil {
			return 0, err
		}

		index, done, err := d.resolve(c, answer)
		if done || err != nil {
			return index, err
		}
	}
}
```

`parseIndex` is a good example of the level of care in this front end.
Only bare digits count as an option number, because `strconv.Atoi` reads
`"+2"` as `2` — and the benefit-DM menu lists its options as `+0`, `+1`,
`+2`, so a player copying the option he wants would silently select the one
above it. A signed answer falls through to the search filter instead; a
bare number the list does not hold is refused by name rather than searched
for, since every UWP in the homeworld list is full of digits.

```bash
sed -n '202,224p' interactive/interactive.go
```

```output
// parseIndex reads an answer as a 1-based option number, reporting the
// 0-based index and whether the answer was a number at all; whether the
// list holds that option is the caller's question. Only bare digits count:
// strconv.Atoi would read "+2" as 2, and the benefit DM menu lists its
// options as "+0", "+1", "+2", so a player copying the option he wants
// would silently select the one above it. A signed answer is not a number
// here, and falls through to the filter, which matches it against the
// option text and shows it under its own number.
//
// A run of digits too long for an int is still a number, and one no list
// holds, so it comes back as -1.
func parseIndex(answer string) (int, bool) {
	if answer == "" || strings.TrimLeft(answer, "0123456789") != "" {
		return 0, false
	}

	n, err := strconv.Atoi(answer)
	if err != nil {
		return -1, true
	}

	return n - 1, true
}
```

## 6. Careers: one loop, thirteen sets of exceptions

`chargen/careerrun.go` is the largest file in the repository, and it is
large for a defensible reason: the thirteen charts share one procedure with
thirteen sets of exceptions, and the alternative to one long shared loop is
thirteen slightly divergent copies of it.

Charts 01-13 are *data* — `career/data/*.json`, 120 KB of skill tables,
target numbers, benefit rows and rank titles, with no conditional logic in
it anywhere. What a chart needs that a table cannot express becomes Go,
behind a two-method unexported interface.

```bash
sed -n '44,61p' chargen/careerrun.go
```

```output
// errUnregisteredCareer reports a career present in career.Available but
// missing from careerRegistry — an internal wiring bug, distinct from the
// user-facing ErrUnknownCareer (which the CLI maps to a usage exit).
var errUnregisteredCareer = errors.New("career has no registered mechanics")

// careerMechanics is one career's exceptional mechanics. The interface is
// unexported and grows with the careers that need more seams (rank,
// commission, muster out land with milestones 3-4).
type careerMechanics interface {
	// begin resolves career entry: automatic for Citizen (chart 04), a
	// To Begin throw for most careers (chart D p. 64; p. 65). It reports
	// whether the career began; a failed attempt costs a year (p. 65).
	begin(r *careerRun) (bool, error)

	// resolveTerm runs the career's Risk/Reward variant for the term
	// (p. 65: Citizen Life for Citizens) and applies its awards.
	resolveTerm(r *careerRun, cc string) (termOutcome, error)
}
```

`careerRegistry` is the wiring. Its key set must match `career.Available`,
and a gate holds them equal — a career with data but no mechanics is an
internal bug (`errUnregisteredCareer`), deliberately distinct from the
user-facing `ErrUnknownCareer` that the CLI maps to a usage exit.

```bash
sed -n '107,123p' chargen/careerrun.go
```

```output
// careerRegistry maps canonical career names to their definition and
// mechanics. Its key set must match career.Available (tested).
var careerRegistry = map[string]func() (*career.Definition, careerMechanics, error){
	"Citizen":     newCitizen,
	"Scholar":     newScholar,
	"Noble":       newNoble,
	"Functionary": newFunctionary,
	"Craftsman":   newCraftsman,
	"Soldier":     newSoldier,
	"Spacer":      newSpacer,
	"Marine":      newMarine,
	"Agent":       newAgent,
	"Rogue":       newRogue,
	"Entertainer": newEntertainer,
	"Scout":       newScout,
	"Merchant":    newMerchant,
}
```

`term` resolves one term of a career, and its structure repays reading. The
Later Education offer comes *before* the term's own step, because "at the
beginning of any term" is where the rule puts it and because a suspended
term is not a term of the career — it must not open one in the transcript.
The death check between them is not defensive padding: once `Dead` is set,
`ageEffects` stops checking, so without this the loop would be unbounded.

```bash
sed -n '518,546p' chargen/careerrun.go
```

```output
// term resolves one term: either the character suspends it for school
// (p. 59) or he serves it.
//
// The offer comes before the term's own step, because "at the beginning of
// any term" is where the rule puts it and because a suspended term is not
// a term of the career — it must not open one in the transcript.
func (r *careerRun) term(number int) (termEnd, error) {
	suspended, err := r.laterEducation()
	if err != nil {
		return termCareerEnded, err
	}

	// Aging kills at school as readily as in service: the years pass
	// either way (chart A p. 89), and the refused applicant's lost year
	// (interpretation I-89) passes too. Either death ends the lifepath
	// here, exactly as serveTerm ends it for a term served — a corpse
	// must not be offered school again, nor serve the term he applied
	// out of. Without this the loop is unbounded: once Dead is set,
	// ageEffects stops checking, so nothing else can ever end it.
	if r.character.Dead {
		return termDied, nil
	}

	if suspended {
		return termContinues, nil
	}

	return r.serveTerm(number)
}
```

### One place a character ages

Everything about the passage of time funnels through `advanceYears`. That
is not tidiness — aging hangs off it. Characters reach Life Stage 5 at age
34 and are then subject to an Aging Check every four years, which is a rule
about elapsed time and not about any of the nine particular events that
elapse it. A stray `c.Age += 4` anywhere else would silently skip an Aging
Check and emit no event saying so.

```bash
sed -n '50,76p' chargen/character.go
```

```output
// advanceYears elapses game time and records it, the single place a
// character ages. Time passes for two printed reasons: a term is four
// years ("the 4-year Term", p. 66) and a failed career entry costs one
// ("Each failed attempt (both Begin or Retry) takes one year", p. 65).
//
// One site rather than nine because aging hangs off it: characters reach
// Life Stage 5 at age 34 and are then "subject to Aging" every four years
// (chart A, p. 89), which is a rule about the passage of time and not
// about any of the particular events that pass it.
//
// cause is the throw or choice that consumed the time, never a step. FR10
// allows a step cause where no throw or choice produced the consequence
// (interpretation I-87); elapsed years are not such a case — something
// always consumes the time — so this site holds to the stricter rule. The
// Aging Checks the span crosses carry their own throws as causes.
func (c *Character) advanceYears(years int, roller *dice.Roller, log *Log, cause int) error {
	from := c.Age

	c.Age += years
	log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceYearsElapsed, Value: years})

	// Aging is a rule about the passage of time, so it resolves here
	// rather than at any of the events that pass it: "Once Aging begins,
	// it occurs every four years on the character's birthday" (chart A
	// p. 89).
	return c.ageEffects(from, c.Age, roller, log)
}
```

## 7. After the careers

Fame, then muster out, then the birthdate — and the order is forced. Fame
is calculated over the *finished* record rather than accumulated as the
lifepath runs, and muster out reads it ("one additional roll if Fame 19+",
p. 68). The birthdate is settled last because p. 58 puts it last and
because it reads an age that muster out is the final chance to change.

A dead character keeps his record but is given nothing: the muster-out step
is not even opened, because a section that would hold nothing should not
appear in the transcript.

```bash
sed -n '716,752p' chargen/character.go
```

```output
// afterCareers runs checklist step E and what follows it: Fame, which
// muster out reads (p. 68), then muster out itself (p. 67).
func afterCareers(c *Character, roller *dice.Roller, log *Log, decider Decider,
	currentYear, careerStartAge int,
) error {
	// Before Fame, because these are the finished record's own totals and
	// everything after reads the skills.
	awardSpecializedKnowledges(c, log, careerStartAge)

	log.Step("Determine Fame", "Book 1 p. 91 chart F")

	if err := computeFame(c, roller, log, decider); err != nil {
		return err
	}

	// "Mustering Out counts up the character's belongings ... and notes
	// them as assets for the adventuring situations to come" (p. 67). A
	// dead character has none, and p. 69 is blunter: "the Character is
	// dead (and all efforts in this particular character creation
	// process are lost)". The record is kept — that much of the sentence
	// the engine declines, I-51 — but nothing is added to it, and the
	// step is not opened for a section that would hold nothing
	// (interpretation I-77).
	if !c.Dead {
		log.Step("Muster Out", "Book 1 p. 67; chart M1 p. 70")

		if err := musterOut(c, roller, log, decider); err != nil {
			return err
		}
	}

	// Last, because p. 58 puts it last and because it reads the age that
	// muster out is the final chance to change.
	if err := c.birthdate(roller, log, currentYear); err != nil {
		return err
	}

```

## 8. Provenance: the three versions a record stamps

Every record carries five identifiers, and understanding what each entitles
you to is most of understanding the compatibility story.

- **`schema_version`** — the shape of the record.
- **`engine_version`** — this implementation of the procedure, *including
  the order in which the seeded stream is consumed*.
- **`policy_version`** — the auto-mode decision table, or `"none"`.
- **`ruleset`** and **`rng.algorithm`** — pinned strings, compared on
  replay.

All are hand-bumped. That sounds fragile and is not: a gate in
`chargen/character_test.go` fails the build if a golden fixture's
replay-relevant shape moves while `engine_version` and `policy_version`
both stay put, which is what makes `task goldens` safe to run.

```bash
sed -n '22,41p' chargen/character.go
```

```output
const (
	// SchemaVersion identifies the character JSON schema.
	SchemaVersion = "0.33.0"

	// Ruleset is pinned: all rule citations resolve against this artifact.
	Ruleset = "Traveller5 Core Rules Book 1, Print Edition 5.1"

	// EngineVersion identifies this implementation of the generation
	// procedure, including the seeded stream's consumption order.
	EngineVersion = "0.45.0"

	// PolicyVersion identifies the auto-mode decision table in POLICY.md
	// (docs/PRD.md, CLI sketch). Changing the policy is a version bump.
	PolicyVersion = "0.25.0"

	// RNGAlgorithm names the recorded random stream: Go math/rand/v2 PCG,
	// seeded as documented at dice.New. The exact string is compared on
	// replay; changing it is a version bump.
	RNGAlgorithm = "math/rand/v2-pcg"
)
```

## 9. Replay — decider number three

`replay` is the reason the `Decider` seam is shaped the way it is. It
harvests every recorded choice event into a `replayDecider`, then calls
`Generate` again — the *same* engine, with a third answering strategy.
There is no separate replay code path that could drift from generation.

Then it verifies in two stages. `compareEvents` reports the first
disagreeing event by the record's own sequence number. But event agreement
is not the whole contract: derived values like the final credits, the skill
list and Fame are stored and recomputed, and no event carries them — so
`compareRecords` compares the marshalled records as well, catching a
derived value that drifted while every event feeding it stayed put.

```bash
sed -n '76,110p' chargen/replay.go
```

```output
func replay(stored Character, provenanceWaived bool) (Character, error) {
	decider := &replayDecider{}

	for _, event := range stored.Events {
		if event.Kind == EventChoice && event.Choice != nil {
			decider.choices = append(decider.choices, recordedChoice{seq: event.Seq, event: *event.Choice})
		}
	}

	replayed, err := Generate(Options{
		Seed:          stored.RNG.Seed,
		Name:          stored.Name,
		Career:        stored.Inputs.Career,
		Homeworld:     assignedHomeworld(stored),
		CurrentYear:   stored.Inputs.CurrentYear,
		RollHomeworld: stored.Inputs.RolledHomeworld,
		Decider:       decider,
	})
	if err != nil {
		return Character{}, fmt.Errorf("re-running the record: %w", err)
	}

	if err := compareEvents(stored.Events, replayed.Events); err != nil {
		return replayed, err
	}

	// The event logs agreeing is not the whole contract: "Derived values
	// are stored and recomputed on replay" (docs/PRD.md, JSON
	// conventions), and no event carries the final credits, skill list or
	// Fame. Comparing the marshalled records catches a derived value that
	// drifted while every event that fed it stayed put.
	if err := compareRecords(stored, replayed, provenanceWaived); err != nil {
		return replayed, err
	}

```

The `replayDecider` itself is short, and its doc comment records a
deliberate *removal* worth knowing about: it does not check that the engine
is asking the recorded question, because a record whose options no longer
match produces a choice event that no longer matches, and `compareEvents`
already reports that against the same sequence number. The pre-check could
not be made to fail any test the event comparison did not already fail, so
it was dropped rather than kept as untested redundancy. The range check
stays, because it distinguishes "the record and the engine disagree" from
"the decider answered wrongly".

```bash
sed -n '328,344p' chargen/replay.go
```

```output
func (d *replayDecider) Choose(c Choice) (int, error) {
	if d.next >= len(d.choices) {
		return 0, fmt.Errorf("%w: after event %d the engine asked %q, past the %d choices the record holds",
			ErrReplayDiverged, d.lastSeq(), c.ID, len(d.choices))
	}

	recorded := d.choices[d.next]
	d.next++
	d.last = recorded.event.Decider

	if recorded.event.Chosen < 0 || recorded.event.Chosen >= len(c.Options) {
		return 0, fmt.Errorf("%w: event %d: recorded the answer %d, outside the %d options",
			ErrReplayDiverged, recorded.seq, recorded.event.Chosen, len(c.Options))
	}

	return recorded.event.Chosen, nil
}
```

Replay is gated on provenance: a record from a different engine is refused
before anything is rolled, because a divergence reported against the wrong
build describes nothing. `--ignore-provenance` waives that for the one
record that cannot be regenerated any other way — a record made by a player
answering each choice, whose `policy_version` is `"none"`, so re-running
from its seed under the auto policy would produce a different character
entirely. It reads and never writes; it is not an upgrade path, and there
is deliberately no upgrade path.

```bash
sed -n '127,144p' chargen/replay.go
```

```output
// checkProvenance rejects a record this build cannot meaningfully re-run.
// Reporting it here rather than letting the run diverge matters: a record
// from another engine would fail at some arbitrary sequence number, and
// that number would describe nothing.
func checkProvenance(stored Character) error {
	var differ []string

	for _, field := range []struct {
		name, got, want string
	}{
		{"schema_version", stored.SchemaVersion, SchemaVersion},
		{"engine_version", stored.EngineVersion, EngineVersion},
		{"ruleset", stored.Ruleset, Ruleset},
		{"rng.algorithm", stored.RNG.Algorithm, RNGAlgorithm},
	} {
		if field.got != field.want {
			differ = append(differ, fmt.Sprintf("%s %s, this build %s", field.name, field.got, field.want))
		}
```

## 10. Rendering — two views of one record

`render` is a pure function of the record. `Sheet` produces the Character
Card; `History` produces the generation transcript, which is the one that
carries the page citations. Neither returns an error: a record read from
disk has had minimal validation, so a malformed event renders as a marked
line rather than panicking, and a lookup into an embedded table that has
since moved omits a line rather than failing.

That degradation is what makes the compatibility promise honest — a record
written by a released version *renders* under every later released version,
where "renders" means "can be read", not "produces identical bytes". The
sheet's layout is explicitly outside the promise.

```bash
sed -n '385,415p' render/render.go
```

```output
// narrative purpose of the event log (docs/PRD.md FR10): the character's
// biography in game terms.
func History(c chargen.Character) string {
	var b strings.Builder

	b.WriteString("# Generation Record\n")

	for _, event := range c.Events {
		b.WriteString(eventLine(event))
	}

	return b.String()
}

// eventLine renders one event of the transcript. Records come from disk
// with minimal validation, so a kind whose payload is missing renders as a
// marked malformed line instead of panicking.
func eventLine(event chargen.Event) string {
	switch {
	case event.Kind == chargen.EventStep && event.Step != nil:
		return fmt.Sprintf("\n## %s\n\n_%s_\n\n", event.Step.Name, event.Step.Cite)
	case event.Kind == chargen.EventThrow && event.Throw != nil:
		return throwLine(event.Seq, event.Throw)
	case event.Kind == chargen.EventChoice && event.Choice != nil:
		return choiceLine(event.Seq, event.Choice)
	case event.Kind == chargen.EventConsequence && event.Consequence != nil:
		return consequenceLine(event.Seq, event.Consequence)
	}

	return fmt.Sprintf("- #%d (%s) [malformed event]\n", event.Seq, event.Kind)
}
```

## 11. The whole thing, end to end

Generate a character from a fixed seed, render the card, and replay the
record. Everything below is reproducible: the same seed under the same
engine version yields the same 151 events.

```bash
d=$(mktemp -d) && go build -o "$d/t5chargen" ./cmd/t5chargen && cd "$d" && ./t5chargen new --auto --seed 7 -o character.json && ./t5chargen render character.json; rm -rf "$d"
```

```output
# Character Card

**Name**:

**UPP**: 764777

**Homeworld**: Regina A788899-C (Ph Pa Ri)

**Age**: 34 (Peak)

**Born**: Forday 229-1071

| Str | Dex | End | Int | Edu | Soc |
| --- | --- | --- | --- | --- | --- |
| 7 | 6 | 4 | 7 | 7 | 7 |

**Education**: University, Major Athlete, Minor Broker — did not graduate

**Career**: Citizen (3 terms), Job Seafarer, Hobby ACV

**Skills**: ACV-2, Actor-1, Admin-1, Animals-1, Aquanautics-2, Athlete-1, Broker-2, Bureaucrat-2, Career: Citizen-3, Computer-1, Rider-2, Seafarer-3, Trader-4, World: Regina-5

**Credits**: Cr45000

**Automatics**: Fame

**Citizen's Pension**: Cr5000 a year from age 66

**Status**: Fame 2 (Close Family)

---

Seed 7 (math/rand/v2-pcg) · schema 0.33.0 · engine 0.45.0 · policy 0.25.0

Ruleset: Traveller5 Core Rules Book 1, Print Edition 5.1
```

The transcript is the same record seen from the other side. Every throw
names its dice, its target, its modifiers and its page; every choice names
who decided, the whole option list, and which was taken; every consequence
names the event that caused it. This slice covers the end of education and
the first Citizen term.

```bash
d=$(mktemp -d) && go build -o "$d/t5chargen" ./cmd/t5chargen && cd "$d" && ./t5chargen new --auto --seed 7 -o character.json && ./t5chargen render --history character.json | sed -n "56,76p"; rm -rf "$d"
```

```output
## Select Career

_Book 1 p. 72 chart E1 step D_

- #43 policy chose "Citizen" of [Scholar, Entertainer, Citizen, Scout, Merchant, Spacer, Soldier, Agent, Rogue, Marine]: Select career — Book 1 p. 72 chart E1 step D

## Citizen: Begin (automatic)

_Book 1 p. 72 chart E1 panel 04_

- #45 policy chose "Serve the term in Citizen" of [Serve the term in Citizen, Trade School, Apprenticeship, College, Masters, Professors, Medical School, Law School]: Suspend the term to return to school? — Book 1 p. 59 (Later Education or Training); chart C p. 60

## Citizen: Term 1

_Book 1 p. 78 chart 04_

- #47 policy chose "Str" of [Str, Dex, End, Int]: Select the term's controlling characteristic — Book 1 p. 65 (Risk and Reward: Select the CC)
- #48 2D = 2+5 = 7 vs 7: success — Book 1 p. 78 chart 04 (Citizen Life vs Str, no mods per p. 65)
- #49 1D = 4 = 4 — Book 1 p. 78 chart 04 table E (roll A reroll if >3, then B, then C)
- #50 1D = 6 = 6 — Book 1 p. 78 chart 04 table E (roll A reroll if >3, then B, then C)
- #51 1D = 5 = 5 — Book 1 p. 78 chart 04 table E (roll A reroll if >3, then B, then C)
```

And the record verifies against itself:

```bash
d=$(mktemp -d) && go build -o "$d/t5chargen" ./cmd/t5chargen && cd "$d" && ./t5chargen new --auto --seed 7 -o character.json && ./t5chargen replay character.json; rm -rf "$d"
```

```output
replayed character.json: 151 events reproduced from seed 7
```

## 12. `audit/` — the documents have a compiler

The most unusual thing in the repository, and the easiest to misread as
over-engineering. The authority for this system is a printed book that
cannot be imported, executed, or diffed, so no test can check the code
against Book 1. What the project does instead is turn *claims about the
book* into structured documents, and then gate the documents mechanically.

`audit` is test-only, holds no rules, and is that machinery: every test
`COVERAGE.md` cites exists; every `ERRATA.md` interpretation is cited from
code; every choice point has a `POLICY.md` rule; no chart field is
transcribed and then read by nothing; no prompt shows a player an
identifier where the chart prints a name; `character.schema.json` describes
what the engine actually writes; every `ERRATA.md` quotation is on the page
it names (that last one needs the private Book 1 PDF, so it skips rather
than fails when the PDF is absent).

The compatibility gate is the clearest example of the pattern — and of why
its fixtures must never be regenerated.

```bash
sed -n '14,35p' audit/compat_test.go
```

```output
// The compatibility corpus: one record per released version, written by
// that version's own binary and never regenerated.
//
// docs/COMPATIBILITY.md makes two promises, and this is the gate under
// them. Prose saying older records still render is worth nothing on its
// own — each fixture here was produced by `go install ...@<its tag>`,
// and is the actual output of a version that no longer exists in the
// tree.
//
// `task goldens` must never touch these. It rewrites ./chargen and
// ./render, deliberately not this directory: a corpus a later engine can
// rewrite proves nothing about what an earlier engine wrote.
const corpusDir = "testdata/corpus"

// TestTheCorpusHoldsEveryReleasedVersion guards the corpus against
// quietly emptying. A compatibility gate with nothing in it passes.
func TestTheCorpusHoldsEveryReleasedVersion(t *testing.T) {
	records := corpusRecords(t)

	if len(records) == 0 {
		t.Fatal("the compatibility corpus is empty; it would pass while proving nothing")
	}
```

## Where to go next

`THEORY.md` is the companion to this document: it argues why the shapes
above are forced rather than showing how they run. `docs/PRD.md` is the
contract, `docs/COVERAGE.md` the rule-by-rule map, `docs/ERRATA.md` the 112
numbered readings taken where the printed rules are ambiguous, and
`docs/KNOWN_LIMITATIONS.md` the honest list of what the tool does not do.

Two things this trace hit that a reader should not have to rediscover are
filed in `.issues/`.

## Index

| # | Severity | Issue | Primary location |
| --- | --- | --- | --- |
| 1 | low | `history-transcript-choice-lines-inline-whole-option-lists` | `render/render.go:446` |
| 2 | low | `careerrun-interleaves-the-shared-term-loop-with-chart-specific-helpers` | `chargen/careerrun.go` |

**Total: 2 issues (0 critical, 0 high, 0 medium, 2 low)**

