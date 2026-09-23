# THEORY.md — the understanding this repository runs on

This is not a description of the files. It is the account of _why_ the code is
shaped as it is, written for whoever changes it next. `docs/PRD.md` says what
the system must do; `WALKTHROUGH.md` walks one run end to end; `CLAUDE.md`
states the working rules. This document is the layer underneath all three: the
handful of commitments that, once you hold them, make every otherwise-arbitrary
decision here look forced.

Read it before changing anything structural. Most of what looks like ceremony
in this repository is load-bearing, and the parts that genuinely are incidental
are named as such below.

## 1. What is actually being modelled

The tempting answer is "a Traveller5 character". Take it and you will make
wrong changes confidently.

What the system models is **a procedure printed in a book**, and what it
produces is not a character but a _defensible derivation_ of one. Traveller5
character generation is a lifepath: a person walks the Master Chargen Checklist
(Book 1 chart E1, p. 72), rolling dice and answering questions, and the
character that falls out at the end is the residue of a hundred small
determinations. The interesting object is the walk. The character is its
by-product.

Everything expensive in this codebase follows from taking the walk seriously:

- The character record embeds its own complete event log, and that log is
  larger than every other field combined. A 34-year-old Citizen from seed 7
  carries 151 events.
- There are two renderings, not one: a character sheet and a _generation
  transcript_ (`render --history`), and only the transcript carries the page
  citations. The sheet is for playing; the transcript is for arguing.
- `replay` does not ask whether a seed still yields a valid character. It asks
  whether the recorded walk reproduces itself, step for step, and stops at the
  first event where it does not.
- Every implemented rule carries the printed sentence it implements, quoted in
  a doc comment at the site. Where the printed rule is ambiguous, the reading
  taken is numbered in `docs/ERRATA.md` and cited from the code.

A sibling project that only wanted characters would have none of this. It would
compute six characteristics, loop some terms, and print a sheet. The reason
this one is 20,000 lines of non-test Go is that it is building an _audit trail_
and the character comes along for free.

The corollary is the single most useful sentence in this document: **the record
is the product.** Rendered output is derived, disposable, and explicitly not
covered by any compatibility promise (`docs/COMPATIBILITY.md`, "During beta").
The JSON is what a user keeps and what a bug report attaches.

## 2. The event log, and why the engine's core loop is inside out

`chargen/event.go` defines four event kinds: `step`, `throw`, `choice`,
`consequence`. Events carry monotonic sequence numbers from 1, and a
consequence names the sequence number of the throw or choice that caused it.
That `Cause` field is not decoration — it is what lets the transcript say _why_
a characteristic moved, and what lets a reader follow the derivation backwards.

The consequence of this design is that the engine's inner loop is not "compute
the character" but "emit events which incidentally accumulate into a
character". Concretely: `Character.advanceYears` is the _only_ place a
character ages, and it exists as one function not for tidiness but because
aging is a rule about elapsed time (chart A, p. 89), not about any of the nine
things that elapse it. Put a `c.Age += 4` anywhere else and you have silently
skipped an Aging Check and emitted no event saying so.

This is the shape to preserve when adding a mechanic. `CLAUDE.md` states the
rule as "New mechanics are not done until their events render in the history
transcript and replay verifies them", and that is not a process nicety: an
effect with no event is invisible to replay, so it is a value nothing verifies,
in a record whose whole claim is that everything in it was verified.

A consequence kind is added, never repurposed. `ConsequenceCharacteristicFloored`
exists because `CLAUDE.md`'s clamping rule (§7 below) demands that a clamp be
visible, and the only way to make it visible is to give it an event.

## 3. The Decider seam

This is the narrowest and most important waist in the system.

Every point where the printed rules ask a person something goes through
`chargen.Decider` — one method, `Choose(Choice) (int, error)`, plus `Kind()`
saying who is answering. There are about forty such choice points, each a
`ChoiceID` constant in `chargen/decider.go` carrying the sentence from Book 1
that creates it.

There are exactly three implementations, and the fact that there are three
rather than two is the point:

| Implementation          | Where                            | What it does                                         |
| ----------------------- | -------------------------------- | ---------------------------------------------------- |
| `chargen.DefaultPolicy` | `chargen/policy.go`              | the fixed, total, versioned auto-mode decision table |
| `interactive.Decider`   | `interactive/`                   | asks a person, line by line                          |
| `replayDecider`         | `chargen/replay.go` (unexported) | replays the recorded answers                         |

Replay works _because_ it is a Decider. It is not a separate code path through
the engine; it is the same engine with a third answering strategy. That is why
the replay contract can be as strong as it is: there is no "replay mode" that
could drift from generation.

Three properties fall out of the seam and are easy to break by accident:

**An answer is an index.** `ChoiceEvent.Chosen` is an integer into
`ChoiceEvent.Options`. Reorder the options a choice point presents and every
record already written now means something different — the same digit selecting
a different thing. `POLICY.md`'s own 0.10.0 entry records this hazard having
been hit.

**A prompt is part of the record.** `choose` (`chargen/character.go:983`)
writes the presented prompt, option list and citation into the event, and
`compareEvents` compares events as marshalled JSON. So rewording a prompt
invalidates every existing record. This is why `t5chargen help` carries the
long explanation of what `--auto` does and does not do: help text is free to
change, prompt text is not. If you find yourself wanting to explain something
in a prompt, the answer is always the help text.

**Decision aids are deliberately outside the record.** `Choice.Scores`,
`Choice.ScoreLabel`, `Choice.Nth`, `Choice.Of` are engine-computed hints — the
current characteristic values behind a controlling-characteristic pick, whether
a career is entered automatically, whether an education row will cost a waiver
attempt. They are shown to people and read by the policy, and they are _not_
logged. That asymmetry is what lets a front end get better at helping without
invalidating a single record. Keep new decision data on that side of the line.

The `Watcher` interface (`newLog`, `chargen/character.go`) is the same idea in
the other direction: a Decider may opt in to seeing events as they are logged,
which is how the interactive session shows a running "age 19 · UPP C469B7 · 2
skills · Select Career" banner above each question. It is an observation
channel, not a control channel, and nothing in the engine's behaviour may
depend on whether anyone is watching.

## 4. Determinism, and the three versions a record stamps

One seeded PCG stream (`math/rand/v2`), created in `Generate` and threaded
through everything as `*dice.Roller`. No wall-clock time. No unseeded
randomness in the engine at all. The single deliberate exception is
`randomSeed` in `cmd/t5chargen/main.go`, which draws a seed from OS entropy
when `--seed` is absent — and the drawn seed is immediately recorded, so the
exception never escapes the CLI boundary.

A record carries three independent versions, and understanding what each
entitles you to is most of understanding the compatibility story:

- **`schema_version`** — the shape of the record. Bumped when the current
  engine's output would be invalid against the previous schema, or vice versa.
  Not bumped for a narrowing constraint that every existing record already
  satisfies (`docs/COMPATIBILITY.md` records the `upp` pattern as the case that
  settled the rule).
- **`engine_version`** — this implementation of the _procedure_, explicitly
  including **the order in which the seeded stream is consumed**. This is the
  subtle one. Adding a roll, removing one, or reordering two rolls changes
  every character from every seed, even if not one rule changed meaning.
- **`policy_version`** — the auto-mode decision table. Deliberately _not_ part
  of the provenance check, because replay reapplies recorded choices and never
  consults the policy. This is also why fixtures stamped `"none"` (generated by
  test deciders) replay at all.

The rule that `engine_version` covers stream consumption order is not left to
discipline. `chargen/character_test.go` around line 100 holds the gate: if a
golden fixture's replay-relevant shape moves while `engine_version` and
`policy_version` both stay put, the test fails with an explanation of why the
bump is needed. That gate is the reason `task goldens` is safe to run — without
it, regenerating fixtures would launder an unversioned engine change into a
green build.

The second half of the same guarantee is `audit/testdata/corpus`: one record
per released version, written by that version's own binary and never
regenerated. `task goldens` names `./chargen ./render` explicitly so it cannot
reach the corpus. A corpus a later engine can rewrite proves nothing about what
an earlier engine wrote — that sentence is the whole justification for the
directory's existence, and anyone "tidying" the test data will delete the only
evidence the compatibility promise has.

## 5. Two rules about the record that read as contradictory and are not

