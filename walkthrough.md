# t5chargen Walkthrough

*2026-08-25T23:57:58Z by Showboat 0.6.1*
<!-- showboat-id: d4ff96d9-7536-462c-b770-e85c7b8fe48d -->

## Overview

`t5chargen` is a command-line character generator for **Traveller5** (T5),
the 2013 edition of the tabletop RPG. It walks a character through the
game's *lifepath*: roll six characteristics, pick a homeworld, go to
school, serve one or more careers term by term, age, and muster out.

Three constraints shape every design decision in the repo, and they are
worth holding in mind throughout:

1. **The printed book is the authority.** Rules come from Traveller5 Core
   Rules Book 1, Print Edition 5.1. Every implemented rule carries a page
   citation in a doc comment at the site that implements it, and every
   place the printed rule is ambiguous is resolved in `docs/ERRATA.md`
   under a numbered interpretation.
2. **Generation is deterministic and replayable.** One seed produces one
   character, exactly. The record carries its own seed, RNG algorithm, and
   engine/schema/policy versions, so `t5chargen replay char.json` re-runs
   the record against itself and exits non-zero at the first divergence.
3. **Data and logic are separated.** Charts, tables, thresholds and labels
   live in embedded JSON. Orchestration and career-specific mechanics are
   typed Go. Neither side is allowed to hold the other's job.

It is Go, and it has **no dependencies** — standard library only.

```bash
cat go.mod
```

```output
module github.com/philoserf/t5chargen

go 1.26.6
```

No `require` block. That is deliberate, not incidental — it means the
engine's behaviour is a function of this repo's source and nothing else,
which is what makes the replay contract meaningful.

## Architecture

The layout divides into three kinds of thing: the CLI and engine, one
package per embedded chart, and the documents plus the tests that guard
them.

```bash
find . -maxdepth 1 -type d -not -name '.*' | sort
```

```output
./audit
./benefit
./calendar
./career
./chargen
./cmd
./dice
./docs
./education
./ehex
./fame
./interactive
./lifestage
./medal
./render
./ship
./skill
./world
```

Reading that list by role:

| package | role |
| --- | --- |
| `cmd/t5chargen` | the CLI: `new`, `batch`, `render`, `replay` |
| `dice` | the dice engine — xD rolls, Flux, target-number throws |
| `chargen` | the generation engine and the character record |
| `career` | the thirteen career definitions, data-driven |
| `render` | character sheet and history transcript output |
| `interactive` | the line-based front end for a human answering choices |
| `audit` | test-only: the gates that keep the documents honest |
| `benefit` `calendar` `education` `ehex` `fame` `lifestage` `medal` `ship` `skill` `world` | one embedded chart or vocabulary each |
| `docs` | the spec, COVERAGE, ERRATA, POLICY, the JSON Schema |

The ten single-chart packages are the bulk of the directory count and the
smallest part of the code. Nine hold one table transcribed from one page;
`ehex` is the odd one out, a pure vocabulary with no data file — the
extended hex digits T5 uses to write characteristics above 9.

## The chart pattern

Every chart package is built the same way: `go:embed` the JSON,
`sync.OnceValues` to parse it exactly once, and — in most of them — a
`validate()` call inside that once-body, so a malformed chart fails at load
rather than at use. `fame` is a representative example.

How much a chart is validated varies a great deal, and not by design:
`career` has 29 validators, `medal` has none, so loading `medal` proves
only that its JSON parses. Treat the pattern below as the intent rather
than a guarantee every package meets.

```bash
sed -n '47,67p' fame/fame.go
```

```output
//go:embed data/fame.json
var fameJSON []byte

var errBadTable = errors.New("invalid fame table")

// Load returns the embedded chart F.
func Load() (*Table, error) { return table() }

var table = sync.OnceValues(func() (*Table, error) {
	var t Table
	if err := json.Unmarshal(fameJSON, &t); err != nil {
		return nil, fmt.Errorf("fame table: %w", err)
	}

	if err := t.validate(); err != nil {
		return nil, err
	}

	return &t, nil
})

```

The data it loads is a plain transcription of the printed chart. Note the
`cite` field — every chart data file carries the page it came from, and
that citation travels into the event log.

