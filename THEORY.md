# THEORY.md — what you need to hold in mind to change this safely

This is not a description of the code. It is an attempt to write down the
understanding that makes the code make sense, for whoever picks it up next.
Read `docs/PRD.md` for what the system is supposed to do and
`walkthrough.md` for how it runs. This document is about why it is shaped
the way it is, and which of its shapes you can change without breaking
something you did not know was load-bearing.

## What the system is actually modelling

The obvious answer is "a Traveller5 character." That answer will lead you
astray.

The system models **a procedure printed in a book**, and its product is not
a character but a _defensible derivation_ of one. Traveller5 character
generation is a lifepath: a player walks a checklist (Book 1 chart E1,
p. 72), rolling dice and making choices, and the character that falls out
at the end is the residue of forty or a hundred small decisions. The
interesting object is the walk, not the residue.

Once you see that, a lot of otherwise-odd decisions become forced. The
character record embeds its entire event log, and that log is bigger than
everything else in the file combined. The Markdown output has two forms —
a character sheet and a _generation transcript_ — and the transcript is
the one that carries page citations. `replay` does not check that a seed
still produces a valid character; it checks that the recorded walk
reproduces itself step for step. The engine's core loop is not "compute a
character" but "emit events which incidentally accumulate into one."

The domain vocabulary is the book's, and it is used as if settled, because
within the book it is. A **term** is four years of a career. A **Continue**
throw at the end of one decides whether there is another. A **controlling
characteristic** (CC) is the attribute a career's Risk and Reward rolls
check, rotated so none is reused until all have been. **Muster out** is
what happens when the careers stop. **Flux** is a signed die mechanic
running -5 to +5 — "Light Die minus Dark Die" (p. 261), and `light` and
`dark` are the variable names in `dice/flux.go`. **eHex** is the extended
hex notation that lets a characteristic above 9 fit in one character of a
UPP. When you meet one of
these words in the code it means exactly what the page means, and the doc
comment quoting the page is at the implementation site because the page is
the specification.

There are 106 numbered interpretations in `docs/ERRATA.md`. That number is
the single most informative fact about this domain: the printed rules are
ambiguous _constantly_, and a working implementation is mostly a sequence
of decisions about what a sentence meant. The interpretations are not
footnotes. They are the design.

## The three invariants

Everything structural in this repo defends one of three properties. If you
are about to change something and you cannot tell which one it serves, find
out before you change it.

**One seed produces one character, exactly.** This is enforced at three
seams rather than by discipline. All randomness comes from a single
`dice.Roller` wrapping a seeded PCG stream, consumed sequentially — the
package doc says so and the type is explicitly not concurrency-safe,
because concurrent consumption would reorder the stream. Every choice goes
through one function, `choose()` in `chargen/character.go`. And nothing
in the engine reads a clock. The subtler half is _ordering_: maps in this
codebase are for lookup and never for iteration. `career.Available()` is a
literal slice in chart order; `skill.Names()` ranges a map and then sorts
before returning. If you introduce a `range` over a map whose order reaches
an option list, you have broken determinism in a way no test will
obviously catch and every stored record will silently fail to replay.

**Every throw, choice, and consequence is recorded, with its page cite.**
`CLAUDE.md` states the rule that a mechanic is not done until its events
render in the transcript and replay verifies them. The consequence events
carry the sequence number of the event that caused them, which is what lets
the transcript say _this skill was awarded because of that choice_. This is
why the `Log` methods are `Step`, `Roll`, `Flux`, `Throw`, `Choice`,
`Consequence` — those are the six things the book's procedure does.

**A stored record can be re-run and checked years later.** This is the
provenance contract, and it is the reason for the four version fields.
`schema_version` describes the record's shape, `engine_version` the
generation logic and the dice stream, `policy_version` the auto-mode
decision table, plus the `ruleset` string and the RNG algorithm and seed.
Read `docs/POLICY.md`'s version history and notice what it records: not
merely what each bump changed but _which records it moves_. "offered at the
beginning of every term of every career and so shifting every subsequent
event in every record" is a blast-radius annotation. The versions are
claims about which stored files remain replayable, which is why bumping one
is a deliberate act and not bookkeeping.

## The choice funnel, and why it is one function

`Decider` is the load-bearing abstraction. It has two methods and three
implementations: the interactive front end, the fixed auto policy, and a
replay decider that hands back recorded answers in order.

The reason this is a single interface with a single call site rather than
a callback here and a flag there is that **replay is only possible if
every decision is capturable in the same shape**. A choice that bypassed
`choose()` would be invisible to the log and would diverge on re-run.

