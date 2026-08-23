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
linear, and force-pushes and deletion are blocked. Every change lands
through a pull request — branch, run the gate, push the branch, then
`gh pr merge --squash --delete-branch`. No review approval is required, so
this costs a solo workflow nothing beyond remembering to branch first. The
rules apply to the repository owner too, which is the point: they exist to
catch an accidental commit on `main`, and an owner exemption would defeat
that. Lift them deliberately (repository settings, or the branches
protection API) if you ever genuinely need to.

Prettier formats the embedded chart data and the documents, but never
`chargen/testdata` or `render/testdata`: those are the engine's own output,
compared byte for byte, and a formatter must not be the thing that decides
what a character record looks like. Regenerate those fixtures with `task
goldens` (which rewrites them and then runs the full gate) rather than by
hand, and never edit one directly — a fixture is only allowed to move when
a change was meant to move it, so read the diff before committing it. Prose that quotes the printed rules
must escape T5's literal asterisks (`"+F +F\* +F\*"`), or prettier reads
them as emphasis and rewrites the quote.

## Layout

- `cmd/t5chargen` — CLI (subcommands: new, batch, render, replay).
- `dice` — dice engine: xD, Flux, target-number throws (PRD FR9).
- `chargen` — engine; consumes a `Decider` for all choice points.
- `career` — data-driven career definitions.
- `render` — character sheet and history transcript output.