`docs/COMPATIBILITY.md` promises that a record written by a released version
**renders** under every later released version, and separately that **replay
stays pinned** to the engine that wrote the record. Those look like opposite
commitments about the same file, and the difference is what each operation
claims.

Rendering reads the record and formats it; it is a pure function of the record
(`render` returns strings, never errors, and degrades to omitting a line rather
than failing when an embedded table has moved). "Renders" means "can be read",
not "produces identical bytes" — the sheet's layout is explicitly outside the
promise. Replay re-runs the _generation procedure_, so it needs the procedure
it is checking. A cross-engine replay could only ever report a difference that
is not a defect.

`--ignore-provenance` is the escape hatch, and its doc comment is worth reading
before you touch it: it exists for the one record that cannot be regenerated
any other way — a record made by a player answering each choice, whose
`policy_version` is `"none"`, so re-running from its seed under the auto policy
produces a different character entirely. It reads and never writes. It is not
an upgrade path, and there is deliberately no upgrade path.

## 6. Charts are data; mechanics are Go

`CLAUDE.md` states the boundary — "tables, thresholds, and labels are embedded
data files; orchestration and career-specific mechanics are typed Go. No rules
language" — and the boundary holds in practice. `career/data/*.json` is 120 KB
of skill tables, target numbers, benefit rows, rank titles and citations. What
it never contains is conditional logic: the keys are `kind`, `name`, `roll`,
`money`, `benefit`, `target`, `mod`, `cite`. Where a career needs a rule that
cannot be expressed as a table cell, it gets Go.

The plug point is `careerMechanics` plus `careerRegistry` in
`chargen/careerrun.go` — thirteen names mapped to constructors
(`newCitizen`, `newScholar`, …), each returning a definition loaded from JSON
and the Go type carrying that chart's peculiarities. `careerrun.go` itself is
the generic term loop: controlling characteristic, skills, risk and reward,
continue roll, aging, career change, muster out. It is the largest file in the
repository (1,580 lines) and it is large for a defensible reason — the thirteen
charts share one procedure with thirteen sets of exceptions, and the
alternative to one long shared loop is thirteen slightly divergent copies of
it.

A career listed in `career.Available` with no registry entry is a wiring bug,
not a user error, and the code says so. That distinction — internal
inconsistency versus bad input — recurs throughout and is worth matching:
`ErrUnknownCareer` and `ErrCareerUnavailable` are user errors that exit 2 via
`isUsageError`; a missing registry entry is not.

The other ten packages (`benefit`, `calendar`, `career`, `education`, `ehex`,
`fame`, `lifestage`, `medal`, `ship`, `skill`, `world`) are each one chart or
vocabulary, all built on the same `go:embed` plus `sync.OnceValues` pattern
with load-time validation. That uniformity is real and worth preserving; the
package _boundaries_ between them are mostly incidental, chosen so each chart
has an obvious home rather than because anything depends on the split.

## 7. Out-of-range values: one answer, not one per site

`CLAUDE.md` fixes this and it is a genuine invariant rather than a style
preference:

- A value a **caller** supplies — a day, a dice count, an eHex digit, a UWP —
  is **refused with an error**. Never repaired. `Options.Homeworld` is the
  clearest case: only the all-zero struct falls back to the default; a
  partly-filled homeworld is validated and rejected.
- A value the **engine derives** **clamps to the rule's own floor and emits a
  consequence saying so.** The record is the product; a clamp nobody can see is
  worse than one they can.
- Nothing exported panics.

The reason the split falls where it does: a caller can be told, and telling
them is cheaper than guessing. The engine, mid-lifepath, has nobody to tell and
no defensible way to abort — so it does the arithmetic the rule implies and
leaves an event behind for the auditor.

## 8. Documents are artifacts, and `audit/` is their compiler

This is the most unusual thing about the repository and the easiest to
misread as over-engineering.

The authority for this system is a printed book that cannot be imported,
executed, or diffed. No test can check the code against Book 1. So the project
does the next thing available: it makes _claims about the book_ into structured
documents, and then gates the documents mechanically.

- `docs/ERRATA.md` — 3,130 lines of numbered interpretations (I-1 … I-112),
  each quoting the printed sentence, on a named page, with the reading taken.
