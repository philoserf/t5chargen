# Milestone 7 — the deferrals that were still real

Milestone 6 closed and every requirement was met, so the question was what
deferred work was left that was worth doing. A sweep of COVERAGE.md's
deferred rows, every `Deferred:` block in the Go source and the ERRATA
entries behind them found four candidates. **Two of them rested on
premises that were no longer true, and a fifth was added by reversing a
decision.**

Cites are to Book 1, Print Edition 5.1.

## What shipped

Eight pull requests, in the order they landed.

|                             |                                                                             |
| --------------------------- | --------------------------------------------------------------------------- |
| Documents back in step      | #103 — four stale claims, and a gate on POLICY.md's stated version          |
| Resigning from the Reserves | #104 — p. 67, deferred four milestones on a cost that had expired           |
| Flight School               | #105 — not the step C row chart C makes it look like                        |
| Branch changes              | #106 — three charts against one prose line, plus a rule in neither          |
| The Scholar's rank titles   | #107 — "Lecturer _of Major_", composed on the sheet                         |
| The distinction returns     | #108 — the non-goal's reason was wrong, and there was nothing to transcribe |
| Knowledge, Knowledge, Skill | #109 — p. 134's progression, which changed what every character is          |
| Career and World Knowledges | #110 — the two the same page computes rather than rolls                     |

## What reading the pages kept doing to the plan

Four of the eight arrived shaped differently from the plan that approved
them, and in every case because someone went back to the printed page.

**The deferral's reason had expired.** I-55 deferred resigning from the
Reserves because the Check "would consume two faces of the seeded stream
in every Armed Forces character ... for an outcome that never varies".
That was right about the cost and wrong about the necessity: it assumed
the throw had to happen to reach the decision. Ordering the offer first
removes it entirely, and the two rules that established that ordering —
OTC/NOTC and the Rogue's Scheme — had shipped in milestone 6, after the
deferral was written and before anyone reread it.

**The deferral's reason was simply false.** Flight School was deferred
because it "awards a 'Flight Branch' that nothing else in the tool
models". Branch is modelled, Flight is a printed Spacer branch, and p. 66
says outright that the degree confers none — in a sentence
`selectBranch`'s own doc comment had quoted all along. The claim had also
been repeated in a pull request description before anyone checked it.

**The rule was larger than the chart suggested.** Chart C files Flight
School with a Pre-Req and a Duration, which reads as a row a school
leaver applies to. Three sentences elsewhere say otherwise: it is
attended "in the first year of his first term in the Navy, Army, or
Marines" (p. 60), it is offered rather than assigned, and there are two
routes in. None of them is in the chart.

**The disagreement had a third party.** I-34 recorded chart 08 against
p. 66 and left the reading open. Chart 07 and chart 12 print the same
sentence, which makes it three charts against one — and p. 66 carries a
second rule, the reroll a commission allows, that nothing disputes and
that the entry had filed with the disagreement instead of implementing.

## The non-goal that was wrong

Milestone 6 scoped out the Skill/Knowledge distinction, arguing that "the
container list does not survive contact with the skill list". That
reasoned from **chart MS**, which prints no contained Knowledges for any
container and calls its own Knowledge list "advisory".

The lists are not in chart MS. They are in the skill descriptions,
pp. 135–167, which print a `KNOWLEDGES` sidebar for six containers, a
titled box for Engineer, and inline enumerations for Animals and Pilot.
And they had been transcribed into `master_skill_list.json` the whole
time.

So the chunk planned as a transcription job contained no transcription.
Checking a reading against the data twice found the data right and the
reading incomplete — Driver's `Automotive`, Flyer's `Aeronautics` and
Seafarer's `Aquanautics` are printed sidebar entries missed by reading
only the prose lists, and `Winged` and `Sub` are deliberate registry
spellings with their own rows in I-9's mapping.

Two of the non-goal's three objections survived, as recorded limits
rather than as scope: Language is excepted by p. 134 in the sentence that
lists the containers, and Musician has no list printed anywhere (I-111).

## What the progression cost

p. 134 changed every generated character. A container skill's first two
receipts buy Knowledges, so five levels of Fighter leave Fighter-3 and a
Knowledge-2 rather than Fighter-5 — which is the page's own worked
example, and what the tool had been getting wrong since the careers
landed.

Three things the plan did not predict, each found by a test rather than
by reading:

- **The Knowledge-6 cap follows the name, not the caller.** Career tables
  award Knowledges outright, and capping by call site let Sub reach 7 and
  Bay Weapons 9.
- **Skill-0 is a real state.** A container received once or twice sits on
  the sheet at 0, which is not the same as absent, and `receipts` carries
  what the level cannot.
- **Flight School's "1x Pilot-3" runs through the progression**, Pilot
  being a container. p. 61's "Pilot+3 for a total of Pilot-4" still holds,
  and only because that character was already past his second receipt.

## What the milestone taught about tests

Six times a test passed while never reaching the case it named. Twice in
Flight School — a sweep whose Honors graduates never entered a service
career, and a first-term limit invisible because accepting closes the row.
Three times in the Career and World Knowledges — an age anchor that hides
at 18 because `(18-2)/4` and `18/4` are both 4, a re-entered career the
auto policy never produces, and a "no schooling" branch that never fires
because every auto character goes to school. Once in the Knowledge cap,
where the test asserted the constant and not the rule.

Each was found by mutation, and the discipline that found them is the one
this repo already had: revert the fix, confirm the exact symptom, and
**check the mutation actually fired** — twice it did not, once failing to
compile and once landing in the wrong function, and both looked like a
test doing its job.

## What is left

Nothing the PRD asks for. All ten functional requirements and all seven
milestones are complete, and 112 interpretations are recorded.

One rule stays genuinely deferred: chart 11's `Capital***` cell. Its
stated blockers were answered in this milestone — a World Knowledge is
modelled and chart MS gives the naming — but the other half of the cell
survives them, "highest held noble Land Grant" ranking a character's
grants against one another, which needs the per-title hex table I-83
declines to invent.

Everything else in COVERAGE.md that is not covered is a non-human rule, a
value generation never reads, or a task-resolution concern: they are
listed there with the reason, and each is a decision rather than an
omission.

What is open is not a rule. Nothing has been released — there are no tags.
