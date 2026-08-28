# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A T5 v5.10 character generator CLI. `docs/PRD.md` is the spec — read it before implementing
anything; it fixes scope (human core lifepath only), the replay/provenance contract, JSON
conventions, the auto-policy requirements, and milestones.

## Ground rules

- **Rules authority**: Traveller5 Core Rules Book 1, Print Edition 5.1, at
  `~/Documents/Traveller/T5/`. Implement from the cited page text, never from memory or from
  the 2008-preliminary extracts in that collection's `Archive/` (locate topics there, verify
  in Book 1). Quote the governing rule in doc comments at the implementation site.
- **Deviations**: never silently deviate from the printed rule; record deliberate deviations
  in `ERRATA.md` with the page cite and rationale.
- **Data/logic boundary**: tables, thresholds, and labels are embedded data files;
  orchestration and career-specific mechanics are typed Go. No rules language.
- **Determinism**: no wall-clock time or unseeded randomness in the engine. All rolls come
  from the seeded stream (Go `math/rand/v2` PCG); every choice goes through the `Decider`
  interface. Changing the RNG or default policy is a version bump.
- **Event log first**: every throw, choice, and consequence emits an event (see PRD FR10).
  New mechanics are not done until their events render in the history transcript and replay
  verifies them.
- **Values outside the rules' range**: one answer, not one per site. A value a _caller_
  supplies — a day, a dice count, an eHex value — is refused with an error. A value the
  _engine derives_ clamps to the rule's own floor and emits a consequence saying so: the
  record is the product, and a clamp nobody can see is worse than one they can. Nothing
  exported panics; where a precondition genuinely holds, state it in the doc comment
  instead of leaving the caller to know. Substituting a symbol the format lacks is never
  the answer.
- Sibling repos `philoserf/traveller` and `philoserf/t5` contain independent chargen
  implementations. Do not import from or copy them — this repo is a deliberate clean-room
  restart; consult them only when explicitly asked.

## Commands

```sh
task          # check + test (the gate; also runs on pre-push via task hooks)
task fmt      # gofumpt for Go, prettier for JSON and Markdown
task test     # go test -race ./...
```

`main` is protected on GitHub: pushes to it are rejected, history is
linear, force-pushes and deletion are blocked, and the CI check
`task (check + test)` must pass before a pull request can merge. Every
change lands through a pull request — branch, run the gate, push the
branch, then `gh pr merge --squash --delete-branch`. No review approval
is required, so this costs a solo workflow nothing beyond remembering to
branch first. The rules apply to the repository owner too, which is the
point: they exist to catch an accidental commit on `main`, and an owner
exemption would defeat that. Lift them deliberately (repository
settings, or the branches protection API) if you ever genuinely need to.

The required check is "strict": a branch must be up to date with `main`
before it merges. That is what makes green mean the code as it will
exist on `main` rather than as it existed when the branch left it, and
the cost is updating a branch that has fallen behind. `task` and CI run
the same thing, so a branch that passes locally and is current passes
there.

Prettier formats the embedded chart data and the documents, but never
`chargen/testdata` or `render/testdata`: those are the engine's own output,
compared byte for byte, and a formatter must not be the thing that decides
what a character record looks like. Prose that quotes the printed rules
must escape T5's literal asterisks (`"+F +F\* +F\*"`), or prettier reads
them as emphasis and rewrites the quote.

Never edit a fixture by hand. Regenerate them with `task goldens`, which
rewrites the fixtures and then runs the full gate. A fixture is only
allowed to move when a change was meant to move it, so read the diff
before committing it.

## Layout

- `cmd/t5chargen` — CLI (subcommands: new, batch, render, replay).
- `dice` — dice engine: xD, Flux, target-number throws (PRD FR9).
- `chargen` — engine; consumes a `Decider` for all choice points.
- `career` — data-driven career definitions.
- `render` — character sheet and history transcript output.

The rest are one embedded chart or vocabulary each, loaded through the same
`go:embed` plus `sync.OnceValues` pattern with load-time validation:
`benefit` (chart M1), `calendar` (the Imperial Calendar and Birth Date
Generation, pp. 262-263), `career` (charts 01-13), `education` (chart C),
`ehex` (the extended hex digits), `fame` (chart F), `lifestage` (chart A's
stages), `medal`, `ship` (chart S), `skill` (chart MS) and `world`
(chart B). `interactive` is the line-based front end for interactive
generation.

`audit` is test-only and holds no rules: it is the guards that keep the
documents honest — that every test COVERAGE.md cites exists, that every
ERRATA.md interpretation is cited, that every choice point has a POLICY.md
rule, that no chart field is transcribed and then read by nothing, that no
prompt shows a player an identifier where the chart prints a name, and that
character.schema.json describes what the engine actually writes.

Three folders, three kinds of thing. `docs` holds documents and nothing
else: the spec, the living COVERAGE/ERRATA/POLICY, the milestone histories
and the JSON Schema with its two examples. `audit` holds the code that
checks them. The root holds only what convention puts there — README,
LICENSE, this file, and the build configuration.
