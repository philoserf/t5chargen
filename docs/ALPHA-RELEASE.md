# The alpha release — what the run found

> **Historical snapshot.** This records the run that produced
> `v0.1.0-alpha.1` on 2026-08-28. See [COVERAGE.md](COVERAGE.md) for
> current coverage and [RELEASE_READINESS.md](RELEASE_READINESS.md) for
> what was verified.

Sixteen pull requests in one afternoon — 15:45 to 19:56 on 2026-08-28 —
from an external prerelease review to a tag. Not a milestone: the PRD's
seven were already complete and nothing here added a rule the book
prints. What it added was the ability to tell whether the documents were
lying.

Cites are to Book 1, Print Edition 5.1.

## What shipped

|                            |                                                                              |
| -------------------------- | ---------------------------------------------------------------------------- |
| The mechanical findings    | #112 — eight direct contradictions the review found                          |
| I-47                       | #113 — an open question the code had silently answered, charged per stint    |
| The citation sweep         | #114, #115 — thirteen citations wrong or missing, then gated against the PDF |
| The smoke test             | #116 — the five workflows, recorded                                          |
| Histories labelled as such | #117 — banners on four milestone documents, and one overstatement            |
| The status vocabulary      | #118 — six words for COVERAGE.md's Status column, and two gates              |
| The ERRATA classification  | #119 — four kinds, and a gate on what each kind owes                         |
| Deviations stamped         | #120 — the PRD's provenance contract, kept for the first time                |
| The PRD frozen             | #121 — twelve dated amendments folded into what they amended                 |
| CI                         | #122 — the gate, run on GitHub                                               |
| `t5chargen version`        | #123 — read from the build rather than stamped on it                         |
| The README                 | #124 — a product page, and KNOWN_LIMITATIONS.md beside it                    |
| Release mechanics          | #125 — RELEASING.md, and the review retired into a readiness record          |
| The tag, and recording it  | #126 — `v0.1.0-alpha.1`                                                      |
| The check required         | #127 — CI made a gate rather than a report                                   |

Ten new gates in `audit`, taking it to thirty-three.

## The defect this run was actually about

Not a bug in the engine. A **claim invalidated by the commit that
invalidates it** — a sentence that was true when written, made false by
the change landing in the same breath, and visible to nobody.

The chain is the evidence, because each link was caught by the next
piece of work rather than by the one that broke it:

- #118 gated COVERAGE's Status column. The `audit` package doc, four
  files away, still described that column as "half gated". Caught in
  #119.
- #119 classified two ERRATA entries as Deviations. The PRD requires
  every record to carry "any applied `ERRATA.md` deviations", and
  `Character.Errata` had never been written. The preamble asserted the
  stamping as though it happened. Caught by writing the classification
  down.
- #120 stamped them. That falsified the preamble's "neither is stamped",
  its count of machine-checked obligations, README's "two things the PRD
  asks for are not done", and the schema's word "interpretations".
  Swept in the same commit — except the last, which the schema gate
  caught itself: `errata` sat on its unexercised list with the reason
  "no rule records one against a character yet".
- #127 required the CI check. `CLAUDE.md` described the branch
  protection as three rules. Caught in the same session, by looking.

Four of those five were caught by a gate or by a person reading. The
fifth was caught because a gate had been written the week before for a
different reason. That is the argument for gates over vigilance: the
vigilance worked here and it worked because someone was already looking.

## Gates that pass on the first run have not been shown to work

Every gate in this run was written **before** the thing it checks was
corrected, and mutation-tested after. Twice that discipline paid
directly.

**#118 was predicted to name eight rows. It named thirty-three.** The
prediction counted rows whose status _word_ was unusual. The gate also
checks whether a row claiming implementation names evidence, so it found
twenty-four vocabulary strays and nine rows asserting work with nothing
behind them.

Reading those nine found something no one was looking for: **three rows
in the chart 11 table carried seven cells against a five-column header.**
Markdown drops the surplus, so `nobleFame`, `MusterOutRow.Power`,
`no_career_change` and the three tests they name rendered as though the
rows named nothing — while the gate, reading by position, reported them
untested. The document was wrong in both directions at once and neither
was visible on the page. That produced a gate nobody had planned,
`TestEveryCoverageRowFitsItsHeader`.

**#119's evidence gate found I-4 and I-37 cited only from COVERAGE's
prose.** The older citation gate searched the whole document and passed;
no table row named a test for either.

## Every prediction that was wrong was worth more than the ones that were right

The house rule is to write down what should move before running
anything. Three predictions failed, and each failure taught something
the successful ones did not.

- **The Scout fixture holds five Land Grants and was predicted to carry
  the I-82 stamp.** It does not: he dies before mustering out (I-77), so
  nothing ever priced them. The prediction read `land_grants > 0` and
  never asked whether the code that applies the deviation had run.
