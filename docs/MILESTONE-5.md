# Milestone 5 — the two modes, and the record as a contract

> **Historical snapshot.** Status and deferrals reflect the repository
> when this milestone closed. See [COVERAGE.md](COVERAGE.md) for current
> coverage.

The tool now does what the PRD said it would. A character can be generated
by a person answering each choice or by the policy answering all of them;
a run of characters can be generated at once; any record can be re-run from
its own file and checked against itself; and the shape of that record is
written down as a schema rather than implied by the Go types.

Cites are to Book 1, Print Edition 5.1.

## What shipped

Ten pull requests, in the order they landed.

|                      |                                                                                                |
| -------------------- | ---------------------------------------------------------------------------------------------- |
| `Decider` may refuse | an error return, so an abandoned session is not an out-of-range answer                         |
| Replay verification  | re-runs a record from its own file; the inputs block, because seed and choices were not enough |
| Batch                | a run at a time, each member replayable from the line it lands in                              |
| Later Education      | the term a character spends at school instead (p. 59)                                          |
| Assigned schools     | ANM School and Command College, resolved as Education inside the term                          |
| Academy Officer1     | chart C's graduation, transcribed and leading nowhere until now                                |
| The last two waivers | Prerequisite and Honors, the two of p. 59's four that were never offered                       |
| Chart B's world list | thirty-six cells, and the one that names no world                                              |
| Interactive mode     | the other decider the PRD has always named                                                     |
| The JSON Schema      | the record as a contract, and a checker that is not vacuous                                    |

## What replay turned out to need

The contract says replay runs "from the recorded seed and choice events".
That is not enough, and the gap was invisible until something depended on
it.

Two of the engine's inputs were recorded nowhere: the `--career` force and
`--current-year`. The force matters because a recorded answer is an
**index**, and forcing a career holds the first career's option list to a
single entry — so a replay that did not know a record was forced would
offer all eleven eligible careers and read the recorded index against that.
Eleven of the fourteen fixtures are forced. Records carry an `inputs` block
now, and chart B's roll joined it later for the same reason.

The other thing replay needed was a version to check against, and this
milestone got that wrong twice. `engine_version` is the only provenance
gate replay has — `policy_version` is deliberately excluded, because
recorded choices are reapplied and the policy is never consulted — and it
was left standing while the event log changed, twice. An old record then
passes the provenance check and dies somewhere in the middle of the log:
the real message was `"change_career": recorded the answer 3, outside the 2
options` at event 60, blaming a career choice for a build mismatch. The
rule is now enforced where the regeneration happens, so `task goldens`
refuses to rewrite a fixture whose replayed content changed while the
version stood still.

## What the rules turned out to be

Seven interpretations, and the pattern in them is that the printed sentence
usually settles the question once you stop reading past it.

**I-88 to I-90, Later Education.** "Substitutes that process for the entire
term" makes the term the thing replaced, so a one-year Trade School costs
four years. Substitution is conditional on acceptance, so a refused
applicant serves the term after all. And "suspend career resolution"
suspends the Continue throw with the rest of it, which means a suspended
term is not a term served and earns no muster-out roll.

I-89 was reached by reading and then confirmed by testing: make a refusal
suspend the term anyway and generation loops forever, because Apprenticeship
has no prerequisite and takes no time, so nothing advances the clock toward
the aging that has to end a lifepath. The printed reading is also the
terminating one.

**I-91 to I-93, assigned schools.** "Some schools are attended during career
resolution" is a different mechanism from suspending a term, and the
contrast is exact: an assigned school is sited inside a term the character
is already spending, so it costs no extra years, where a suspended term is
the thing being spent.

**I-94, the Academy.** Chart C's "C5=8 BA Officer1" had its Edu and its
degree applied and its Officer1 sitting in a string, doing nothing —
because Officer1 is not something the schooling does to the character, it
is a rank in a career he has not joined yet.

**I-95 and I-96, the last two waivers.** p. 59 names four waiver-able
events and two were offered. Prerequisite was unreachable rather than
unimplemented: the engine offered only the rows a character already
qualified for, which is a waiver with nothing to waive. Honors is the odd
one — its failure "has no effect", so the waiver buys the status and not
the level, which is the shape the page states for the waiver it does
explain.

Working out what the policy should do with those unified the two waiver
rules rather than complicating them: waive what is at stake.

**I-97, deep space.** Chart B's last cell names no world and carries no
UWP. Keeping that apart from the partial UWP FR2 refuses took two attempts:
the first reading made "trade classifications with no UWP" the marker,
which is precisely the shape an existing test already forbade. A caller can
build that shape even though no `--homeworld` string produces it, so the
relaxation would have made FR2's guarantee conditional on nobody exercising
it.

## The schema, and why it is not a library

The record is now `docs/character.schema.json`, draft 2020-12, with a
minimal and a complete example. It is precise about the envelope and
deliberately loose about which payload fields each consequence kind
carries — `omitempty` makes those sets ragged, and fifty-five branches in
the schema would be a second copy of the code. That rule is pinned in
`docs` instead, as an observed golden.

The checker is written rather than imported: 250 lines against six
third-party modules, in a repo with none. The risk in that trade is
specific and this project has met it before — a validator with a bug passes
everything, and a gate that passes everything looks exactly like a gate
that works. So the checker is not trusted on the strength of the fixtures
passing. Every rule the schema states has a record that must fail because
of it, and every keyword the checker implements is mutation-tested.

## What was dropped

A full-screen terminal front end was planned and abandoned partway. It cost
three concurrency bugs, none of them about Traveller: the engine's outcome
was consumed by whichever of two goroutines reached the channel first; an
escape followed by a character is how a terminal sends alt-character, so
the tests that abandoned a session never reached the branch they were
testing and one of them passed for the wrong reason; and a test pipe closed
underneath its own writer.

It would have added twenty-five modules to a repo with none, and what it
bought over the line-based front end was presentation — its own test
asserted the two produce the same character. The `interactive` package
remains the seam a terminal front end would sit on, if one is ever worth
its own project.

## What is left

Milestone 6, which is the rules this milestone did not reach: the Rogue's
previous-career Scheme, chart 11's `Capital***` cell, `Career:`/`World:`
knowledges and Sciences past level 6. Three of the four wait on the same
open question — whether education may award a bare container skill (p. 134)
— which is why they moved together.