Two details of `Choice` are worth understanding before you extend it.

First, `Scores`, `ScoreLabel`, `Nth`, and `Of` are engine-provided decision
aids that are deliberately **not recorded**. Their purpose is to let a
policy weigh a stake without parsing prompt text. The consequence is the
part that matters: because they are not recorded, rewording a prompt cannot
change what character a seed produces. If you record them, you have coupled
presentation to output and every prompt edit becomes a version bump.

Second, `choose()` distinguishes three failure modes that a naive
implementation would collapse: an empty option list (an engine bug — a
rule reached a state with nothing to offer), a decider that _refused_ by
returning an error (an abandoned interactive session, or a replay
divergence), and a decider that answered out of range (a decider that
replied wrongly). Refusal and wrong answer are different events with
different causes, and the error text says which.

Around this sits a fourth interface, `Watcher`, which is worth studying as
a small masterclass in defending an invariant at a type boundary. It exists
so the interactive front end can narrate events as they occur. Its doc
comment enumerates why it cannot affect a character: it gets copies, it
returns nothing, it is consulted after the event is already recorded, and
neither `DefaultPolicy` nor the replay decider implements it. That is four
independent reasons, written down, for a feature that could have been a
one-line callback. Match that standard when you add a seam near the engine.

## Where the data/logic boundary actually falls

`CLAUDE.md` says tables, thresholds, and labels are embedded data;
orchestration and career-specific mechanics are typed Go; no rules language
in the data. That is true, but the useful version is more specific.

**The data holds what the chart tabulates. Go holds what the chart's
footnotes say.** `career.Definition` reads as a transcription of a printed
career chart, and its most instructive field group is the Continue target:
`ContinueTarget`, `ContinueCharacteristic`, and `ContinueFame`, exactly one
of which is set. The printed rule takes three forms across the thirteen
charts — a fixed roll-low number, a characteristic, or the career's own
tracked value — and the data models all three rather than flattening them
into a number the code would have to reinterpret. That is the boundary
working: the data stays a transcription, and the code branches on shape.

What is left over goes behind `careerMechanics`, an unexported interface
with two methods, `begin` and `resolveTerm`. Thirteen careers implement it.
The Scholar's publications and tenure, the Noble's exile, the Rogue's
schemes and prison, the Craftsman's Masterpieces — all genuinely
procedural, none expressible as a table.

Two optional interfaces extend that seam by type assertion rather than by
widening it, and both are worth knowing about because they are the pattern
you should follow rather than adding a method every career must stub.
`characteristicRaiser` notifies a career when a characteristic award lands,
and exists for exactly one rule: chart 11's "each increase in Soc during
CharGen awards a Land Grant." Look at its guard — the notification fires
only when the increase _actually landed_, because the p. 68 maximum can
refuse one, and a refused increase must not award a grant. That
three-line check is a rule, not defensive coding.

The boundary's honest weakness is that the data grew a validation language.
`career/career.go` carries 29 `validate*` functions, and load-time
validation across the eleven chart packages runs from 29 down to zero
(`medal` validates nothing). Nothing in the gate requires a chart package
to validate its chart, so how much checking a chart gets is a matter of
who wrote it and when.

## Apparent duplication that is not

Two places will look like copy-paste until you find the distinguishing
detail, and in both cases the detail is the whole point.

`chargen/education.go` and `chargen/assignedschool.go` both put a character
through schooling during a career. They are separate because of one clause:
Later Education substitutes a process "for the entire term" (p. 59), while
an assigned school — ANM School, Command College — is sited _inside_ a term
the character is already spending. One costs a term and one does not. If
you unify them you will silently start charging four years for a Command
College year that chart 07 expressly places in Year 1 of a term the officer
is serving anyway.

`education/data/education.json`'s Graduation column looks like a reward and
is not. Page 62 settles it: "a character with Edu=9 can function at the
equivalent of a Masters in Educational situations even if he does not have
the formal diploma." The Edu values _are_ credential-equivalents — 8
Bachelors, 9 Masters, 10 Doctor, 12 Professor — which is why the chart
writes `Edu=8` and not `Edu+8`. Schooling moves you _to_ the level its
credential represents. Reading that as "at or above, award +1" turns
graduation into a ratchet, and that single misreading was the root cause of
a run of play-found bugs fixed across PRs #67–#75 (a character taking the
Service Academy three times, and College after University). Interpretations
I-98 through I-105 record the repair. If you touch chart C, re-read I-105
first.