- **All eighteen render fixtures were predicted to move.** Thirteen did
  — the sheets, which carry the provenance line. The five history
  transcripts do not carry it.
- **A local build was predicted to report `(devel)`.** Installing from
  the module proxy and reading it back with `go version -m` showed three
  cases, not two: a `go build` inside the tree reports a VCS
  pseudo-version with `+dirty` appended. That is strictly better than
  what was planned for — a bug report from a working copy names its
  commit and says whether it was clean — and it would have been
  documented wrongly.

## A test that passed without reaching its case, again

Milestone 7 recorded six of these. This run added one, written in this
run, by the same hand that wrote the rule against them.

`TestBuildVersionIsNeverEmpty` asserted `buildVersion() != ""` and
passed — while never reaching the fallback it existed to check, because
a test binary carries build info of its own. Mutating `return devel` to
`return ""` left it green. The reading was split into `versionFrom`,
which takes the build info as an argument, and is now driven through
five cases including the two that cannot occur under `go test`.

The lesson is not "write better tests". It is that **a passing test is
evidence of nothing until the mutation is run**, and that knowing this
does not confer immunity.

## Three ways of looking that did not work

Worth recording because each looked like it was working.

**Phrase matching over a hard-wrapped document.** The first scan for
self-declared deviations missed I-82, whose "This is a deviation" spans
a line break. Redone over normalised whitespace. Every prose search in
this repository has the same hazard.

**A non-greedy regex to the next heading.** Splitting ERRATA into
entries with `^### (I-\d+):.*?(?:\n### |\z)` consumes the next heading
and returns **every other entry** — 56 of 112. Caught only by the sanity
check that refuses to run on an implausible count. Every scan in `audit`
now carries one, and this is why.

**A tightened pattern that silently dropped a case.** Between the first
and second deviation scans, `not applied` became `is not applied`, which
stopped matching I-55. It was classified by default rather than by
reading. Reading it afterwards confirmed the default was right, which is
luck rather than method.

## The review was mostly right, and where it was wrong that mattered too

Of roughly thirty findings, two were wrong: `docs/audits/` had been
removed deliberately, so the dangling reference was the artifact rather
than the absence; and the injury-tail consolidation it proposed already
existed at one call site.

One of the refutations was itself wrong. `docs/audits/` was reported
here as "verified false" when the review had been correct at the time it
was written. Being wrong about a reviewer being wrong is the expensive
kind, and it argues for the same discipline in both directions.

Two recommendations were **adapted rather than followed**, and the
reasons are in the `audit` package doc rather than in a commit message
nobody will find: test references in ERRATA became a gate on the citing
COVERAGE row, because fifty copied test names is a second place to rot;
and the proposed split of the long narratives into a decision-history
document was declined, because two documents can disagree about one
decision.

## What the release itself proved

The README's install line was the one claim on the page that could not
be verified by running it, because the tag did not exist. Everything
else was executed: every command, every flag, the character card
regenerated and diffed byte for byte against the page.

That card also caught something small and characteristic. It was first
pasted hand-wrapped, with the empty name line and the provenance footer
tidied away — an edited example presented as output. And the fence was
tagged ` ```markdown `, which means **prettier formats its contents**;
it had already repadded the characteristics table out of agreement with
what the tool prints. Retagged ` ```text ` and asserted equal.

After the tag: installing from the module proxy into an empty `GOPATH`
reported `v0.1.0-alpha.1`. That is why the check installs rather than
builds — a `go build` in the work tree reports a pseudo-version whatever
the tag says, so building can never prove the tag reaches the binary.

## What would be done differently

**Write the gate before the plan estimates the work.** The plan for
#118 predicted eight rows from a measurement that counted the wrong
thing. The gate was the correct measurement and it took ten minutes to
write. Estimating from a hand count when a machine count is cheap is how
a plan acquires a number nobody rechecks.

**Sweep for falsified claims in the same commit, not the next one.**
Every link in the chain above was found late. The commit that changes a
fact should search for the sentences asserting it — `grep` for the claim
before writing the code that breaks it.

**Say what a green check proves.** `task` green means less for a prose
change than for a code change, and #121 said so in its own pull request
because PRD prose is reviewed rather than gated. That sentence should be
in more of them.

## What is open

**README and the PRD are the only ungated documents**, and this run was
the second time drift appeared in exactly those two. No workable gate
has been proposed — one was tried and rested on a wrong fact about
`--career craftsman`. They are also the two documents most needing a
human reader, which is not a coincidence. Recorded in the `audit`
package doc as a position rather than an omission.

Everything else is in [RELEASE_READINESS.md](RELEASE_READINESS.md),
finding by finding.

Nothing about the rules is open. What is untested is contact with users,
which is what an alpha is for.