- `docs/COVERAGE.md` — every rule, with a status drawn from a closed
  six-word vocabulary (`covered`, `interpretation`, `accepted exception`,
  `out of scope`, `unreachable`, `play-time rule`) and the test or gate that
  holds it.
- `docs/POLICY.md` — one rule per choice point, versioned.

`audit/` is test-only, holds no rules, and is the machinery that keeps those
honest: every test COVERAGE.md cites exists; every ERRATA interpretation is
cited from code; every choice point has a POLICY rule; no chart field is
transcribed and then read by nothing; no prompt shows an identifier where the
chart prints a name; `character.schema.json` describes what the engine actually
writes; the compatibility corpus still replays; every ERRATA quotation is on
the page it names (`task citations`, which needs the private PDF and skips
rather than fails without it).

Understand `audit/` as a compiler for the documents and its size stops looking
disproportionate. Delete a gate and the corresponding document immediately
begins to rot, because nothing else can notice.

Three directories, three kinds of thing, and the split is principled: `docs/`
holds documents, `audit/` holds the code that checks them, the root holds what
convention puts there plus the two whole-system documents (`THEORY.md`,
`WALKTHROUGH.md`).

## 9. The seams, and which are principled

**CLI ↔ engine** (`cmd/t5chargen/main.go` ↔ `chargen.Generate`). Principled.
The engine is the single validator; the CLI's job is to decide whether a
returned error was the user's fault (`isUsageError` → exit 2) or the
operation's (exit 1). The comment on `isUsageError` names the one case that
does not fit cleanly — a `--current-year` the character has outlived is only
knowable once the age is, so it surfaces from the engine and is still the
caller's flag that was wrong.

**Engine ↔ front end** (`Decider`, `Watcher`). Principled, and the strongest
boundary in the system. `interactive/` is a line-based front end written so it
is fully testable with a scripted reader and no terminal; a fancier terminal UI
would be a view over it, not a replacement for it, because what the engine
consumes is indices into option lists.

**Record ↔ render.** Principled: `render` is a pure function of the record, and
that is what makes the sheet and the transcript trustworthy as evidence.

**Record ↔ replay.** Principled, and stated as a contract in the PRD.

**`chargen` internal structure.** Historical. Thirty-four non-test files in one
package, ~12,000 lines of Go (25,000 with its tests), with the term loop, the thirteen careers, education,
muster out, fame, aging, the policy and replay all sharing a package namespace.
Everything in it is unexported-to-the-world by Go's rules, so the package
boundary is doing real work; the _file_ boundaries within it are convention.
This is the place where the theory is thinnest — see §11.

**`interactive` ↔ `chargen` label coupling.** Nearly principled, and honestly
flagged in the code itself. The front end reads two `ScoreLabel` string
constants (`ScoreQualifies`, `ScoreAutomatic`) to decide how to render a score
as words rather than a digit. The doc comment on `boolean` names the cost — a
coupling to two strings — and names the trigger for revisiting it: a third such
label.

## 10. What the system is shaped to accommodate

Cheap, because the design anticipated them:

- **A new career.** Add the chart JSON, a constructor, a registry entry, a
  `careerMechanics` implementation for its peculiarities, COVERAGE rows, ERRATA
  entries for its ambiguities, POLICY rules for its choice points. Long, but
  entirely on rails.
- **A new choice point.** A `ChoiceID`, a POLICY rule (gated — a choice point
  without one fails the audit), and a case in the policy's `decide`.
- **A new consequence kind.** A constant, an emit site, a line in the
  transcript renderer.
- **A new interpretation.** An ERRATA entry; if it is a _Deviation_ rather than
  a reading, also a constant in `chargen/deviation.go`, membership in
  `Deviations`, a stamping site, and a case in `TestDeviationsAreStamped`. The
  gate holds the ERRATA headings and the Go constants equal in both directions.
- **A new front end.** Implement `Decider`; optionally `Watcher`.

Expensive, because it contradicts something structural:

- **Non-human characters.** `Characteristics` is six named fields and `UPP()`
  formats exactly six eHex digits. The PRD's non-goals exclude sophont variants
  precisely because they are not a table change.