```bash
head -13 fame/data/fame.json
```

```output
{
  "cite": "Book 1 p. 91 chart F",
  "stack_limit": 20,
  "no_eligibility_dice": 1,
  "noble_soc_multiplier": 1.5,
  "medals": {
    "XS": 0,
    "WB": 1,
    "MCUF": 1,
    "MCG": 2,
    "SEH": 3,
    "*SEH*": 4
  },
```

## Entry point: `cmd/t5chargen`

`main` does almost nothing but delegate. The interesting part is what the
usage string reveals about the tool's shape.

```bash
sed -n '36,44p' cmd/t5chargen/main.go
```

```output

const usage = `usage:
  t5chargen new [--auto] [--seed N] [--name X] [--career citizen] [--homeworld "UWP TC..."|random]
                [--current-year 1105] [-o file] [--force]
                (without --auto the player answers each choice; --auto applies POLICY.md)
  t5chargen batch --count N --auto [--seed N] [--name X] [--career citizen]
                  [--homeworld "UWP TC..."|random] [--current-year 1105] [-o dir/|file.jsonl] [--force]
  t5chargen render [--format md] [--history] character.json
  t5chargen replay [--ignore-provenance] character.json
```

`new` is the same engine either way: with `--auto` a fixed policy answers
every choice; without it, a human does. That symmetry is the central design
idea of the codebase, and the next section is where it lives.

Dispatch is a flat switch:

```bash
sed -n '/^func run(/,/^}/p' cmd/t5chargen/main.go
```

```output
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
```

One detail worth pausing on. The engine forbids unseeded randomness, but
the CLI has to get a seed from somewhere when the user doesn't supply one.
The comment marks the exception explicitly and explains why it doesn't
break the contract:

```bash
sed -n '50,54p' cmd/t5chargen/main.go
```

```output

// randomSeed draws a seed from the OS entropy source. This is the one
// deliberate exception to the repo's no-unseeded-randomness rule, which is
// engine-scoped: the CLI may pick the seed, and the chosen seed is recorded
// in the character's rng provenance so replay stays exact.
```

## The dice engine

`dice` is the only source of randomness. It wraps Go's `math/rand/v2` PCG
generator and records the individual die faces in roll order, so the event
log can show a player exactly what came up rather than only the total.

```bash
sed -n '23,52p' dice/dice.go
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

// Roll rolls n dice and sums them.
//
// "D (Capital D) indicates that a standard six-sided die is used. The number
// in front of the die tells how many of these dice to roll" and "2D. Roll
// two dice: results 2 to 12 (or 8D: Roll eight dice for results 8 to 48)"
// (p. 19).
func (r *Roller) Roll(n int) Roll {
	return r.RollMod(n, 0)
}
```

The seed expansion, the PCG algorithm, and the *order faces are consumed*
are all part of the engine's version contract. Change any of them and the
same seed produces a different character, so the change is an engine
version bump — which is why `engine_version` is recorded in every record.

## The choice funnel

Wherever a T5 rule says "select", the engine has to ask someone. There is
exactly one interface for that, and exactly one function that calls it.

`Decider` is the seam. It has two production implementations — the
interactive front end and the auto policy — and a third used only by
replay.

```bash
sed -n '/^\/\/ Decider resolves choice points/,/^}/p' chargen/decider.go
```

```output
// Decider resolves choice points. Interactive play and the auto-mode
// policy are its two implementations (docs/PRD.md, Decisions): every
// choice in the engine goes through this interface so replay can reapply
// recorded choices.
type Decider interface {
	// Choose returns the index of the selected option. An error refuses the
	// choice outright and ends generation: an interactive session the player
	// abandoned (docs/PRD.md, CLI sketch: "Interrupted interactive sessions
	// produce no output file"), or a replay whose recorded choice no longer
	// matches the choice the engine presents. Both are distinct from an
	// out-of-range index, which is a decider that answered wrongly rather
	// than one that declined to answer.
	Choose(c Choice) (int, error)

	// Kind identifies the decider in choice events (docs/PRD.md FR10:
	// "who decided — player or policy").
	Kind() DeciderKind
}
```

A `Choice` carries more than the question. Alongside the prompt, the
options, and the page cite, it can carry *decision aids* — `Scores`,
`ScoreLabel`, `Nth`, `Of` — that help a decider weigh the answer.

These are deliberately **not recorded in the event log**. That separation
matters: it means a policy can weigh a stake without parsing prompt text,
and rewording a prompt cannot change what character a seed produces.

```bash
awk '/^type Choice struct/,/^}/' chargen/decider.go
```

```output
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
```

And here is the funnel itself. Every choice point in the engine — career
selection, skill picks, waivers, muster-out rolls — goes through this one
function. It validates the answer, logs the choice event, and hands back
both the index and the event's sequence number so any consequence can name
what caused it.

```bash
sed -n '/^func choose(log \*Log/,/^}/p' chargen/character.go
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

