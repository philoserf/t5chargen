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

It is held by a test rather than by this paragraph. Each corpus fixture
is the actual output of `go install ...@<its tag>`, and none is ever
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

## The command line

The record is the product, but a referee scripting `t5chargen` binds to
more than the record. This is what a script may rely on, and what it may
not.

### Held from the beta onward

- **Exit status.** 0 is success. 2 is the caller's fault: a flag the
  command cannot use, a career that does not exist or cannot open a
  lifepath, a homeworld that is not one, a current year the character
  cannot have been born before. 1 is an operation that ran and failed: a
  file that cannot be read or written, a record that does not replay, an
  abandoned session.
- **The subcommands**: `new`, `batch`, `render`, `replay`, `version`,
  `help`.
- **Streams.** Records and sheets go to stdout; prompts, progress and
  diagnostics go to stderr, so stdout is always parseable.
- **`-o` is checked before anything is generated.** An interactive run is
  never refused after its last question. `new -o` names one file: a
  directory, or a path ending in a separator, is refused. If a record
  cannot be written to `-o` after all, it goes to stdout, and the run
  exits 1.
- **`batch -o dir/`** writes one file per member, named
  `character-<seed>.json` for the seed that produced it; member _i_ is
  seed base+_i_. A path ending in a separator is a directory, created if
  missing; an existing directory is one too; anything else is a JSONL file.
- **JSONL** is one complete record per line. A JSONL file or a directory
  is written whole or not at all. JSONL on stdout is a stream: a batch
  that fails partway has already written the members before the failure,
  and the non-zero exit status is what says the stream is incomplete.
- Nothing is overwritten without `--force`.

### Exempt, permanently

- The character sheet's and the history transcript's layout and wording.
  They are derived from the record and can be derived again; the record
  is what is kept.
- Diagnostic text. Match the exit status, not the message.

### Still moving until 1.0

Flags may be added, renamed or removed; a release that does so says so in
`CHANGELOG.md`.

The three versions a record stamps move independently of the release tag
and of each other. `CHANGELOG.md` names the ones that moved.
