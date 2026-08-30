# Compatibility

What a released version promises about records written by another one.

`docs/BETA_READINESS.md` §2 asks for this before another engine release,
on the grounds that the project versions records carefully and then says
nothing about what the versions entitle you to. This is that statement.

Two promises, and the corpus that holds them: `audit/compat_test.go` over
`audit/testdata/corpus`, one record per released version, written by that
version's own binary.

## Render forward

**A record written by a released version renders under every later
released version.** `render`, with or without `--history`, reads any
record this project has published.

This is the promise that matters to someone keeping characters. A record
is the character; a version that could not read last month's records would
make the format's careful versioning pointless.

It is held by a test rather than by this paragraph. The corpus fixtures
are the actual output of `go install ...@v0.1.0-alpha.1` and are never
regenerated — `task goldens` rewrites `./chargen` and `./render`, and
deliberately not that directory. A corpus a later engine can rewrite
proves nothing about what an earlier engine wrote.

## Replay stays pinned

**Replay requires the engine version that wrote the record.** A record
from another engine is refused, and the refusal names the version it
wants.

This is not a limitation to route around; it is what replay means. Replay
re-runs the engine from the recorded seed and choices and compares the
result byte for byte. A different engine legitimately produces a different
character — that is what an engine version change _is_ — so replaying
across versions could only ever report a difference that is not a defect.

`--ignore-provenance` runs it anyway and says so, for when you mean to.

## Getting an older executable

Every released version stays installable from the module proxy:

```sh
go install github.com/philoserf/t5chargen/cmd/t5chargen@v0.1.0-alpha.1
```

That is the supported route to replaying an old record, and it needs
nothing from this repository: the proxy keeps published module versions
independently of GitHub. Deleting a tag here does not withdraw a released
version, and neither does deleting a release.

Attached binaries on a GitHub release are a convenience with no retention
promise. `go install` is the route that keeps working.

## What breaks a schema version

`schema_version` tracks the shape of the records the engine writes, not
the precision of the document describing it (`docs/PRD.md`).

**A bump** — a change that would make the current engine's own output
invalid against the previous schema, or a record valid under the previous
schema unreadable now. Removing a field, renaming one, narrowing a type,
or adding a required field.

**Not a bump** — a constraint that narrows the schema to what the engine
already produced, an added optional field, or a documentation fix. The
case that settled it was the `upp` pattern: added after the clamp that
made an unrepresentable UPP impossible, so every record already written
was inside it.

Records carry their version, so the question is always about a record in
hand rather than about the file in this repository.

## During beta

Flags, output text and the character sheet's layout may change; they are
not covered by either promise above. The record format is the part held
still.

The three versions a record stamps move independently of the release tag
and of each other. `CHANGELOG.md` names the ones that moved.