## The seams

**The rulebook** is the most important external dependency and it is not in
the repo. Book 1, Print Edition 5.1, lives at `~/Documents/Traveller/T5/`
and is not redistributed. `CLAUDE.md` forbids implementing from memory or
from the 2008-preliminary extracts in that collection's `Archive/` — locate
a topic there if you must, verify in Book 1. Every rule carries a page cite
so a reader with the book can check the claim. This is the seam where
correctness is ultimately decided, and it is the one your tests cannot
reach.

**The CLI/engine boundary** carries one deliberate exception to the
determinism rule, marked as such: `randomSeed()` draws from OS entropy when
the user supplies no seed. The rule is engine-scoped — the CLI may pick a
seed, and the chosen seed is recorded, so replay stays exact.

**`chargen` and the front ends.** `render` and `interactive` both consume
the record and neither can influence it. Every prompt and option a player
sees originates in the engine, because that text is recorded content —
which is why a gate reading only auto-generated fixtures still covers it.
`interactive` does own a little player-facing text of its own, but only
presentation-only annotations on options the engine supplied (`"  [needs a
waiver]"`, `"  [automatic]"`, from the `Scores` a `Choice` carries). Those
are never recorded, so they sit outside both the identifier gate and the
version contract. Keep it that way: text that reaches a record belongs to
the engine.

**The golden fixtures** are the seam between a change and its blast radius.
Fourteen JSON records and eighteen Markdown files, regenerated only by
`task goldens`, never edited by hand, excluded from prettier so a formatter
cannot decide what a character record looks like. In a system where one
clause change shifts every downstream event, the fixture diff is the only
practical way to see what a change did. Read it; do not accept it.

**`audit`** is the seam between the code and the documents, and it is the
most unusual thing in the repo. It is a test-only package holding no rules,
whose entire job is to fail the build when a document stops describing the
code: that every test COVERAGE.md cites exists, that every ERRATA
interpretation is cited, that every choice point has a POLICY.md rule, that
no chart field is transcribed and then read by nothing, that no prompt
shows a player an identifier where the chart prints a name, that the JSON
Schema describes what the engine actually writes.

## The habit that explains the documents

The repo treats its own fallibility as a first-class artifact, and it does
so _in the place where being wrong would recur_. This is the thread that
ties the documents together, and once you see it you will stop reading them
as ordinary documentation.

The dead-data gate's doc comment names the field that escaped it and
explains why name-matching cannot catch a field whose name collides with a
read one. `replayDecider.Kind()` documents what replay does **not** prove:
a record whose decider fields were altered replays clean, so replay attests
that the recorded choices rebuild the recorded character, not that the
named decider would make them. `MusterOutM2` is transcribed specifically so
its disagreement with the career pages stays visible. `docs/MILESTONE-4.md`
opens by saying the milestone was called complete once, was wrong, and is
closed on the second attempt. The PRD carries a "Verified at
implementation" amendment recording that the UWP alone does not suffice for
homeworld skills. The README has a paragraph about its own earlier draft
being wrong.

This is why the gates in `audit` exist at all. This repo was built in six
days across 79 commits, and at that velocity a document that merely
_claims_ to describe the code will drift within a week. So the claims that
can be checked mechanically are checked, and the ones that cannot are
written down as known limits.

Notice the commit subjects while you are here: "a program is attempted
once, which is what stops the Edu ratchet"; "do not offer a title the
character has no way to hold"; "say what a row costs, not that it is out
of reach". The log is written in the domain's voice, describing what
changed for the character rather than what changed in the code. That is a
convention worth continuing — it makes `git log` a rules changelog.

## What the system is shaped to accommodate

**Another career.** This is the best-supported change. Add the definition
to `career/data/`, implement `careerMechanics`, register it in
`careerRegistry`, add a COVERAGE section in chart order. The gates will
tell you what you forgot.

**Another decider.** The interface is two methods. A weighted-random
policy, or one reading answers from a file, drops in without touching the
engine — provided it does not become a second thing that decides _what
options exist_, which is the engine's job.

**Another interpretation.** Number it in ERRATA, cite it in COVERAGE, cite
it at the implementation site. The gates enforce two of the three.

**Another output format.** `render` reads the finished record and has no
path back into the engine.

What would require rethinking something fundamental:

**Non-human characters** are excluded by the PRD, and the exclusion runs
deeper than a flag. Characteristic C6 is Soc for humans and Caste for some
sophonts; the Scholar's chart branches on C5=Tra; several deferred rules
are deferred _because_ they are non-human. Adding sophonts means the
characteristic set stops being fixed, which reaches the UPP, the eHex
encoding, every career's controlling characteristics, and chart C.

