# Prerelease Review

The codebase completes the practical v1 PRD very well—roughly 97–99%—but
the claim that “nothing … is outstanding” is slightly too strong.

## What Is Convincingly Complete

- All ten functional areas have implementations: characteristics,
  homeworlds, education, 13 careers, skills and Knowledges, aging,
  muster-out, canonical records, dice, and event logs.
- Both interactive and automatic generation exist, along with batch,
  render, and replay commands (`cmd/t5chargen/main.go`).
- Replay reconstructs generation from the seed and recorded choices,
  compares events, and then compares the complete derived record
  (`chargen/replay.go`).
- Character records contain the required provenance and generation inputs
  (`chargen/character.go`).
- The project has golden records for every career, schema checks,
  replay-tampering tests, CLI tests, dice tests, and uncommon-branch tests.
- `go test ./...` passes across all 17 packages.

## Principal Functional Gap

The Noble `Capital***` skill-table result remains deliberately
unimplemented. The rules require identifying the “highest held” land-grant
world, while the record does not retain enough information to rank grants.
Selecting it currently produces an error (`docs/COVERAGE.md`, Career 11).

This is a narrow, uncommon branch, but it falls within the broad language of
FR4 and FR5 rather than an explicit PRD non-goal.

## Other Caveats

- Mentoring and Training Course are unavailable because they require the
  non-human `Tra` characteristic. That is consistent with the human-only
  scope, although FR3 names Mentoring without explaining that it is
  unreachable.
- General dice procedures above 10D and half-dice are deferred because
  character generation never uses them. This satisfies chargen needs,
  though “xD throws” could be read more broadly.
- Merchant ship ownership during Fame, the alternative real-birthday
  method, and several in-play effects are omitted for documented ordering
  or non-goal reasons.
- Some valid but unusual branches are intentionally unreachable under the
  default auto policy, but remain usable through the interactive `Decider`
  path.

## Documentation Cleanup

- The PRD says all requirements are complete despite the outstanding Noble
  result (`docs/PRD.md`, Milestones).
- The README still says the Skill/Knowledge distinction is a non-goal, even
  though it was restored and implemented.
- The PRD retains superseded milestone text and two closure paragraphs,
  making the current scope harder to determine than necessary.

## Conclusion

This is a mature, unusually well-audited implementation of the PRD, not
merely a broad prototype. The user-facing goals are complete and the literal
rules coverage is nearly complete, with one genuine career-rule hole and
several documentation inconsistencies.

This review did not independently verify the Traveller rules against the
private source PDFs. It validates repository-to-PRD alignment, not absolute
rules accuracy.

## Milestone Document Audit

The substantive “shipped” claims in `docs/MILESTONE-4.md` through
`docs/MILESTONE-7.md` match the current implementation and tests. The main
problems are temporal presentation and one overstated completion claim.

### Findings

1. **Milestone 7 overstates completion.** It says “Nothing the PRD asks
   for” remains, then immediately identifies the Noble `Capital***` rule as
   deferred (`docs/MILESTONE-7.md`, What is left). The engine explicitly
   returns `errNotImplemented` if that cell is selected
   (`chargen/careerrun.go`, `awardOpenCell`). Because FR4 and FR5 broadly
   require Noble mechanics and skill awards, the document should call this
   an accepted PRD exception or narrow the PRD.

2. **Milestone 6 reads as a current conclusion even though Milestone 7
   reversed it.** It says Knowledges are a v1 non-goal, describes output as
   Fighter-5, and leaves Flight School deferred. Current code implements
   Knowledge–Knowledge–Skill progression, Knowledge caps, Career and World
   Knowledges, and Flight School. This is understandable as history, but the
   file needs a prominent supersession notice linking to Milestone 7.

3. **The code comment for the remaining `Capital***` gap is stale relative
   to Milestone 7.** It says no Book 1 chart supplies a World Knowledge
   naming convention (`chargen/careerrun.go`, `awardOpenCell`), while
   Milestone 7 correctly says that blocker was resolved and only land-grant
   ranking remains. The runtime error itself correctly names the remaining
   blocker.