Three failure modes, kept distinct on purpose:

- **No options presented** — an engine bug; the rule reached a state where
  it had nothing to offer.
- **The decider refused** (returned an error) — an abandoned interactive
  session, or a replay that no longer matches. Generation ends, named with
  the choice point that was declined.
- **The decider answered out of range** — a decider that replied wrongly,
  which is not the same thing as one that declined to reply.

## Generation: `chargen.Generate`

This is the spine. It follows the checklist the rulebook prints on p. 72
(chart E1) step by step: **A** generate characteristics, **B** determine a
homeworld, **C** education and training, **D** select career, **E** and
what follows — Fame, then muster out.

```bash
sed -n '/^func Generate(opts Options)/,/^}/p' chargen/character.go
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

	if err := runCareer(opts.Career, roller, &log, opts.Decider, &character); err != nil {
		return Character{}, err
	}

	// Fame is calculated over the finished record, not accumulated
	// (chart F p. 91), and muster out reads it — "one additional roll if
	// Fame 19+" (p. 68).
	if err := afterCareers(&character, roller, &log, opts.Decider, opts.CurrentYear); err != nil {
		return Character{}, err
	}

	return character, nil
}
```

Four things in that function are worth naming.

`checkSharedData()` loads the registries several open-selection rules read
from, up front, so a broken chart is a startup error rather than a
confusing mid-generation failure that blames the wrong thing.

`policyVersion` is `"none"` unless the *default policy itself* decided.
The field attests which decision table governed the run — an interactive
run is governed by no table, and says so.

`assigned := supplied(opts.Homeworld)` distinguishes a homeworld the caller
named from one the character chose. Replay needs that distinction: an
assigned world means step B had nothing to decide and offered one option,
while a chosen one was picked from chart B's thirty-four.

Fame is computed **over the finished record** rather than accumulated
during it, because muster out reads it.

## The event log

Every throw, choice, and consequence emits an event. This is not a debug
facility bolted on afterwards — it is the primary artifact. The rule in
`CLAUDE.md` is that a new mechanic is not done until its events render in
the history transcript and replay verifies them.

An `Event` is a tagged union of four shapes:

```bash
sed -n '/^type Event struct/,/^}/p' chargen/event.go
```

```output
type Event struct {
	Seq  int       `json:"seq"`
	Kind EventKind `json:"kind"`

	Step        *StepEvent        `json:"step,omitempty"`
	Throw       *ThrowEvent       `json:"throw,omitempty"`
	Choice      *ChoiceEvent      `json:"choice,omitempty"`
	Consequence *ConsequenceEvent `json:"consequence,omitempty"`
}
```

```bash
grep -n '^func (l \*Log)' chargen/event.go
```

```output
396:func (l *Log) Events() []Event {
446:func (l *Log) Step(name, cite string) int {
453:func (l *Log) Roll(roll dice.Roll, cite string) int {
465:func (l *Log) Flux(flux dice.FluxRoll, cite string) int {
477:func (l *Log) Throw(throw dice.Throw, mods []Mod, cite string) int {
492:func (l *Log) Choice(choice ChoiceEvent) int {
501:func (l *Log) Consequence(consequence ConsequenceEvent) int {
509:func (l *Log) append(event Event) int {
```