**Concurrency inside a generation** contradicts the sequential-stream
requirement. Batch mode parallelises across characters, each with its own
`Roller`, which is the only shape that works.

**Making the engine incremental** — pausing a lifepath and resuming it —
would break the assumption that `Generate` runs start to finish from a
seed. Replay works by re-running everything.

If a new requirement arrives, the maintainer who understands the theory
asks first: _does this add a decision?_ If yes, it needs a `ChoiceID`, a
POLICY.md rule, and a version bump, and it will move every fixture. The
maintainer who does not understand the theory adds a parameter, reads a
chart value directly instead of through its package, or "simplifies" one of
the optional interfaces into a required method — and the failure will
surface as a fixture diff they cannot explain.

## Uncertainties, and where the theory is thin

Everything below is inference from code, tests, comments, and history
unless marked otherwise. Where a claim rests on something the author said
rather than on the artifact, I say so.

**COVERAGE's Status column has drifted, and I am confident about this
one.** Five rows still say `deferred (M3)` for rules that shipped in
milestones 3 and 4 — Trade/Art/Science cells, generic Risk/Reward, To Begin
throws with retry, Rank/Commission/Promotion, and Starship Skill cells.
Verified against the code. The seven rows
marked `deferred (M6)` are correct. What makes this worth flagging in a
theory document rather than only as a bug is _why it was invisible_: line
220 and line 343 carry textually identical claims — "(errors if selected)"
— and only one is true. `EntryCapital` does still error, deliberately;
`EntryStarship` has not since `groupCells` gained it. When the true and the
false claim are the same sentence, review cannot distinguish them. Treat
COVERAGE's Status column as reviewed rather than gated, and verify a
deferral against the code before planning work from it.

**"The event log is the primary artifact" is nearly but not entirely
true.** `compareRecords` exists because derived values — final credits, the
skill list, Fame — appear in no event, so the log agreeing is not
sufficient for replay. The record is therefore not a pure projection of the
log. I believe the intended reading is that the log is primary _for the
procedure_ and the record is primary _for the result_, but I am inferring
that from the shape of the comparison rather than from a stated rule.

**There is no codebase-wide convention for out-of-range values**, and I
cannot tell whether that is an oversight or a decision never written down.
Twelve sites answer the question at least six different ways, four of them
by panicking, and `chargen/fame.go:210` clamps a negative where `:232` does
not — a disagreement inside one file, which reads more like accretion than
intent. The convention that settles it is now recorded in `CLAUDE.md`
under "Values outside the rules' range"; the sites that predate it have not
all been brought into line.

**The code-to-data direction of the skill vocabulary is unguarded**, and
this looks like a genuine gap rather than a decision. Data-to-code is
thorough — every transcribed skill name is validated against chart MS at
load. But five Go constants name chart MS group headings
(`skill/skill.go:176-180`) and `skill.validate()` checks only the list
length and the cite, so a renamed group in the data still loads clean.
`awardNewTrade` then reads the resulting empty list as chart 01's "all
trades already held" and silently drops the benefit.

**The `Err()` accessors on `skill` and `medal` are half a pattern.**
They exist because those two packages return bare values rather than
`(value, error)`, so a load failure has no return path. `chargen.checkSharedData`
calls `skill.Err()`; nothing calls `medal.Err()`. My reading is that
`checkSharedData` is the right idea applied to two charts out of eleven and
never finished — but it is equally consistent with a deliberate decision
that only the shared registries need an early check, and I cannot tell
which from the code.

**Milestone 6 is genuinely open, not merely unimplemented** — and this is
the author's stated position, not my reading of the code. Page 134's
container-skill rules — Knowledge-Knowledge-Skill progression, the
Knowledge-6 maximum, Career and World knowledges — need design decisions
before code. Implementing the container rule adds a choice point to every
container award, which needs a POLICY.md rule for what auto picks; "terms
lived on a world" is an open interpretation; and Musician reads as a
container by pp. 134 and 157 but is not one in chart MS. Do not treat those
seven COVERAGE rows as a to-do list.

**On the audit backlog:** a systematic package-by-package audit produced
46 findings, kept in a local `.issues/` directory that is ignored globally
and so is not part of this repository. They were verified when filed; their
suggested fixes were not, and two were later proven wrong. If you have that
directory, weight the findings and re-derive the fixes. If you do not, the
uncertainties above are the part worth keeping.