- **An upgrade path for old records.** Deliberately absent, and the absence is
  the contract. Anyone who "fixes" replay to accept mismatched engines has not
  removed a limitation; they have removed the meaning of replay.
- **Concurrency, or generating several characters in one engine instance.** One
  seeded stream, threaded as a pointer, consumed in a strict order that is
  itself versioned. `batch` runs generations sequentially and derives per-record
  seeds; that is the supported shape.
- **Undo in interactive generation.** The log is append-only and the stream has
  been consumed. Abandoning is the only reverse gear, and by design it leaves
  no file at all.
- **Anything time-dependent.** No wall clock anywhere. `--current-year` is an
  input for exactly this reason.

Where a maintainer who _doesn't_ hold the theory does damage: reordering a
choice's options to be alphabetical; rewording a prompt to be clearer;
regenerating the compatibility corpus so the diff goes away; adding a rule
effect without an event; adding a convenience `rand` call; changing a chart
label because it reads oddly. Every one of those is a small, well-intentioned
change that silently invalidates records already in users' hands.

## 11. Uncertainties, and where I am inferring

Marked honestly, because the difference between a theory and a design document
is that a theory says which of its claims to trust.

**Inferred from code alone, not from a stated decision.**

- That the file split inside `chargen` is convention rather than structure. No
  document says so; I am reading it off the fact that the files import each
  other freely and share unexported helpers across boundaries. If there is an
  intended internal layering, it is not written down and is not enforced.
- That `careerrun.go`'s size is deliberate rather than deferred. The
  alternative reading — that it was going to be split and never was — is
  consistent with the evidence. The doc comments read as deliberate, and the
  shared-loop-plus-exceptions design is coherent, so I lean toward deliberate;
  I could be wrong.
- Not inferred, but worth flagging as unenforced: the `Watcher` seam's
  observation-only intent _is_ stated — "Watching is not deciding. A Watcher
  cannot change anything" (`chargen/event.go`, on the `Watcher` interface) —
  and nothing enforces it. A Decider that also watched could let observed
  events change its answers. It would be legal, deterministic, replayable, and
  squarely against the stated intent.

**Where the theory is thinnest.**

- The boundary between "a reading" and "a deviation" in ERRATA. 112
  interpretations, but only two are classified as Deviations and stamped into
  records. The classification is gated for consistency, but what makes an entry
  one rather than the other is a judgement the documents describe more than
  they define. A new ambiguity does not obviously sort itself.
- `docs/BETA_READINESS.md` is the plan of record for the phase this project is
  currently in, and it is the one document in `docs/` that has drifted: its
  numbered sections each open with a **Done** annotation and then continue with
  the original recommendation in unmarked present-tense imperative, so the same
  section both reports work finished and instructs that it be done. Filed as
  `beta-readiness-recommendations-contradict-their-own-done-annotations`.
- Nothing enforces that a _new_ rule effect emits an event. The gates check
  that documented rules have tests and that documented interpretations are
  cited; the "every effect emits an event" rule lives in `CLAUDE.md` and in
  review discipline. Replay would catch a _non-deterministic_ effect, but a
  deterministic effect with no event replays perfectly and is simply invisible
  in the transcript. This is the invariant with the widest gap between how
  load-bearing it is and how much machinery protects it.

**Where I looked and found no tension**, which is worth recording so the next
reader does not re-do it: the data/logic boundary holds (no conditionals in
`career/data`); `engine_version` bumps are gated; the compat corpus is
protected from `task goldens`; the rules collection `CLAUDE.md` names
(`~/Documents/Traveller/T5/`) exists; the milestone documents are all banner-
labelled as historical snapshots; the CLI's error paths are specific, correctly
coded (2 for usage, 1 for operational), and `--career` accepts lowercase names
despite the error message listing them capitalised. The repository is in
markedly better shape than a first review usually finds.

## Index

| #   | Severity | Issue                                                                  | Primary location                             |
| --- | -------- | ---------------------------------------------------------------------- | -------------------------------------------- |
| 1   | medium   | `beta-readiness-recommendations-contradict-their-own-done-annotations` | `docs/BETA_READINESS.md:56,68,76,86,103,125` |

**Total: 1 issue (0 critical, 0 high, 1 medium, 0 low)**