`Step` marks a checklist stage. `Roll` and `Flux` record dice. `Throw`
records a target-number check with its modifiers. `Choice` records a
decision and who made it. `Consequence` records what changed — and carries
the sequence number of the event that caused it, which is how the
transcript can say *this skill was awarded because of that choice*.

## The character record

The record is the deliverable. Its first six fields are pure provenance,
and they exist so a record can be re-run years later and checked.

```bash
sed -n '/^type Character struct/,/Inputs Inputs/p' chargen/character.go
```

```output
type Character struct {
	SchemaVersion string `json:"schema_version"`
	Ruleset       string `json:"ruleset"`
	EngineVersion string `json:"engine_version"`
	PolicyVersion string `json:"policy_version"`
	RNG           RNG    `json:"rng"`
	// Errata lists applied ERRATA.md deviations. Unlike policy_version
	// (always present — the POLICY.md version, or "none" when the run's
	// choices were not governed by the default policy), an empty list is
	// deliberately absent from the JSON: the PRD requires recording "any
	// applied deviations", so absence means none were applied.
	Errata []string `json:"errata,omitempty"`

	// Inputs are the generation inputs replay reconstructs the run from.
	Inputs Inputs `json:"inputs"`
```

`Inputs` is the subtlest part of the record, and each field is there
because replay would otherwise diverge. The comments are worth reading in
full — they are a catalogue of ways a re-run can go wrong.

```bash
sed -n '/^type Inputs struct/,/^}/p' chargen/character.go
```

```output
type Inputs struct {
	// Career is the --career force ("--career forces the first career
	// only", docs/PRD.md CLI sketch). It is not merely a preference: a
	// force holds the first career's option list to one entry, so a
	// replay that did not know about it would present a different list
	// and read the recorded index against it.
	Career string `json:"career,omitempty"`

	// CurrentYear is the Imperial year generation ended in, which fixes
	// the birth year (p. 58). Recoverable as birth year plus age, but
	// that is reconstruction; the input is recorded as an input.
	CurrentYear int `json:"current_year"`

	// HomeworldAssigned records that the caller named the homeworld, so
	// step B had nothing to decide and offered it alone. Without it a
	// replay cannot tell a world that was assigned from one the character
	// chose off chart B: the record stores the world either way, and
	// handing it back as an assignment would offer one option against an
	// index that names one of thirty-four.
	HomeworldAssigned bool `json:"homeworld_assigned,omitempty"`

	// RolledHomeworld records that the homeworld was determined on chart
	// B rather than assigned (p. 56: "Select or determine a Homeworld").
	// The resulting world is stored like any other, but the two dice that
	// found it came out of the seeded stream, so a replay that did not
	// know to roll them would diverge from the next throw onward.
	RolledHomeworld bool `json:"rolled_homeworld,omitempty"`
}
```

`RolledHomeworld` is the clearest case: the two dice that found the world
came out of the seeded stream, so a replay that didn't know to roll them
would be off by two faces from that point onward and every subsequent
throw would differ.

## Careers: data plus a mechanics seam

Thirteen careers is where the data/logic boundary earns its keep. Most of
what distinguishes one career from another is *tabular* — which
characteristics it checks to begin, what its Continue target is, which
skills each column offers. That goes in JSON.

What is left over is genuinely procedural: the Scholar's publications and
tenure, the Noble's exile and elevation, the Rogue's schemes and prison.
That goes in Go, behind a two-method interface.

```bash
sed -n '/^\/\/ careerMechanics is one career/,/^}/p' chargen/careerrun.go
```

```output
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

Two methods. Everything else a career needs is data — and where a career's
entry is the generic To Begin throw chart D describes, it does not
implement `begin` at all: it embeds `baseMechanics`, which reads the throw
off the definition's `BeginChecks`.

The registry wires each career name to a constructor returning both
halves:

```bash
sed -n '/^var careerRegistry/,/^}/p' chargen/careerrun.go
```

```output
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

The data side is `career.Definition`. Its fields read as a direct
transcription of the printed career chart — and the comments record which
chart cell each field came from.

```bash
sed -n '92,119p' career/career.go
```

