# Changelog

What changed between releases, for someone deciding whether to upgrade.

The three versions a record stamps — schema, engine and policy — move
independently of the release tag and of each other, so each entry names
the ones that moved. A record carries all three, which is what makes a
bug report from an old binary still answerable.

This file starts at the first release. Everything before it is in git.

## Unreleased

Nothing since `v0.1.0-beta.1`.

## v0.1.0-beta.1 — 2026-09-23

schema 0.33.0 · engine 0.45.0 · policy 0.25.0 — **all three unchanged.**

The first beta, for testers. Nothing about a generated character moved,
and records written by every earlier release still replay under this one.
The beta is where the project meets its users: what it asks of testers,
and what ends it, is in
[docs/RELEASE_READINESS.md](docs/RELEASE_READINESS.md).

- The history transcript wraps a long choice's options beneath its line
  instead of printing them inline, so chart B's worlds and Citizen's Job
  and Hobby lists no longer make lines of over a thousand characters.
  Every option is still shown.
- The engine refuses a build whose embedded chart data failed to load
  before it rolls any dice, rather than at whichever step first reads it.
- Every record's characteristics, skills and age are now held to its own
  event log by a test, so a rule effect that skips the log fails the gate.
- Records are checked against the published JSON Schema by an imported
  validator in the tests; the shipped binary is still standard library
  only.

## v0.1.0-alpha.3 — 2026-09-23

schema 0.33.0 · engine 0.45.0 · policy 0.25.0 — **all three unchanged.**

The build handed to testers. Nothing about a generated character moved:
a record written by an earlier release still replays under this one.

- `new` checks `-o` before the first question, and a record `-o` cannot
  take after all goes to stdout instead, so neither loses a player's
  answers. `new -o` names one file; a directory is refused by name.
- `batch` resolves `-o` before it generates anything and writes one
  member at a time: `--count 20000` went from 4.6 GB of memory to 17 MB,
  with byte-identical output. A JSONL file or a directory is still
  written whole or not at all. JSONL on stdout now streams, so a batch
  that fails partway has already written the members before the failure;
  the non-zero exit status says the stream is incomplete.
- [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md) says what the command
  line promises a script: exit statuses, which stream carries what, the
  rules for `-o`, and batch's file names.
- `t5chargen help` says what `--auto` produces — every automatic
  character is a Citizen — and how to force another career. A
  subcommand's `--help` goes to stdout and exits 0.
- An option number the list does not hold is refused instead of being
  searched for, and an accidental paste no longer ends a session.
- `nilaway` runs in the gate. Its eleven findings were guards it could
  not see, not nils a run could reach; each is now visible to it.

## v0.1.0-alpha.2 — 2026-08-30

schema 0.33.0 · engine 0.45.0 · policy 0.25.0 — **all three unchanged.**

Nothing about a generated character moved. A record written by
`v0.1.0-alpha.1` replays under this release exactly, which is the
strongest thing this entry says: everything below is the scaffolding
around the engine rather than the engine.

- Beta preparation: a smoke matrix over macOS and Linux against both the
  Go version `go.mod` declares and the current release; Windows declared
  unsupported; fuzz targets over UWP parsing, the eHex digits, render and
  replay; a release workflow that builds from a pushed tag and drafts the
  release; `t5chargen help`; issue templates; this file.
- [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md): a record written by a
  released version renders under every later released version, and replay
  stays pinned to the engine that wrote it. Held by a corpus of records
  written by each released binary, not by the paragraph saying so.
- Go 1.27 is the declared floor.
- Test-suite repairs: seven tests that declined to run now assert or
  fail, with a gate that keeps it that way, and one rule — chart 02's
  Major-or-Minor cell — that had no test at all now has one. COVERAGE.md
  had called both covered.
- `LICENSE` is verbatim MIT again; the Far Future Enterprises attribution
  it also carried is in [README](README.md), where it always was too.

This is the first release built by
[.github/workflows/release.yml](.github/workflows/release.yml), so it is
also the first with attached binaries and checksums for macOS and Linux.
`go install` remains the route the README names.

## v0.1.0-alpha.1 — 2026-08-28

schema 0.33.0 · engine 0.45.0 · policy 0.25.0

The first release. All seven PRD milestones closed: the human core
lifepath, every one of the thirteen careers, education, muster out,
aging, career changes and the fame system, with 111 recorded
interpretations where the printed rules were ambiguous.

Records are versioned and replayable, and the character sheet and history
transcript render from the record alone.

Known limitations are in [docs/KNOWN_LIMITATIONS.md](docs/KNOWN_LIMITATIONS.md).
Chart 11's `Capital***` cell is the one rule in v1 scope that is
deliberately incomplete.
