# Changelog

What changed between releases, for someone deciding whether to upgrade.

The three versions a record stamps — schema, engine and policy — move
independently of the release tag and of each other, so each entry names
the ones that moved. A record carries all three, which is what makes a
bug report from an old binary still answerable.

This file starts at the first release. Everything before it is in git.

## Unreleased

Nothing released yet since `v0.1.0-alpha.1`. Landed on `main`:

- Beta preparation: a smoke matrix over macOS and Linux against both the
  Go version `go.mod` declares and the current release; Windows declared
  unsupported; fuzz targets over UWP parsing, the eHex digits, render and
  replay; a release workflow that builds from a pushed tag and drafts the
  release; `t5chargen help`; issue templates; this file.
- Go 1.27 is the declared floor.
- Test-suite repairs: seven tests that declined to run now assert or
  fail, with a gate that keeps it that way, and one rule — chart 02's
  Major-or-Minor cell — that had no test at all now has one.

No schema, engine or policy version changed.

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