```output
type Definition struct {
	Name string `json:"name"`
	Cite string `json:"cite"`

	// BeginChecks lists the characteristics the To Begin throw may check
	// (chart 05: "To Begin C1 or C2 or C3"); empty means Begin is
	// automatic (chart 04: "To Begin Auto").
	BeginChecks []string `json:"begin_checks,omitempty"`

	// RetryCheck is the characteristic for the career's Retry row
	// (chart 05: "Retry R&R C5"; interpretation I-8, ERRATA.md).
	RetryCheck string `json:"retry_check,omitempty"`

	// ControllingCharacteristics lists the characteristics available for
	// the career's Risk/Reward variant, in chart order (p. 64; for
	// Citizen, chart 04's "Citizen Life C1 C2 C3 C4"). Empty where the
	// variant rotates none (chart 03's "Risk & Reward Talent").
	ControllingCharacteristics []string `json:"controlling_characteristics,omitempty"`

	// The Continue target takes one of three forms, exactly one of which
	// is set: a fixed roll-low target (chart 04: "Continue 10-"), a
	// characteristic (chart 05: "Continue Int"), or the career's own
	// tracked value (chart 03: "Continue Fame"). The third consolidates
	// into a kind enum when the other value-target careers land.
	ContinueTarget         int    `json:"continue_target,omitempty"`
	ContinueCharacteristic string `json:"continue_characteristic,omitempty"`
	ContinueFame           bool   `json:"continue_fame,omitempty"`

```

Note `ContinueTarget` / `ContinueCharacteristic` / `ContinueFame`: exactly
one is set. The printed rule takes three forms across the charts — a fixed
roll-low target, a characteristic, or the career's own tracked value — and
the data models all three rather than flattening them into a number the
code would have to reinterpret.

`career/career.go` is the largest file in the repo, and a large share of it
is **validation** that runs at load time. A misspelled skill name in a
career's JSON is a build fault, not a runtime surprise:

```bash
grep -c 'func validate\|) validate' career/career.go
```

```output
30
```

## Replay

Replay is the repo's central claim, so it is worth seeing exactly how it
works. It is simpler than you might expect: harvest the recorded choices,
re-run `Generate` with a decider that hands them back in order, then
compare.

```bash
sed -n '/^func replay(stored Character/,/^}/p' chargen/replay.go
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

	return replayed, nil
}
```

Two comparisons, not one. `compareEvents` checks that the same throws and
choices happened in the same order. `compareRecords` then marshals both
characters and compares the bytes — because derived values like final
credits, the skill list, and Fame appear in no event, so a derived value
could drift while every event that fed it stayed identical.

The replay decider is the third `Decider` implementation:

```bash
sed -n '/^func (d \*replayDecider) Choose/,/^}/p' chargen/replay.go
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

The doc comment on `replayDecider.Kind()` is unusually candid about what
replay does *not* prove, and it is a good example of the repo's habit of
writing down the limits of its own guarantees:

> This makes attribution unverifiable, and deliberately so: the replay
> decider is not the original, so the only kind it can report is the
> recorded one, which then matches itself in `compareEvents`. A record
> whose decider fields (or `policy_version`, excluded for the same reason)
> were altered replays clean. Replay attests that the recorded choices
> rebuild the recorded character, not that the named decider would make
> them.

## Output: `render`

Two functions, two audiences.

```bash
grep -n '^func [A-Z]' render/render.go
```

```output
22:func Sheet(c chargen.Character) string {
336:func History(c chargen.Character) string {
```

`Sheet` is the character card a player uses at the table. `History` is the
generation transcript — every step, throw, choice and consequence with its
page citation, which is what makes a generated character auditable against
the printed rules.

## The audit package

`audit` holds no rules and ships no production code. It is the set of gates
that keep the documents honest.

```bash
grep -h '^func Test' audit/*.go | sed 's/(t \*testing.T) {//'
```

```output
func TestNoChartDataIsTranscribedAndForgotten
func TestFameAndMedalAgreeOnCodes
func TestCoverageNamesRealTests
func TestEveryInterpretationIsCited
func TestEveryChoicePointHasAPolicy
func TestEveryCareerHasCoverage
func TestCareerSectionsAreInChartOrder
func TestNoRowDefersToAClosedMilestone
func TestNoPromptShowsAnIdentifier
func TestEveryFixtureValidates
func TestExamplesValidate
func TestTheExamplesTrackTheEngine
func TestAnUnresolvableRefIsRefused
func TestAKeywordBesideARefStillApplies
func TestTheCheckerCatchesWhatItClaimsTo
func TestTheMinimalExampleIsMinimal
func TestEverySchemaPropertyIsExercised
func TestEachConsequenceKindKeepsItsShape
func TestEveryConsequenceKindIsAccountedFor
func TestEverySchemaVocabularyMatchesTheEngine
func TestAWholeNumberSatisfiesTypeNumber
func TestANonScalarConstIsCompared
func TestThePatternKeywordIsEnforced
```

Read those names as a list of ways documentation rots, each one closed:

- **`TestNoChartDataIsTranscribedAndForgotten`** — a field transcribed from
  the page, validated at load, and then read by nothing. Three shipped that
  way before this gate existed.
- **`TestCoverageNamesRealTests`** — a renamed test leaves COVERAGE.md
  claiming a rule is proven by something that no longer exists.
- **`TestEveryInterpretationIsCited`** — an ERRATA entry recorded and then
  lost track of.
- **`TestEveryChoicePointHasAPolicy`** — a choice the engine can present
  that the auto policy has no documented rule for.
- **`TestNoPromptShowsAnIdentifier`** — a prompt that shows a player a
  field name (`scholar_rank`) where the chart prints a rule. Every id in
  the chart data is snake_case and nothing the book prints contains an
  underscore, so an underscore in player-facing text is always a leaked
  identifier.
- The `schema` group — that `character.schema.json` describes what the
  engine actually writes, checked against every golden fixture.

## Putting it together

Generation is deterministic, so the same seed gives the same character
every time. Here is a full run:

```bash
go run ./cmd/t5chargen new --auto --seed 314 --current-year 1105 --force -o /tmp/demo.json && go run ./cmd/t5chargen render /tmp/demo.json | head -24
```

```output
# Character Card

**Name**:

**UPP**: 787767

**Homeworld**: Regina A788899-C (Ph Pa Ri)

**Age**: 29 (Adult)

**Born**: Thirday 263-1076

| Str | Dex | End | Int | Edu | Soc |
| --- | --- | --- | --- | --- | --- |
| 7 | 8 | 7 | 7 | 6 | 7 |

**Education**: College, Major Athlete, Minor Broker — did not graduate

**Career**: Citizen (2 terms), Job Mechanic, Hobby ACV

**Skills**: ACV-2, Actor-1, Admin-1, Broker-2, Bureaucrat-2, Computer-2, Mechanic-4, Trader-2

**Credits**: Cr30000

```

The full sheet continues with benefits, an automatic Fame award, and a
Citizen's Pension of Cr5000 a year from age 66. It ends with the
provenance contract in one line — the seed, the RNG algorithm, and the
three versions that together determine what that seed means:

```bash
go run ./cmd/t5chargen render /tmp/demo.json | tail -5
```

```output
---

Seed 314 (math/rand/v2-pcg) · schema 0.31.0 · engine 0.41.0 · policy 0.22.0

Ruleset: Traveller5 Core Rules Book 1, Print Edition 5.1
```

The history transcript shows the same run as the rules that produced it —
the checklist stages, the throws with their page cites, and each choice
with the decider that made it:

```bash
go run ./cmd/t5chargen render --history /tmp/demo.json | head -24
```

```output
# Generation Record

## Generate Characteristics

_Book 1 p. 72 chart E1 step A_

- #2 2D = 2+5 = 7 — Book 1 p. 56 chart A
  - #3 (from #2) Str = 7
- #4 2D = 2+6 = 8 — Book 1 p. 56 chart A
  - #5 (from #4) Dex = 8
- #6 2D = 3+4 = 7 — Book 1 p. 56 chart A
  - #7 (from #6) End = 7
- #8 2D = 4+3 = 7 — Book 1 p. 56 chart A
  - #9 (from #8) Int = 7
- #10 2D = 3+3 = 6 — Book 1 p. 56 chart A
  - #11 (from #10) Edu = 6
- #12 2D = 6+1 = 7 — Book 1 p. 56 chart A
  - #13 (from #12) Soc = 7

## Determine A Homeworld

_Book 1 p. 72 chart E1 step B_

- #15 policy chose "Regina A788899-C (Ph Pa Ri)" of [Alell B56789C-A (Ph Pa Ri), Boughene A8B3531-D (Fl Ni), Capital A586A98-F (Hi Cx), Dorannia E42158A-8 (He Ni Po), Efate A646930-D (Hi In), Feri B584879-B (Ph Pa Ri), Magash A400976-F (Va Hi Na In Cp), Hefry C200423-7 (Va Ni), Jenghe C799663-9 (Ni), Earth A867A69-F (Ga Hi), Lakou E779454-7 (Ni Da), Macene Belt B000453-E (As Ni), Knorbes E888787-2 (Ag Ri An), Preslin B430679-C (De Ni Na Po), Yori C560757-A (De Ri), Regina A788899-C (Ph Pa Ri), Ruie C776977-7 (Hi In), Tremous Dex B511411-C (Ic Ni), Uakye B439598-D (Ni), Vland A967A9A-F (Hi Cs), Wroclaw C5667BF-7 (Ag Ri), Menorb C652998-7 (Hi Po), Yorbund C7C6503-9 (Fl Ni), Traltha B590630-6 (De He Ni An), Dentus C979500-A (Ni), Vanzeti C52A531-C (Wa Ni), Syr Darya E55769C-5 (Ni Ag), Aramis A5A0556-B (He Ni Cp), Rhylanor A434934-F (Hi Cp), Raschev C8697C4-6 (Ri), Ara Pacis A437678-B (Ni), Roup C77A9A9-7 (Wa Hi In), Pax Rulin A402231-E (Ic Va Lo Cp), Space (Ds)]: Select a homeworld — Book 1 p. 58 (as assigned, selected, or random); chart B p. 56
```

Every line carries the page it came from. `#3 (from #2)` is a consequence
naming the throw that caused it — the causal link described earlier,
rendered. That long option list is the whole of chart B: the policy is
recorded as having chosen Regina *from thirty-four alternatives*, so the
record proves which choice was made rather than merely what resulted.

And the record verifies against itself:

```bash
go run ./cmd/t5chargen replay /tmp/demo.json
```

```output
replayed /tmp/demo.json: 109 events reproduced from seed 314
```

## The development gate

One command runs everything, and it also runs on pre-push:

```bash
grep -A6 '^  default:' Taskfile.yml | head -12
```

```output
  default:
    desc: Run check + test
    aliases: [all]
    cmds:
      - task: check
      - task: test

```

`check` is modernize, gofumpt, prettier, `go vet`, and golangci-lint
configured with `default: all` — every linter on, with an inline comment
justifying each one disabled. `test` is `go test -race ./...`.

Two conventions worth knowing before changing anything:

- **Never edit a golden fixture by hand.** `chargen/testdata` and
  `render/testdata` are the engine's own output, compared byte for byte.
  Regenerate with `task goldens` and read the diff — a fixture is only
  allowed to move when a change was meant to move it.
- **`main` is protected.** Every change lands through a branch and a pull
  request, including the owner's. The protection exists to catch an
  accidental commit on `main`, and an owner exemption would defeat it.

## Where to start reading

If you are picking this up cold, three files carry most of the meaning:

| file | why |
| --- | --- |
| `docs/PRD.md` | the spec — fixes scope, the replay contract, JSON conventions, milestones |
| `chargen/character.go` | `Generate`, the record, and the `choose` funnel |
| `docs/COVERAGE.md` | every implemented rule mapped to its page cite, and every deferral |

`docs/ERRATA.md` is the fourth: every place the printed rules were
ambiguous and a decision had to be made, numbered and argued. Reading it
is the fastest way to understand why the code is shaped as it is — most of
the non-obvious structure in `chargen` exists to serve one of those
interpretations.
