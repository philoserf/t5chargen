# t5chargen

A Traveller5 (T5) character generator CLI in Go, targeting the full v5.10 human
lifepath — characteristics, homeworld, education, all 13 careers, aging, muster
out — with deterministic replay and a complete generation event log.

Go module: `github.com/philoserf/t5chargen`. Standard library only.

## Status

The spec is [docs/PRD.md](docs/PRD.md). Milestones 1 (foundations) and 2
(pre-career: homeworld skills, education) are complete; milestone 3 (careers)
is in progress with 2 of 13 careers playable.

What works today:

- Characteristic generation, UPP, homeworld skills from UWP trade
  classifications, and pre-career education (ED5, Trade School,
  Apprenticeship, College, University, Service Academy).
- The **Citizen** and **Scout** careers end-to-end: controlling
  characteristics, per-term skills, Risk & Reward, injury and death,
  continuation checks.
- A deterministic engine: one seeded RNG stream, every choice through the
  `Decider` interface, a fixed auto-mode policy ([POLICY.md](POLICY.md)), and
  an event log recording every throw, choice, and consequence with page
  citations.
- A JSON character record carrying full provenance (seed, RNG algorithm,
  schema/engine/policy versions, ruleset), rendered as a Markdown character
  sheet or a generation-record transcript.

Not yet implemented: the other 11 careers, aging, career changes, muster out
and benefits, fame processing, interactive mode, batch generation, and the
replay verifier. [COVERAGE.md](COVERAGE.md) maps every implemented rule to its
page cite and lists the deferrals; [ERRATA.md](ERRATA.md) records deliberate
interpretations where the printed rules are ambiguous.

## Usage

```sh
t5chargen new --auto --seed 42 -o char.json   # generate a character record
t5chargen render char.json                    # Markdown character sheet
t5chargen render char.json --history          # generation-record transcript
```

`new` requires `--auto` (interactive mode is planned). Optional flags:
`--name`, `--career citizen|scout`, `--homeworld "UWP TC TC..."` (for example
`"A788899-C Ph Pa Ri"`; defaults to Regina), `--force`. Omitting `--seed`
draws one from OS entropy; the seed is recorded, so any record can be
regenerated exactly.

## Relationship to sibling repos

Chargen implementations also exist in [`philoserf/traveller`](https://github.com/philoserf/traveller)
(rules library) and [`philoserf/t5`](https://github.com/philoserf/t5) (toolkit). This repo is a
deliberate fresh start focused on the PRD's distinct goals: an auditable generation event log,
a replay/provenance contract, and a `Decider` interface separating rules from choice policy.
It shares no code with either.

## Rules source

Book 1 of the Traveller5 Core Rules, Print Edition 5.1 (the v5.10 three-book split), held in
the private Traveller collection at `~/Documents/Traveller/T5/`. Every rule carries a page
citation verifiable against that artifact. PDFs are not redistributed here.

## Development

```sh
task deps   # install toolchain (Homebrew)
task        # check (modernize + gofumpt + vet + lint) + test
task hooks  # install the pre-push gate
```

MIT licensed.
