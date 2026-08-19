# t5chargen

A Traveller5 (T5) character generator CLI in Go: the full v5.10 human lifepath —
characteristics, homeworld, education, all 13 careers, aging, muster out — with
deterministic replay and a complete generation event log.

Go module: `github.com/philoserf/t5chargen`. Standard library only.

**Status: pre-implementation.** The spec is [docs/PRD.md](docs/PRD.md); nothing is built yet.

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