4. **Milestone 4 contains superseded “Still deferred” and “Not in this
   milestone” sections without clearly labeling them as historical state.**
   Later Education, Rogue previous-career Schemes, interactive mode, batch,
   replay, and the schema are all implemented now. A short historical
   snapshot notice would prevent confusion.

5. **Milestone 5 has the same temporal ambiguity.** Its “What is left” list
   now contains three implemented areas and only one genuinely deferred
   result. It should point explicitly to Milestones 6 and 7 for current
   status.

### Claims Confirmed Against the Code

- Milestone 4: aging, career changes, all careers, Fame, muster-out, land
  grants, ship shares, and birthdates.
- Milestone 5: interactive mode, refusal-capable `Decider`, batch output,
  replay inputs and verification, later education, assigned schools,
  waivers, homeworld selection, and schema validation.
- Milestone 6: OTC and NOTC, and Rogue previous-career Schemes.
- Milestone 7: reserve resignation, Flight School, branch changes, Scholar
  rank-title rendering, Skill/Knowledge progression, and Career and World
  Knowledges.
- Milestone 7's 112 interpretations and absence of Git tags remain accurate.
  The four resolved audit findings it originally cited now live in
  `docs/audits/` rather than private `.issues` state.

All tests pass across 17 packages, including the documentation gates. Those
gates validate coverage rows, named tests, milestone references in Go
comments, and policy versioning, but deliberately do not detect stale
free-form milestone prose.

Overall, the milestone history is technically strong. Milestones 4–6
preserve obsolete “current” conclusions without enough signposting, while
Milestone 7 makes one completion claim broader than the code supports.

## Coverage Document Audit

`docs/COVERAGE.md` is mostly trustworthy and unusually well supported, but
it contains one direct contradiction, one stale milestone status hidden by
an incomplete audit gate, and two rows without the direct tests its own
contract promises.

### Findings

1. **The Skill/Knowledge status is factually stale.** The Foundations table
   says the Knowledge-6 cap and Skill/Knowledge distinction are v1
   non-goals. The same document later says both are covered, and current
   code implements the cap through `KnowledgeMax` and `levelCap`. The
   Foundations row should be rewritten.

2. **`Capital***` says `deferred (M7)` even though M7 is closed.** The
   deferral itself is truthful—the code returns `errNotImplemented`—but its
   ownership status is false. The audit test misses this because
   `closedMilestones` in `audit/docs_test.go` lists only M1–M6. The row
   should become an accepted or unowned deferral, or M7 should be added to
   the closed list so the test forces that correction.

3. **Two education rows advertise incomplete test coverage.** “Human Tra
   checks at Edu/2” says a direct test is pending, while “Apprenticeship
   unrestricted skill list” names no test. The code exists, and adjacent
   tests exercise Apprenticeship indirectly, but this falls short of the
   document's stated contract that each rule maps to a test.

4. **The document names the coverage gate at the wrong path.** It cites
   `docs/docs_test.go`; the gate lives at `audit/docs_test.go`.

5. **The introduction retains milestone-3-era planning as present-tense
   status.** It assigns muster out, aging, career changes, and Fame to
   milestone 4 even though all have shipped. This should be marked as
   historical context or replaced with the current outstanding-scope
   summary.

