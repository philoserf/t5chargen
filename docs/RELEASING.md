# Releasing

The procedure this repository has run by hand. It exists so the next
release is the same release, and so that what a version number means is
written down rather than remembered.

## The three versions, and when each moves

They are constants in `chargen/character.go`, and every record stamps all
three. `t5chargen version` prints them.

| Constant        | Moves when                                                                        |
| --------------- | --------------------------------------------------------------------------------- |
| `SchemaVersion` | the shape of the records the engine writes changes                                |
| `EngineVersion` | generated output changes at all — a rule, the RNG, the stream's consumption order |
| `PolicyVersion` | the auto-mode decision table in [POLICY.md](POLICY.md) changes                    |

`SchemaVersion` tracks the records, not the document describing them. A
constraint that narrows the schema to what the engine already produced is
a clarification and does not bump; one that would invalidate a record the
current engine writes is a bump. Stamping a new field into records that
the schema already permitted is not a schema change — that case is
engine 0.45.0, which added the `errata` list without moving the schema.

`PolicyVersion` is stated in POLICY.md as well as declared in Go, and a
gate holds the two equal. Move both.

The release tag is separate from all three: it names a build, they name
contracts.

## Before tagging

Run in this order. Each step can invalidate the one before it.

1. **`task goldens`** if anything touched the engine. It rewrites the
   fixtures and then runs the gate. **Read the diff** — a fixture may
   only move when a change was meant to move it, and predicting what
   should move before running this is how a surprise gets noticed.
2. **`task`** — the gate: `go fix` cleanliness, `go mod tidy -diff`,
   prettier, vet, golangci-lint (formatting included), the tests under
   `-race`, and the coverage ratchet.
3. **`task citations`** — holds every ERRATA.md quotation to the page it
   cites, against Book 1 Print Edition 5.1. It needs the private PDF and
   skips without it, so a maintainer with the PDF must run it.
4. **The smoke test** — the five workflows, against a built binary rather
   than `go run`:

   ```sh
   go build -o /tmp/t5chargen ./cmd/t5chargen
   /tmp/t5chargen new --auto --seed 7 -o auto.json     # automatic
   yes 1 | /tmp/t5chargen new --seed 7 -o inter.json   # interactive
   /tmp/t5chargen batch --count 5 --auto -o batch.jsonl
   /tmp/t5chargen render auto.json && /tmp/t5chargen render --history auto.json
   /tmp/t5chargen replay auto.json && /tmp/t5chargen replay inter.json
   ```

   The interactive record is the one worth checking: it carries
   `policy_version: "none"`, and replaying it is the only way back to the
   run it describes.

5. **Read the documents that make claims about completeness** — README,
   [PRD.md](PRD.md), [COVERAGE.md](COVERAGE.md),
   [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md). The gates hold the
   tables and the statuses; prose is reviewed by a person, which is
   recorded as a deliberate limit in the `audit` package doc.

## Tagging

Tag the exact commit that was verified, on `main` after the pull request
merged — not a branch head, and not a commit with anything unpushed.

```sh
git checkout main && git pull
version=v0.1.0-alpha.1   # the tag being released
git tag -a "$version" -m "$version"
git push origin "$version"
```

There is no release workflow (removed 2026-10-12). Pushing the tag makes
the version installable through the module proxy; any binaries and the
GitHub release are made by hand.

Prereleases take `-alpha.N` / `-beta.N`, which keeps `@latest` resolving
past them once a stable version exists.

## After tagging

Write the release notes. They say what it is, what works, what the known
limitations are, and that the rules belong to Far Future Enterprises —
which is a paragraph a person writes.

Check the tag reaches the binary. A `go build` in the work tree reports a
VCS pseudo-version; only a module install proves it:

```sh
GOPATH=$(mktemp -d) go install "github.com/philoserf/t5chargen/cmd/t5chargen@$version"
t5chargen version   # must report the tag, not (devel)
```

Then update [RELEASE_READINESS.md](RELEASE_READINESS.md) with the tag it
names, and add the release to the compatibility corpus — written by the
released binary, never by the work tree:

```sh
corpus=audit/testdata/corpus
t5chargen new --auto --seed 7 -o "$corpus/${version}_auto.json"
t5chargen new --auto --seed 26 --career scout -o "$corpus/${version}_scout.json"
```

and name the tag in `TestTheCorpusHoldsEveryReleasedVersion`'s list. That
list is written out rather than read from git, so nothing notices a
release missing from it; v0.1.0-alpha.2 went five weeks without its
records.

## What is not automated, and why

No Homebrew formula. It is a maintenance commitment better made once
there are users asking for it, and `go install` plus the attached
binaries covers everyone until then.

The release notes are not generated. What the release _is_ stays a paragraph a
person writes.

`task citations` is outside the gate. It holds ERRATA.md's
quotations to the pages they cite against Book 1, Print Edition 5.1 — a
purchased artifact this repository does not redistribute and cannot. It
skips wherever `T5_RULES_PDF` is unset, so a pass without the PDF is a
pass it never performed. Run it
locally before tagging; it is step 3 above.
