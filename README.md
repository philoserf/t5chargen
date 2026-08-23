# t5chargen

A Traveller5 (T5) character generator CLI in Go, targeting the full v5.10 human
lifepath — characteristics, homeworld, education, all 13 careers, aging, muster
out — with deterministic replay and a complete generation event log.

Go module: `github.com/philoserf/t5chargen`. Standard library only.

## Status

The spec is [docs/PRD.md](docs/PRD.md). Milestones 1 (foundations), 2
(pre-career: homeworld skills, education), 3 (careers) and 4 (the rest of
the lifepath) are complete. A character runs end to end: characteristics,
homeworld, education, all thirteen careers with aging and career changes,
Fame, muster out, and a birthdate.

What works today:

- Characteristic generation, UPP, homeworld skills from UWP trade
  classifications, and pre-career education (ED5, Trade School,
  Apprenticeship, College, University, Service Academy).
- The **Citizen**, **Scholar**, **Entertainer**, **Scout**, **Merchant**,
  **Noble**, **Agent**, **Rogue**, and the Armed Forces (**Spacer**,
  **Soldier**, **Marine**) careers end-to-end: controlling characteristics,
  per-term skills, Risk & Reward, injury and death, rank with commission,
  promotion and tenure, the Entertainer's Fame and Talent, the Scholar's
  publications and waivers, the Noble's exile and elevation, the Rogue's
  schemes, prison, and payoffs, continuation checks.
- A deterministic engine: one seeded RNG stream, every choice through the
  `Decider` interface, a fixed auto-mode policy ([POLICY.md](POLICY.md)), and
  an event log recording every throw, choice, and consequence with page
  citations.
- A JSON character record carrying full provenance (seed, RNG algorithm,
  schema/engine/policy versions, ruleset), rendered as a Markdown character
  sheet or a generation-record transcript.

Not yet implemented: interactive mode, batch generation, the replay
verifier, and the formal JSON Schema — all milestone 5.

An earlier draft of this section listed FR7's Land Grant and Ship Share
values and FR8's birthdate as deferred against the spec, on the strength of
a rulebook sweep that missed the pages. Both are implemented now, and
[docs/MILESTONE-4.md](docs/MILESTONE-4.md) records what the sweep got
wrong.

[COVERAGE.md](COVERAGE.md) maps every implemented rule to its page cite and
lists the deferrals; [ERRATA.md](ERRATA.md) records deliberate
interpretations where the printed rules are ambiguous.

## Usage

```sh
t5chargen new --auto --seed 42 -o char.json   # generate a character record
t5chargen render char.json                    # Markdown character sheet
t5chargen render char.json --history          # generation-record transcript
```

`new` requires `--auto` (interactive mode is planned). Optional flags:
`--name`, `--career citizen|scholar|entertainer|scout|merchant|spacer|soldier|noble|marine|agent|rogue`, `--homeworld "UWP TC TC..."` (for example
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
