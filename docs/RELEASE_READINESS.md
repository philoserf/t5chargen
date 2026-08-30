# Release readiness

What was reviewed, what was run, what is knowingly incomplete, and the
decision. This replaces the long prerelease review, whose own last
recommendation was to become this once its findings were dispositioned.
The review itself is in git.

The bulk of this document records `v0.1.0-alpha.1`, the release it was
written for. Later releases add a section rather than overwrite it: what
was verified before a tag is a fact about that tag, and rewriting it
would lose the record the document exists to keep.

## Validated

|                |                                                                                                           |
| -------------- | --------------------------------------------------------------------------------------------------------- |
| **Tag**        | `v0.1.0-alpha.1`, prerelease                                                                              |
| **Commit**     | `5fabd48f12fc4cb8cf7fd822d51b2995a61976a3`                                                                |
| **Ruleset**    | Traveller5 Core Rules Book 1, Print Edition 5.1                                                           |
| **Versions**   | schema 0.33.0 · engine 0.45.0 · policy 0.25.0                                                             |
| **Gate**       | `task` — 17 packages, `-race`, green                                                                      |
| **Citations**  | `task citations` — every ERRATA.md quotation on the page it cites, checked against the private Book 1 PDF |
| **CI**         | green on `main`, running the same `task`                                                                  |
| **Smoke test** | the five workflows against a built binary                                                                 |

The smoke test covered automatic, interactive, batch, render (sheet and
history), and replay. The interactive record is the one that matters: it
carries `policy_version: "none"` and replayed clean at 241 events, which
is the case the PRD says only replay can recover.

## Known exceptions

Three, all deliberate, all recorded where a user or a maintainer will
meet them. [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md) is the
user-facing statement.

- **Chart 11's `Capital***` cell returns an error** rather than inventing
  a ranking Book 1 does not print (I-83). COVERAGE.md's only accepted
  exception, and gated as such in both documents.
- **Two deviations from the printed rule**, I-82 and I-112, each stamped
  into the `errata` field of every record it governed.
- **Musician is awarded whole** (I-111), Book 1 printing no instrument
  list for it.

Everything else absent is a PRD non-goal or a rule outside character
generation, recorded row by row in [COVERAGE.md](COVERAGE.md) under one
of six statuses.

## Disposition of every review finding

