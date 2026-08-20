# ERRATA.md — deviations and interpretations

Deliberate deviations from, and interpretations of ambiguities in,
Traveller5 Core Rules Book 1, Print Edition 5.1. Nothing here is applied
silently: entries are cited at the implementation site, and applied
deviations are listed in each character record's `errata` field when they
alter output relative to the printed rule. Interpretations (readings of
ambiguous text, not deviations) are recorded here for audit but not stamped
into records.

## Interpretations

### I-1: Citizen Job roll landing on the "No Skill" cell (p. 78)

Chart 04 table E contains a "No Skill" cell (A1 B3 C5). The chart says the
first Citizen Life success "provides a Job, randomly on Citizen Skills and
Knowledges with Skill-4" and that "Once determined, Job and Hobby cannot be
changed" — but not what happens when the random roll lands on "No Skill".

Two readings:

1. **The Job remains undetermined**; the next Citizen Life success retries
   the determination. (Implemented — less punitive, and "No Skill" is not a
   skill to determine as the Job.)
2. The Job is determined as nothing, permanently, per "cannot be changed".

Implemented at `chargen/citizen.go` (`determineJob`).

### I-2: Job/Hobby determination landing on a skill already held (p. 78)

Chart 04 says the first success "provides a Job, randomly on Citizen Skills
and Knowledges with Skill-4 (later receipts are Skill-1)" — but not what
happens when the determined Job (or selected Hobby) is a skill the
character already holds from another source (a table C award in an earlier
term).

Two readings:

1. **The award counts as a later receipt: +1.** (Implemented — "later
   receipts are Skill-1" reads as receipts of a skill already held.)
2. The Job determination is itself a first receipt: award the full
   Skill-4 (or Hobby Skill-2) on top of the held level.

Implemented at `chargen/citizen.go` (`firstReceiptLevels`).

### I-3: Hobby selection excludes the determined Job (p. 78)

Chart 04 has the second success provide "a Hobby selected from Citizen
Skills and Knowledges", without saying whether the already-determined Job
may be selected. The ladder's alternation ("successes alternate between
Job or Hobby skill levels") implies two distinct pursuits.

Two readings:

1. **The Job's skill is excluded from the Hobby alternatives.**
   (Implemented — keeps Job and Hobby distinct so the alternation is
   meaningful; also prevents the deterministic auto policy from always
   collapsing them when the Job roll lands the first-listed entry.)
2. Any table E skill may be selected, including the Job's.

Implemented at `chargen/citizen.go` (`determineHobby`).