6. **The documented status vocabulary is no longer exhaustive.** The
   header permits “covered,” “deferred (M#),” and “interpretation,” while
   rows also use “unreachable,” “out of scope,” “deliberately not
   implemented,” and unowned deferrals. This weakens the promise of
   mechanically meaningful statuses.

### Claims Confirmed Against the Code

The implementation references, named tests, career sections,
interpretation citations, policy choice points, schema fixtures, and
transcribed-data accounting all pass the existing audit suite. The major
coverage claims for careers, aging, muster-out, replay, batch, interactive
generation, Skill/Knowledge progression, and rendering agree with current
code.

The remaining documented exclusions generally match implementation and PRD
scope:

- Many Dice and half-dice
- Non-human `Tra`, `Caste`, and clone rules
- QREBS allocation and masterpiece appreciation
- Merchant ship ownership during Fame
- Default-skill use during task resolution
- Alternative player-birthday input
- In-play advancement

This audit checks document-to-code truth. It does not independently verify
the rule interpretations against the private source PDFs.

## Errata Document Audit

`docs/ERRATA.md` is substantially credible as an implementation rationale,
but it is not fully confirmed. One interpretation is now reachable without
having been resolved, two entries contain stale implementation status, and
the automated gate confirms cross-reference presence rather than semantic
correctness.

### Findings

1. **I-47 leaves a now-reachable rule unresolved.** It says repeated Scout
   stints create an open question over whether the Sanity penalty is
   calculated per stint or across all Scout service, and says the case was
   unreachable until career changes existed. Career changes are now
   implemented. Current code silently chooses the per-stint interpretation
   by calculating from only `r.record.Terms` in
   `careerRun.recordSanityMod`. The interpretation should be decided,
   documented, and tested for a Scout who leaves and returns.

2. **I-36 says commissioned branch rerolls remain deferred under I-34.**
   Milestone 7 implemented both promotion selections and commission
   rerolls. The earlier I-34 text correctly records the implementation,
   making I-36 internally stale.

3. **I-100 says Masters, Professors, Medical School, and Law School are not
   implemented.** All four are implemented and covered by education tests.
   That historical sentence needs correction or a dated supersession note.

4. **I-35's heading is malformed as `### I-35:### I-35:`.** The audit
   regular expression still happens to recognize it, so the test passes,
   but the rendered heading is visibly wrong.

5. **The audit provides weaker confirmation than the document's
   introduction suggests.** `TestEveryInterpretationIsCited` checks only
   that every `I-N` heading appears somewhere in `COVERAGE.md`. It does not
   verify that the named implementation exists, that the implementation
   follows the selected reading, that a test exercises the interpretation,
   or that later code has not made prose stale.

6. **I-2 records a latent implementation hazard rather than a completed
   interpretation.** `firstReceiptLevels` uses skill level as a proxy for
   receipt count, which is unsound for container skills held at Skill-0
   after their first two receipts. The document says no current call site
   exposes it. That appears accurate, but it should remain tracked as
   technical debt because a new multi-level container award could activate
   it.

### Claims Confirmed Against the Code

- All 112 interpretation identifiers exist and are cited by
  `docs/COVERAGE.md`.
- The sampled implementation sites exist and generally match their selected
  readings.
- I-4's “known residual” is explicitly superseded by I-9, and current data
  uses canonical Master Skill List names.
- The intentional exclusions for QREBS, Merchant Ship Owner Fame,
  per-title land-grant hexes, the alternative birthdate, and Musician's
  unavailable Knowledge list match current behavior.
- The complete Go test suite passes across all 17 packages.
- No character currently stamps an applied deviation into `errata`; the
  schema and audit documentation acknowledge that no output-altering
  deviation currently does so.

Absolute confirmation of the page quotations and selected readings would
require access to the private Traveller PDF. This audit confirms repository
consistency, not the source text itself.

## Documentation Reshaping for v1

The principal recommendation is to stop making every document serve
simultaneously as specification, current status, audit trail, and project
diary. For v1, each document should have one clear job.

### `README.md`: the Product Page

Keep the README concise and user-facing:

1. What the tool does
2. Supported ruleset and scope
3. Installation
4. Primary commands
5. Output and replay guarantees
6. Known v1 limitations
7. Links to deeper documentation

Remove milestone history, implementation archaeology, resolved deferrals,
and detailed rules arguments. Remove the stale statement that the
Skill/Knowledge distinction is a non-goal. List the Noble `Capital***`
result under known limitations while that branch still returns an error.

A suitable opening would be:

> `t5chargen` is a deterministic Traveller5 human character generator. It
> supports interactive and automatic generation, all 13 Book 1 careers,
> JSON and Markdown output, batch generation, and replay verification.

### `docs/PRD.md`: the Frozen v1 Contract

- Preserve the problem, goals, non-goals, functional requirements, replay
  contract, JSON conventions, and architecture.
- Fold amendments into the final requirements instead of retaining a
  chronological edit history.
- Remove the old “Amended,” “Reversed,” “Verified,” and “Resolved” passages
  and duplicate milestone closures.
- State each final interpretation once.
- Move implementation discoveries and decision history to milestone or
  decision-history documents.
- Resolve the `Capital***` contradiction by implementing it, declaring an
  explicit v1 exception, or narrowing FR4 and FR5.

The PRD should let a reviewer determine pass or fail without reconstructing
the history of changing decisions.

### `docs/COVERAGE.md`: the Release Compliance Matrix

Keep this as the authoritative rule-to-code-to-test map, with a fixed status
vocabulary:

| Status             | Meaning                                 |
| ------------------ | --------------------------------------- |
| Covered            | Implemented and directly tested         |
| Accepted exception | In v1 scope but deliberately incomplete |
| Out of scope       | Excluded by the PRD                     |
| Unreachable        | Cannot occur for a v1 human             |
| Play-time rule     | Valid rule outside character generation |
| Interpretation     | Covered using the cited ERRATA reading  |

Then:

- Correct the stale Knowledge-6 row.
- Change `Capital***` from `deferred (M7)` to `accepted exception`.
- Add direct tests for the two Apprenticeship rows.
- Correct `docs/docs_test.go` to `audit/docs_test.go`.
- Remove milestone-era planning from the introduction.
- Require every Covered row to name a direct test or an explicitly
  identified invariant or gate.

For v1, this should be the evidence behind any rules-complete claim.

### `docs/ERRATA.md`: Decisions, Not Open Questions

Classify entries explicitly as:

- Interpretation — implemented
- Deviation — implemented and stamped
- Accepted exception
- Open question

I-47 currently describes an open question while code silently selects one
behavior. Decide and test the repeated-Scout-stint rule before v1, or
classify it as an accepted exception.

Also correct the I-35 heading, remove I-36's stale deferral, update I-100's
education status, and give each implemented interpretation a test reference.
Long historical narratives can move to a separate decision-history document.

### `docs/MILESTONE-*.md`: Archived Development History

Keep the milestone files, but add an unmistakable banner to each:

> Historical snapshot. Status and deferrals reflect the repository when
> this milestone closed. See `COVERAGE.md` for current coverage.

This preserves useful reasoning without allowing old “What is left” sections
to compete with current release status. Milestone 7's completion statement
should acknowledge the remaining `Capital***` exception.

### `docs/PRERELEASE_REVIEW.md`: Temporary Release Checklist

Turn every finding in this review into one of:

- fixed before v1;
- accepted v1 exception;
- documentation-only cleanup; or
- post-v1 issue.

Once resolved, replace the long review with a short release-readiness record
containing the reviewed commit, tests run, known exceptions, source-validation
status, and release decision.

### Additional Documents

Add two small documents:

- `docs/KNOWN_LIMITATIONS.md` for concise, user-impacting limitations only;
  link to ERRATA where more explanation is needed.
- `docs/RELEASING.md` for version bumps, golden regeneration, validation,
  smoke tests, tag creation, and release procedure.

### Recommended Hierarchy

```text
README.md
├── docs/PRD.md                 final contract
├── docs/COVERAGE.md            compliance evidence
├── docs/ERRATA.md              rule decisions
├── docs/POLICY.md              automatic choices
├── docs/KNOWN_LIMITATIONS.md   user-visible exceptions
├── docs/character.schema.json  record contract
├── docs/RELEASING.md           maintainer procedure
└── docs/MILESTONE-*.md         historical record
```

### Recommended v1 Release Bar

Before tagging v1:

1. Decide whether `Capital***` is implemented or an accepted exception.
2. Resolve and test I-47's repeated-Scout behavior.
3. Fix every direct contradiction recorded in this review.
4. Add historical banners to the milestone documents.
5. Make README, PRD, COVERAGE, and ERRATA agree on one scope.
6. Add tests for the two uncovered Apprenticeship claims.
7. Strengthen the closed-milestone gate to include M7.
8. Run the complete suite and smoke-test interactive, automatic, batch,
   render, and replay workflows.
9. Verify page citations against the private Book 1 PDF.
10. Tag the exact reviewed commit and record it in the release-readiness
    note.

The code is close to v1. The remaining work is primarily turning a rich
development record into a clear hierarchy where users see capabilities,
maintainers see evidence, and historical reasoning remains available without
competing with current truth.

## Consolidate the injury outcome tail

Five career implementations repeat the same injury-result plumbing after
calling `careerRun.injury`: set the term's end cause, return on death, and end
the career on disability. A shared helper could remove roughly 35 lines from
`chargen/armedforces.go`, `chargen/merchant.go`, `chargen/scout.go`,
`chargen/scholar.go`, and `chargen/agent.go`.

This is a behavior-preserving refactor. The full suite must pass without any
golden fixture changing; a moved fixture means the refactor altered seeded
behavior and its diff must be investigated rather than regenerated.

## Completed reduction work

The earlier reduction pass also proposed a shared To Begin implementation and
removing the unused `medal.Cite` and `medal.Err` exports. Both have landed:
`baseMechanics.begin` now owns the generic entry path, and neither dead export
remains. Golden wrapper tests, `Watcher`, `characteristicRaiser`, and the
single-chart packages were reviewed and deliberately retained because their
structure carries useful domain or determinism boundaries.

## Audit Backlog Disposition

The package-by-package audit's findings lived in a git-ignored `.issues/`
directory, since removed. Forty-two were resolved by the audit pass
(#81–#95) and each named the pull request that fixed it. Four were left
with a status of "finding stands", and this is what became of them, checked
against the code on 2026-08-28.

**Three are closed and need no tracking.**

- _Dead-data gate launders validator reads through helpers._ Resolved by
  recording the limit rather than fixing it: no static test over this
  source can decide a field reached through a shared helper with a
  selecting argument, because what makes it unread is which value the
  argument takes. The limit is written into `audit/deaddata_test.go`
  beside the name-collision limitation it already records.
- _Knighthood arithmetic is unvalidated_ and _a priced benefit may carry
  no price._ Both fixed by the same change, and both had overstated their
  impact: `benefit_test.go` already pinned every value, so nothing was
  ever silent. What the load-time guards buy is the failure arriving as a
  data fault naming the row rather than as an assertion in an unrelated
  test. `closedEntitlements` closes the reverse direction the tests left
  open.

**One survives, and has since recurred.**

- _README and PRD are the only ungated documents._ The drift it described
  was real and was fixed (#83); the gate it proposed was not implementable
  and rested on a wrong fact about `--career craftsman`. The observation
  underneath is still true, and this review is its second confirmation:
  the README's stale Skill/Knowledge non-goal and the PRD's milestone
  contradictions above are both drift in exactly the two documents nothing
  checks. Every other document is gated — COVERAGE's rows and statuses,
  ERRATA's citations, POLICY's rules and version, the schema and its
  examples, and Go comments naming a closed milestone.

  Tracked as a post-v1 issue rather than a release blocker. No gate has
  been proposed that would work, and the two documents are the ones most
  needing a human reader.

**The reduction plan does not survive.** Its one item this review carries
forward — consolidating the injury outcome tail across five career files —
describes work already done. There is one call site, `careerrun.go:179`,
and the comment above it records the consolidation.

## I-47 Disposition

**Decided 2026-08-28: charged per stint.** The review was right that the
entry described an open question while the code silently chose a reading;
it had done so since career changes made two Scout stints possible.

The reading stands as implemented, for reasons the entry now carries:
chart 05 prints the Sanity rule inside the Scout's own box rather than in
a chapter about the character, it rounds within a period of service —
three terms cost what two cost — and the reason it gives is an unbroken
stretch of endurance that leaving ends. p. 134's "Knowledge equal to the
number of terms served" totals across stints instead, and the difference
between the two is where each sentence is printed and what it is for.

What was missing was not the decision but the evidence. A Scout who
serves three terms, leaves and returns for one owes -1 where summing the
stints would owe -2, and nothing tested it because the auto policy never
changes careers. `TestScoutSanityIsChargedPerStint` drives it with a
decider that leaves and returns, and rejects the summed reading
specifically: mutating the engine to total the stints fails it at "a
stint of 3 terms owes San -2, want -1".

Release bar item 2 is closed. Item 1 — whether `Capital***` is
implemented or an accepted exception — was closed as an accepted
exception in #112.

## Citation Sweep

Release bar item 9, run 2026-08-28 against Book 1, Print Edition 5.1.

**Method.** Every page of the PDF extracted separately, then every quoted
string in ERRATA.md checked against the pages its own entry cites.
Matching normalises ligatures, smart quotes, dashes and hyphenation
across line breaks; splits a quote at `...` and at the `/` that joins
chart cells, requiring each run to appear; accepts an editorial `[s]`
either kept or dropped; and searches consecutive cited pages joined, for
a quote that runs over a page break.

**Result.** 254 quotes checked across the 112 interpretations.

|                                 |     |
| ------------------------------- | --- |
| found on a page the entry cites | 213 |
| found only elsewhere            | 3   |
| not matched                     | 38  |

**Thirteen citations were wrong or missing, and are fixed.** The shape
was consistent: an entry quotes chart text, names the chart by number,
and never gives its page — "chart 13's" for a rule on p. 87, "chart 08"
for one on p. 82. A reader who does not already know which page a chart
sits on cannot check those. One was wrong rather than absent: I-107
attributed "reduce CC by negative Mods and Flux" to p. 65, and the line
is on the career charts.

**The three remaining "elsewhere" are an artefact of the checker**, not
of the document: the quotes belong to I-65, I-69 and I-77, each of which
cites the right page, and the checker attributed them to a neighbouring
entry. Verified by hand.

**The 38 unmatched are not citation faults.** Three are the repo quoting
its own earlier prose rather than the book. The rest are PDF extraction
artefacts — the raw layout joins words across column boundaries
("acharacter" for "A character" on p. 67), and chart cells and formulas
extract in an order the page does not read in. Each was spot-checked
against the page rather than assumed.

**The sweep is now a gate.** `task citations` runs it, against a PDF the
caller names; everywhere else it skips, the rules being a purchased
artifact this repository does not redistribute. Rerunning it after every
change to ERRATA.md is cheaper than another manual pass, and it already
caught three citations a manual pass had missed — the first run of it as
a gate failed on I-75 twice and I-86 once, each quoting text and naming a
chart or a cross-reference where a page belonged.

It fails only where a quotation is printed **somewhere the entry does not
cite**, which is what a wrong citation looks like. A quotation the
extraction cannot find at all is reported and not held against the
citation: that is usually the extraction, and thirty-eight of them are.

**What the sweep does not establish.** That a quote appears on the page
it cites is not that the reading drawn from it is right. This checks
provenance, not interpretation.

## Smoke Test

Release bar item 8, run 2026-08-28 against a built binary at d7dbd10,
schema 0.33.0 / engine 0.44.0 / policy 0.25.0.

`task` green across all 17 packages; `task citations` green.

| workflow    | checked                                                                                                                                                    |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| automatic   | a record written, its provenance and `inputs` complete, and the same seed twice byte-identical                                                             |
| interactive | answers on stdin produce `policy_version: "none"`, every choice event recording `player`, and prompts shown; an abandoned session writes no file           |
| batch       | five records to JSONL with seeds derived 100–104, three to a directory as `character-2NN.json`; refused without `--auto`                                   |
| render      | sheet and history transcript, a JSONL run rendered a card at a time, a directory refused, and `--format` gone as of #101                                   |
| replay      | the automatic record, the interactive one, and every line of the run; tampering refused; foreign provenance refused and re-run under `--ignore-provenance` |

Flags exercised: `--name`, `--seed`, `--current-year`, `--career`,
`--homeworld` both supplied and `random`, `-o`, `--force`. A partial UWP
and a career that cannot open a lifepath are both refused with a usage
status.

**Two things the smoke test confirmed rather than assumed.** The
interactive record replays — the milestone-5 guarantee that a session a
person drove is a record like any other. And the two records it generated
were dropped into `chargen/testdata` and validated against
`character.schema.json` by the repository's own gate, which globs that
directory, so the schema describes what the binary writes and not only
what the fixtures happen to contain.

One thing looked wrong and was not: `rolled_homeworld` is absent from the
top level of a record generated with `--homeworld random`. It belongs to
the `inputs` block, which is where replay reads it, and the record round
trips.

Release bar items 8 and 9 are closed. What remains is item 10, the tag.
