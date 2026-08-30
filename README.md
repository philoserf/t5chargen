# t5chargen

Generate Traveller5 characters from the command line — the full human
lifepath, rolled by the book, with every throw and choice recorded so the
result can be audited and reproduced exactly.

```
$ t5chargen new --auto --seed 7
```

```text
# Character Card

**Name**:

**UPP**: 764777

**Homeworld**: Regina A788899-C (Ph Pa Ri)

**Age**: 34 (Peak)

**Born**: Forday 229-1071

| Str | Dex | End | Int | Edu | Soc |
| --- | --- | --- | --- | --- | --- |
| 7 | 6 | 4 | 7 | 7 | 7 |

**Education**: University, Major Athlete, Minor Broker — did not graduate

**Career**: Citizen (3 terms), Job Seafarer, Hobby ACV

**Skills**: ACV-2, Actor-1, Admin-1, Animals-1, Aquanautics-2, Athlete-1, Broker-2, Bureaucrat-2, Career: Citizen-3, Computer-1, Rider-2, Seafarer-3, Trader-4, World: Regina-5

**Credits**: Cr45000

**Automatics**: Fame

**Citizen's Pension**: Cr5000 a year from age 66

**Status**: Fame 2 (Close Family)

---

Seed 7 (math/rand/v2-pcg) · schema 0.33.0 · engine 0.45.0 · policy 0.25.0

Ruleset: Traveller5 Core Rules Book 1, Print Edition 5.1
```

Rules baseline: **Traveller5 Core Rules Book 1, Print Edition 5.1**.
Every rule carries a page citation.

## Install

```sh
go install github.com/philoserf/t5chargen/cmd/t5chargen@v0.1.0-alpha.1
```

Go 1.27 or later. No other dependencies — the tool is standard library
only.

Supported on macOS and Linux, which CI builds and tests on both the Go
version `go.mod` declares and the current release. Windows is not
supported: the tool is pure Go and may well run there, but nothing tests
it, and an untested platform is not a supported one.

The version is named explicitly rather than `@latest`, which stops
resolving to a prerelease once a stable version exists.

## Stability

Prerelease. Flags and output may still change, and
[CHANGELOG.md](CHANGELOG.md) says what moved between releases.

The record format is the part that is held still. It is versioned, its
schema is published (`docs/character.schema.json`), and a record written
by a released version renders under later released versions. Replay stays
pinned to the engine that wrote the record — that is the provenance
contract, not a limitation to route around, and `--ignore-provenance` is
there when you mean to.

## Report a problem

<https://github.com/philoserf/t5chargen/issues>

A rules report is worth far more with the record attached: it carries the
seed and every choice, so it reproduces exactly. `t5chargen help` says
what else to include, and the issue templates ask for it.

## Use

```sh
t5chargen new --auto --seed 7              # the policy answers every choice
t5chargen new                              # you answer every choice
t5chargen new --auto --seed 7 -o char.json # write the record
t5chargen render char.json                 # the character sheet, again
t5chargen render --history char.json       # every throw and choice, in order
t5chargen replay char.json                 # check the record reproduces itself
t5chargen batch --count 20 --auto -o npcs/ # twenty NPCs, one file each
t5chargen version                          # the build, and what a record stamps
```

`new` writes JSON to stdout unless given `-o`. Omitting `--seed` draws one
from OS entropy and records it, so any character can be regenerated
exactly.

Useful flags: `--career scout` forces the first career, `--homeworld
"A788899-C Ph Pa Ri"` supplies a world (`random` rolls one on chart B;
the default is Regina), `--name` names the character, `--current-year`
sets the Imperial year adventuring begins in (default 1105), and
`--force` allows overwriting an existing file.

## What a record guarantees

The JSON record is the source of truth; the character sheet is a render
of it. Every record carries an event log of the entire generation — each
step entered, each throw with its dice, target, modifiers and page
citation, each choice with who decided and what the alternatives were,
and each consequence tied to the throw or choice that caused it.

