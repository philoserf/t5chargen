# Changelog

What changed between releases, for someone deciding whether to upgrade.

The three versions a record stamps — schema, engine and policy — move
independently of the release tag and of each other, so each entry names
the ones that moved. A record carries all three, which is what makes a
bug report from an old binary still answerable.

This file starts at the first release. Everything before it is in git.

## Unreleased

Nothing since `v0.1.0-alpha.2`.

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
