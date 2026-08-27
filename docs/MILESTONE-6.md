# Milestone 6 — the rules that were left, and the ones that were not rules

Milestone 6 was planned as seven COVERAGE.md rows. One of them shipped as
a rule, six became a v1 non-goal, and the substantial work of the span
belonged to no milestone at all.

That is the honest shape of it, and it is worth writing down rather than
narrating six rows into a milestone. Forty-four pull requests landed
between milestone 5 closing and this document; four of them are milestone 6.

Cites are to Book 1, Print Edition 5.1.

## What milestone 6 turned out to be

|                              |                                                                                                     |
| ---------------------------- | --------------------------------------------------------------------------------------------------- |
| The Status gate              | #96 — no COVERAGE.md row may defer to a milestone that has shipped                                  |
| The Skill/Knowledge non-goal | #97 — five of the seven rows, and chart 11's `Capital***` cell, scoped out of v1                    |
| OTC and NOTC                 | #98 — chart C rows, not milestone 6, and the substantial rules actually left                        |
| The Rogue's Scheme           | #100 — "A Rogue may select for his Scheme (rather than roll) any previous career" (chart 10, p. 84) |

Two more closed the milestone rather than filling it: #101 dropped
`--format`, the last thing in the PRD's CLI sketch the tool refused, and
#102 gated the claims the Go source makes about its own milestones.

## Why six rows became a non-goal

p. 134 says the first two receipts of a container Skill award a contained
Knowledge instead, and the third awards the Skill at level-1. Five of
milestone 6's rows were that machinery; a sixth, chart 11's `Capital***`,
awards a World Knowledge and goes with them.

Reading settled the rule (#66) without settling whether to build it. What
settled that was the cost. Knowledges are a category of thing the record
does not carry, as psionics is. The rule reaches into every career:
expanding a container is a choice at every award, which needs a POLICY.md
rule and moves every fixture. And the container list does not survive
contact with the skill list — chart MS gives Musician no contained
Knowledges at all, Language is excepted by p. 134 itself, and
`The Sciences` is a chart MS heading that looks like a container and is
not.

What it costs is stated in the non-goal rather than left to be discovered:
the book's own example musters out with Fighter-3 and Slug Thrower-2 where
this tool gives Fighter-5. That is a different character for the same
rolls, not a missing refinement.

## What the span was actually about

The forty PRs that were not milestone 6 fall mostly into two threads.

**The education thread (#67–#80)** began with a played character whose
record read `[Citizen]` after twenty-three Service Academy attendances and
an Edu of F. One clause was behind all of it: p. 62 makes chart C's
Graduation values positions on a scale, so "(If Edu already at this level,
award Edu+1)" is a consolation and not a ratchet (I-105). Fixing it opened
the top of chart C — Masters, Professors, Medical School, Law School — and
then a sweep of every menu the engine shows a player.

**The audit pass (#81–#95)** answered a review of the whole repo. Nine of
its suggested fixes turned out to be wrong, and four issues resolved as
recorded limits rather than as code. The most useful thing it produced was
not a fix: it established that a suggested fix is a hypothesis, and that
the way to test one is to revert it and confirm the exact symptom returns.

## What the milestone taught about documents

Four times in one week a claim in this repo stopped being true and nothing
noticed.

#83 corrected five COVERAGE.md rows that deferred to shipped milestones.
#96 corrected eight more and built a gate for the Status column. #97 left
two stale notes in prose beside the rows it corrected, caught by eye in
#100. #102 found twenty-six Go doc comments deferring work to milestones 3
and 4 long after those delivered it — the `career` package doc still said
v1 ships the Citizen career only.

Each time the rot moved somewhere the previous gate could not see. The gates
now cover COVERAGE.md's Status column, every Go doc comment, and POLICY.md's
stated version against the constant the engine stamps. They do not cover
prose, and no rule was found for prose that would not either miss most of it
or flag most of it. That limit is recorded in the `audit` package doc, where
the next person to propose a prose gate will find it.

## What is left

Nothing the PRD asks for. All ten functional requirements and all six
milestones are complete; what is deliberately absent is listed as a
non-goal there and, rule by rule, in COVERAGE.md.

One rule stays deferred with no milestone owning it, which is the honest
place for it: **Flight School** (chart C p. 60) wants an Honors BA, which
is implemented, but awards a "Flight Branch" that nothing else in the tool
models.

**`render --format txt`** is not deferred but dropped (#101), and the
`--format` flag with it. It bought Markdown with the emphasis markers
stripped, at the price of a second set of golden sheets to hold that
output honest — and a deferral carried for four milestones without ever
being weighed is not a plan.