That log makes two things possible.

**Audit.** `render --history` prints the lifepath as a transcript, so any
character can be checked against Book 1 by reading it.

```
- #2 2D = 6+1 = 7 — Book 1 p. 56 chart A
  - #3 (from #2) Str = 7
```

**Replay.** `replay` re-runs the engine from the recorded seed, inputs and
choices, recomputing every throw and comparing the result. It exits
non-zero at the first event that disagrees.

```
$ t5chargen replay char.json
replayed char.json: 151 events reproduced from seed 7
```

Records also carry the schema, engine and policy versions, the RNG
algorithm and seed, the pinned ruleset, and any ERRATA deviations applied
— so a character stays auditable after the tables underneath it change.
The record's shape is [docs/character.schema.json](docs/character.schema.json).

## What it covers

All thirteen careers end to end — Citizen, Scholar, Entertainer, Scout,
Merchant, Noble, Agent, Rogue, Craftsman, Functionary, and the Armed
Forces (Spacer, Soldier, Marine) — with per-term skills, Risk & Reward,
injury and death, rank, commission and promotion, and each career's own
mechanics: the Entertainer's Fame and Talent, the Scholar's publications
and tenure, the Noble's exile and elevation, the Rogue's schemes, prison
and payoffs, the Craftsman's masterpiece, the Agent's cover careers and
commendations.

Around them: characteristics, homeworld skills from UWP trade
classifications, chart C's education from ED5 up through Masters,
Professors, Medical and Law School with OTC and NOTC beside them, aging,
career changes, Fame, muster out, and a birthdate.

Skills follow p. 134's Knowledge-Knowledge-Skill sequence: the first two
receipts of a container skill award one of its Knowledges instead, so a
character receiving Fighter five times leaves with Fighter-3 and a
Knowledge-2. Careers and worlds leave Knowledges of their own —
`Career: Scout`, `World: Regina` — each capped at 6.

**Read [docs/KNOWN_LIMITATIONS.md](docs/KNOWN_LIMITATIONS.md) before
filing a bug.** One rule is deliberately not implemented, two depart from
the printed page and say so in the record, and the auto policy declines
several rules a player can take.

## Documentation

|                                                        |                                                        |
| ------------------------------------------------------ | ------------------------------------------------------ |
| [docs/PRD.md](docs/PRD.md)                             | what v1 is, stated once                                |
| [docs/COVERAGE.md](docs/COVERAGE.md)                   | every rule, its page, its implementation, its test     |
| [docs/ERRATA.md](docs/ERRATA.md)                       | every interpretation and deviation, with the reasoning |
| [docs/POLICY.md](docs/POLICY.md)                       | what `--auto` decides, choice by choice                |
| [docs/KNOWN_LIMITATIONS.md](docs/KNOWN_LIMITATIONS.md) | what it does not do                                    |
| [docs/RELEASE_READINESS.md](docs/RELEASE_READINESS.md) | what was verified before the tag                       |
| [docs/RELEASING.md](docs/RELEASING.md)                 | the release procedure                                  |

## Traveller5

This is an unofficial fan-made tool. It is not affiliated with, endorsed
by, or supported by Far Future Enterprises. _Traveller_ is a registered
trademark of Far Future Enterprises, and the Traveller5 rules — including
the charts and tables this tool implements — are their copyright. The
rulebooks are not redistributed here; you need your own copy of Book 1 to
check a citation, and to play.

The MIT licence below covers this repository's own code.

## Development

```sh
task deps   # install the toolchain (Homebrew)
task        # the gate: check + test
task hooks  # run the gate on every push
```

`task` is what CI runs, so green means the same thing on both.
Contributions should keep it green and leave the golden fixtures moved
only deliberately — `task goldens` regenerates them.

MIT licensed. See [LICENSE](LICENSE).