**Milestone documents (5 findings) — all fixed.** Milestone 7's
overstated completion, milestone 6 reading as current after 7 reversed
it, the stale `Capital***` code comment, and the superseded "still
deferred" lists in milestones 4 and 5. Historical banners on all four
(#117); the code comment corrected (#112).

**Coverage document (6 findings) — all fixed.** The stale
Skill/Knowledge status, `Capital***` marked `deferred (M7)` after M7
closed, two education rows advertising incomplete coverage, the gate
cited at the wrong path, milestone-3-era planning left in the present
tense, and a status vocabulary the document no longer used (#112, #118).

**Errata document (6 findings) — all fixed.** I-47's open question
resolved and charged per stint (#113); I-36 and I-100's stale
deferrals; I-35's malformed doubled heading; the audit confirming less
than the document claimed, answered by the classification gates (#119);
I-2's latent hazard.

**Early caveats (7) — fixed or documented.** Mentoring's unreachability
now stated in FR3 (#121); the README's stale Skill/Knowledge non-goal
(#117); the PRD's superseded milestone text and doubled closure
paragraphs (#121); the PRD's completeness claim against the Noble gap
(#112, #120). Dice above 10D, half-dice, Merchant ship ownership and
the real-birthday option are PRD non-goals or play-time rules, recorded
in COVERAGE.md. The unreachable-under-auto branches are
KNOWN_LIMITATIONS.md's own section (#124).

**Documentation reshaping (7 recommendations) — all done.** README as a
product page and KNOWN_LIMITATIONS.md (#124); the PRD folded into a
frozen contract (#121); COVERAGE's fixed status vocabulary and its gate
(#118); ERRATA classified with gates on what each kind owes (#119);
milestone documents banner-marked as history (#117); RELEASING.md and
this record (#125).

**Two recommendations adapted, with the reasons recorded in the `audit`
package doc.**

- _A test reference on each implemented ERRATA entry_ — about fifty
  entries — would create a second place for test names to rot against
  the first. Instead a gate requires the COVERAGE row citing an entry to
  name a test. Same guarantee, one place to keep true; it found two
  entries cited only from prose (#119).
- _Move the long narratives to a separate decision-history document_ —
  declined. Reasoning at the site is deliberate here, and splitting
  three thousand lines produces two documents that can disagree about
  one decision.

**Two findings were wrong, and are recorded as wrong.**

- _`docs/audits/` is referenced but missing._ True when written; the
  directory had been removed deliberately, and the dangling reference
  was the artifact.
- _Consolidate the injury outcome tail across five career files._ The
  consolidation already existed. There is one call site,
  `chargen/careerrun.go:179`, and the comment above it records the work.

**One finding survives as a post-v1 issue.** README and the PRD are the
only documents nothing gates, and this review was the second time drift
appeared in exactly those two. No gate has been proposed that would
work — one was tried and rested on a wrong fact — and they are the
documents most needing a human reader. Recorded in the `audit` package
doc rather than tracked as a blocker.

## Release bar

The review's own ten items, each checked rather than assumed.

|                                                   |                                              |
| ------------------------------------------------- | -------------------------------------------- |
| 1. `Capital***` decided                           | accepted exception, gated in both documents  |
| 2. I-47 resolved and tested                       | charged per stint; `TestScoutSanityModifier` |
| 3. Direct contradictions fixed                    | eight mechanical fixes (#112)                |
| 4. Historical banners                             | all four milestone documents                 |
| 5. One scope across README, PRD, COVERAGE, ERRATA | #117–#124                                    |
| 6. Apprenticeship claims tested                   | `chargen/latereducation_test.go`             |
| 7. Closed-milestone gate includes M7              | `closedMilestones` in `audit/docs_test.go`   |
| 8. Full suite and the five workflows smoke-tested | run above                                    |
| 9. Citations verified against Book 1              | `task citations` passes                      |
| 10. Tag the reviewed commit and record it         | `v0.1.0-alpha.1` at `5fabd48`, 2026-08-28    |

## Decision

**Tagged `v0.1.0-alpha.1` at `5fabd48`, 2026-08-28.** The first release
of this repository.

Alpha rather than 1.0 because nothing had been released before and
nobody has used it: the rules are implemented and gated, and what is
untested is contact with users. The procedure is
[RELEASING.md](RELEASING.md).

## After the tag

Every check the procedure asks for, run against the released artifact
rather than a local build.

- `go install github.com/philoserf/t5chargen/cmd/t5chargen@v0.1.0-alpha.1`
  into an empty `GOPATH` succeeds from the module proxy.
- `t5chargen version` reports **`v0.1.0-alpha.1`** — not `(devel)`, not a
  pseudo-version. This was the one line in README that could not be
  verified before the tag existed, and it is why the check installs
  rather than builds: a `go build` in the work tree reports a VCS
  pseudo-version whatever the tag says.
- The five workflows run on that binary: automatic, interactive, batch,
  render, replay. The automatic record replayed at 151 events and the
  interactive one at 241.
- The released binary reproduces README's pasted character exactly,
  compared byte for byte against the page.

---

## v0.1.0-alpha.2 — 2026-08-30

|                |                                                               |
| -------------- | ------------------------------------------------------------- |
| **Tag**        | `v0.1.0-alpha.2`, prerelease                                  |
| **Commit**     | `9733452`                                                     |
| **Versions**   | schema 0.33.0 · engine 0.45.0 · policy 0.25.0 — **unchanged** |
| **Goldens**    | `task goldens` — no fixture moved                             |
| **Gate**       | `task` — green, locally and on the tagged commit in CI        |
| **Citations**  | `task citations` — 254 quotations checked against Book 1      |
| **Smoke test** | the five workflows against a built binary                     |

Nothing about a generated character changed: the only non-test Go to move
since the first tag was the CLI's help text. That is why all three record
versions stand, and it is the claim this release is really making.

### After the tag

The first release built by
[.github/workflows/release.yml](.github/workflows/release.yml), so the
workflow's own first run is part of what was verified.

- The artifacts are built with `go install module@tag`, which is both the
  build and the proof that the published module installs.
- The downloaded `t5chargen_darwin_arm64` matches its line in
  `SHA256SUMS`, reports **`v0.1.0-alpha.2`**, and generates a character.
- That released binary replays a record written by `v0.1.0-alpha.1` —
  151 events from seed 7 — which is
  [COMPATIBILITY.md](COMPATIBILITY.md)'s render-forward promise held
  against a real artifact rather than a fixture.

### Knowingly incomplete

The beta bar in [BETA_READINESS.md](BETA_READINESS.md) is met on every
item tooling can meet. What remains is the part no tooling does: several
independent users completing the core workflows, and their reports
dispositioned. That is what this alpha is for.
