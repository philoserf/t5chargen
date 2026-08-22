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

Scope (added 2026-08-20, with homeworld skills): "receipts" are career
receipts — skills received during this career (a table C award in an
earlier term). Levels held at career entry from chart B homeworld grants
(p. 56) are not receipts under the Job/Hobby rule: they do not demote the
determination, which awards its full Skill-4/Skill-2 on top of the granted
level. The alternative (any held level demotes to +1) would make a
homeworld grant strictly worsen the character's eventual Job skill, which
the chart's "later receipts" language cannot mean. Skills granted during
career entry itself (a To Begin outcome, milestone 3) are career receipts:
the baseline is captured before the begin seam runs, so they demote a
later determination.

Implemented at `chargen/careerrun.go` (`careerRun.firstReceiptLevels`,
against the career-entry baseline `entryLevels`; career-generic since the
runner extraction, applying to every registered career).

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

### I-4: One skill, several printed spellings (pp. 56, 60, 78, 132, 154)

Book 1 spells two skills inconsistently across its own charts, and the
engine merges skill awards by exact name, so one spelling must be
canonical:

- **High-G**: chart B p. 56 prints "Hi-G" (Oc row); p. 60 and chart 04
  p. 78 print "High-G"; the Master Skill List p. 132 prints "High-G"; the
  p. 154 description heading prints "HI-G" ("Hi-G (High-Gravity,
  Hi-Gravity)…").
- **Hostile Environ**: chart B p. 56 prints "Hostile Env"; chart 04 p. 78
  prints both "Hostile Env" (table E) and "Hostile Environ" (table C); the
  Master Skill List p. 132 prints "Hostile Environ"; the p. 154 heading is
  "HOSTILE ENVIRONMENT".
- **Programmer** (added 2026-08-20, with education): chart C p. 60 prints
  "Program" in The Trades; chart 04 p. 78 and the Master Skill List p. 132
  print "Programmer".

The Master Skill List (p. 132) is taken as the canonical registry:
**High-G**, **Hostile Environ**, and **Programmer**. All data files use
these spellings regardless of the source chart's printed form, so grants
from different charts stack as one skill.

Implemented in `world/data/homeworld_skills.json`,
`career/data/citizen.json`, and `education/data/education.json`.

Chart C p. 60 abbreviates further; `education/data/education.json`
normalizes nine of its printed forms to the registry spellings already in
`career/data/citizen.json` so grants stack: Hvy Wpns → Heavy Wpns,
Battle Dress → BattleDress, Slug Throw → Slug Thrower, J-Drives → Jump
Drive, M-Drive → Maneuver, P-Systems → Power System, Winged → Wing,
Sub → Submersible, Navigation → Navigator. (Chart C's "Ship" is left as
printed: Seafarer Ship and piloted Spacecraft are distinct skills.)

Known residual: for five of these the shared spelling still differs from
the exact MSL p. 132 strings (Heavy Weapons, Slug Throwers, Jump Drives,
Maneuver Drive, Power Systems), and `career/data/citizen.json` itself
splits Navigator (chart C p. 78) from Navigation (job table). A
registry-wide canonicalization is deferred to its own change.

### I-5: a failed education year consumes a year (pp. 59-60)

"Each Success is one year" (p. 59) says what a pass costs; the chart's
Duration says what the full program costs. Neither states whether the year
of a failed Pass/Fail check elapses when attendance ends unwaived.

Two readings:

1. **The failed year elapses** — the character attended the year and
   failed it. (Implemented; also consistent with waived reinstatement,
   where the year passes without a skill.)
2. Only passed (or waived) years elapse; a dropout leaves mid-year.

Implemented at `chargen/education.go` (`passFailYear`).

### I-6: human Tra checks use Edu at half value, rounded up (pp. 55, 60)

Chart C's Apprenticeship Pass/Fail checks Tra, which v1 humans lack.
"Training and Education can be substituted for each other at half value"
(p. 55) supplies the substitution; no rounding rule is stated for half
characteristics. Rounded up, extending "the practice is to always round in
favor of the rolling player" (p. 19, half-dice) to the roll-low target.
The alternative (round down) is a smaller target.

Implemented at `chargen/education.go` (`checkValue`).

### I-7: the Apprenticeship skill list is unrestricted (p. 60)

Chart C's Apprenticeship provides "Skill+4 or Knowledge+4" with no source
list (contrast Training Course's "from School=S"). Implemented as a free
selection from the full Available Skills matrix. The alternative reading
restricts it to the School column like Training Course.

Implemented at `chargen/education.go` (`awardApprenticeship`).

### I-8: Scout "Retry R&R C5" (p. 79; Archive lineage)

Chart 05 box A reads: "To Begin C1 or C2 or C3 / Risk & Reward C1 C2 C3 /
Retry R&R C5 / Continue Int". The Retry row's object is ambiguous. The
Archive's 2008 preliminary Scout box reads "To Begin 6 / To Retry C5" —
there, retry belonged to Begin — but the rules authority is the Book 1
print, whose label is literally "Retry R&R".

Readings:

1. **A failed Reward roll may be retried once against C5** (with the same
   opposite-sign mods). (Implemented — the literal "Retry R&R" object,
   applied to the roll whose failure has no already-suffered consequence;
   retrying a failed Risk would undo an injury already taken.)
2. The To Begin throw may be retried against C5 (the Archive lineage);
   under this reading Scout's Begin gets a second attempt and R&R none.
3. The whole Risk & Reward cycle re-runs once with C5 as the controlling
   characteristic.

Under reading 1, Scout's To Begin has no retry ("Some Careers allow
Retry", p. 65 — Scout's box lists none for Begin).

Implemented at `chargen/scout.go` (`retryReward`).

### I-9: Career and education chart labels vs the Master Skill List (pp. 75-88, 60 vs p. 132)

The career and education charts abbreviate skill names to fit their cells,
and chart 04 prints two different names for one skill: table C row 3-2
reads "Navigator" while table E cell 1-4-6 reads "Navigation". Chart MS is
the authority — "The Master Skill List shows the available skills" (p. 132)
— and its list of Skills "is complete; there are no others available".

The transcriptions therefore store Master Skill List names, not the printed
abbreviations. Storing the printed forms would put two names for one skill
on a character sheet, where they would neither stack nor sum.

The chart label to Master Skill List mapping, with rows added as each
career lands:

| Chart label | Master Skill List | Charts |
| --- | --- | --- |
| BattleDress | Battle Dress | 04 table E; C Available Skills |
| Bay Wpns | Bay Weapons | 04 table E; C |
| Fwd Obs | Forward Observer | 04 table E; C |
| Heavy Wpns | Heavy Weapons | 04 table E; C |
| Jump Drive | Jump Drives | 04 table E; C |
| Maneuver | Maneuver Drive | 04 table E; C |
| Naval Arch | Naval Architect | 04 table E; C |
| Navigation | Navigator | 04 table E (table C prints Navigator) |
| Pilot-ACS | Spacecraft ACS | C |
| Pilot-BCS | Spacecraft BCS | C |
| Power System | Power Systems | 04 table E; C |
| Slug Thrower | Slug Throwers | 04 table E; C |
| Submersible | Sub | 04 table E; C |
| Wing | Winged | 04 table E; C |

Enforced at load time: `skill.Validate` rejects any name in the career,
education, or homeworld data that is not a Master Skill List entry, so a
future transcription cannot reintroduce a variant spelling. This supersedes
the residual noted under I-4.

Implemented at `skill/` (`skill.Resolve`), with the checks in
`career.validateEntry`, `career.validateJobTable`, `education.validate`,
and `world.Grant.validate`.

### I-10: The three Grav knowledges (p. 132)

Chart MS lists a knowledge named Grav under three different parent skills:
Driver, Flyer, and Seafarer. They are distinct bodies of knowledge — driving
a grav vehicle, flying one, and operating a grav seacraft — but the list
prints the same word for all three, and the book's skill notation is bare
("M-Drive-2", "Slug Thrower-2", p. 133-134).

Stored qualified as "Driver: Grav", "Flyer: Grav", and "Seafarer: Grav",
following the book's own convention for its Specialized knowledges
("Career: Scout-4", "World: Egareva-6", p. 134). Storing them bare would
stack three unrelated knowledges into one entry. Knowledges whose printed
name has exactly one parent stay unqualified, as printed.

Chart C's Available Skills matrix resolves its three Grav rows by the group
each is listed under. Chart 04 table E prints Grav once (cell 2-1-1) while
listing every other Driver, Flyer, and Seafarer knowledge, so its single
cell is underdetermined; the character chooses which Grav, in Master Skill
List order.

Implemented at `skill/skill.go` (`addKnowledges`) and
`chargen/careerrun.go` (`resolveSkillName`).

### I-11: The "Spacecraft" cell (p. 78 chart 04 table E)

Table E cell 3-6-6 reads "Spacecraft". Chart MS has no such entry: the
Pilot knowledges are "Small Craft", "Spacecraft ACS", and "Spacecraft BCS"
(p. 132). Small Craft appears separately in the matrix, so the cell covers
the two Spacecraft knowledges.

Resolved as a choice between Spacecraft ACS and Spacecraft BCS, in Master
Skill List order. The alternative reading — a single generic Spacecraft
knowledge outside chart MS — is rejected because the list of Skills is
closed and the knowledge list names both hull classes explicitly.

Implemented at `skill/data/master_skill_list.json` (`labels`) and
`chargen/careerrun.go` (`resolveSkillName`).

### I-12: "Terms x2" counts completed terms (p. 80 chart 06)

Chart 06 box A gives the Officer Promotion target as "Terms x2". The chart
does not say whether Terms counts terms completed or the term in progress.

The p. 65 worked example is the discriminator. It resolves Continue as
"7 +Terms (7 **+0**) = 7" in the character's first term and "7 +Terms
(7 **+1**) = 8" in his second: Terms counts *completed* terms, and is zero
during the first.

Implemented that way, so a first-term officer's promotion target is
0 x 2 = 0 — an automatic failure unless the chart's "+3 if Int 8+" applies,
which yields 3. No Merchant is promoted out of their first term on the
strength of terms alone, which reads as intended: the ladder rewards
service length.

Implemented at `chargen/merchant.go` (`advancementTarget`).

### I-13: Advancement at the top of a ladder (p. 80 chart 06)

Chart 06 lists ranks R X through R2 for ratings and M1 through M6 for
officers, and prints no rule for attempting promotion at the top of either.

A character at the ladder's top does not attempt the throw at all: the roll
is not made and no die is consumed. The alternative — rolling and
discarding a success — would consume dice from the seeded stream and make
an unreachable promotion visible in the event log as a success with no
consequence.

An officer at M6 therefore attempts nothing; a rating at R2 may still
attempt Officer Commission, which targets M1 explicitly rather than the
next rank in class.

Implemented at `chargen/merchant.go` (`eligibleForAdvancement`).

### I-14: The seventh Ship Share receipt (p. 80 chart 06)

Chart 06's Escalating Ship Shares table runs "First 1 Ship Share" through
"Sixth 6 Ship Shares" and stops. A Merchant may serve more than six terms
and so reach a seventh Reward success.

The seventh and later receipts award six shares each — the table's last
printed value — rather than extrapolating to seven and beyond. The table is
printed as a closed list, and the escalation already outpaces "a typical
merchant ship is 10 to 20 shares" by the sixth receipt.

The alternative reading (continue the arithmetic progression) is defensible
and changes only long-career outcomes; the choice is recorded here because
the chart is silent, not because one reading is clearly right.

Implemented at `chargen/merchant.go` (`awardShipShares`).

### I-15: Entry-track failure (p. 80 chart 06)

Chart 06 offers three entry paths — "To Begin 4th Officer Int", "To Begin
Spacehand Dex", "To Begin Temp Auto" — and lists no Begin retry.

A character attempts the one track selected. Failure costs one year ("Each
failed attempt (both Begin or Retry) takes one year", p. 65) and the career
is not begun, exactly as for the Scout's single To Begin.

The alternative reading — that failing a checked track falls through to the
automatic Temp berth — would make the Merchant career impossible to fail
and so make the two checked tracks costless to attempt. That is rejected:
p. 65 treats a failed Begin as ending the attempt at that career, and the
chart marks only Temp as "Auto".

Implemented at `chargen/merchant.go` (`begin`).

### I-16: Disability and advancement (p. 80 chart 06)

Chart 06 says a Merchant disabled by a Risk failure "Muster[s] Out at Term
end with Double Benefits" but does not say whether the term's remaining
steps still happen.

The term completes except for Continue, following the Scout precedent
(chart 05; recorded under the injury rules at p. 65): the Reward roll, the
advancement attempts, and the term's skills all resolve, and only the
Continue throw is skipped. A disability suffered in the field does not
retroactively cancel the promotion earned in the same four years.

Implemented at `chargen/careerrun.go` (`term`) via `termOutcome.endCareer`.

### I-17: Character generation throws are Checks (p. 134; chart 10 p. 84)

The career charts say "Roll for Risk against CC+ Mods" and "Continue Str"
without stating what happens on the highest possible roll. Taken literally
as a roll-low comparison, a target at or above the maximum roll can never
fail — and because several careers set the Continue target from a
characteristic that the Personal skill column raises, a character whose
Continue characteristic reaches 12 would serve terms forever.

Book 1 states the governing rule generally: "Automatic Failure. Without
regard to skill levels, any of the Checks fails on the highest possible
roll. 1D fails on 6; 2D fails on 12; 3D fails on 18." (p. 134, restated
p. 135.) Chart 10 restates it beside its own starred Risk & Reward and
Continue rows: "But, 12 is always automatic failure."

Every character-generation throw is therefore resolved as a Check: To
Begin, the career's Risk and Reward rolls, advancement attempts, Continue,
and the education apply, pass/fail, Honors, and Waiver throws. A natural
12 on 2D fails regardless of target.

This is what guarantees a career can end, and it pairs with the rule at
the other extreme: a Continue roll of exactly 2 is a Mandatory Continue
(p. 66), a roll of exactly 12 an automatic failure.

Chart 10's restatement is read as emphasis on the general rule rather than
a Rogue-only special case; the alternative — applying it only where a
chart prints it — would leave the other charts' characteristic targets
unbounded, which no printed rule supports.

Implemented at `dice/throw.go` (`Check`), applied at every chargen throw
site.

### I-18: The Entertainer rolls no Risk and Reward (p. 77 chart 03; p. 66)

Chart 03 box A reads "Risk & Reward Talent", but the page prints no Risk or
Reward outcome box — unlike charts 05, 06, 08, 10 and 13, each of which
tabulates what failure and success produce. Taken alone, the row invites
the reading that the Entertainer runs the generic p. 65 cycle against
Talent, whose Risk failure would reduce Talent, award a Wound Badge, and
kill the character at Talent zero.

Book 1's career prose settles it. Immediately before the generic Risk and
Reward sequence, it enumerates the careers that replace it:

> The Citizen Career uses a variant of Risk and Reward called Citizen Life.
> Only one roll is made to determine Success or Failure. No Mods are used.
> The Functionary Career uses a variant of Risk and Reward called Office
> Politics. Separate successive rolls are made for Risk and Reward. No Mods
> are used. **The Entertainer Career focuses on Fame and resolves the
> current level of Fame for the character.** The Craftsman Career focuses
> on the creation of Masterpieces and their attendant impact on personal
> success. (p. 66)

The Entertainer's variant is therefore the Fame resolution itself: the
required Flux and its two optional companions. "Risk & Reward Talent" names
Talent as the value governing that variant, in the slot where other charts
name a series of characteristics.

Implemented with no Risk or Reward throw, and with the term's outcome being
whether Fame increased — which is what table B keys its extra skills to
("If Fame Increases 2 and Talent+1"). Consistent with chart 01, which says
of the Craftsman in the same breath: "He does not roll Risk and Reward."

The Archive's preliminary Entertainer also prints no Risk and Reward box
(locate-only; Book 1 governs).

Implemented at `chargen/entertainer.go`.

### I-19: The Entertainer's first term rolls no Fame Flux (p. 77 chart 03)

The prose says "At the start of each Term, events in the Entertainer's
career may change Fame. Roll Flux up to three times (the first is
required...)". The Entertainer Fame And Talent table says otherwise for the
first term: its Term 1 row is "Fame =2D, Talent = Talent", while Terms 2
through 6 each read "Fame +F +F* +F*".

The table governs, being the specific rule. The 2D rolled before Begin
("Before Begin ... roll initial Fame and Talent") is the first term's Fame,
so the first term has no Flux, cannot increase Fame, and cannot earn the
"If Fame Increases" extra skills or Talent.

Implemented at `chargen/entertainer.go` (`resolveTerm`).

### I-20: A Comeback replaces the term's Flux progression (p. 77 chart 03)

The chart gives the Comeback as "Reset Fame to 2D; Talent is unchanged.
Comeback is possible any number of times" without saying when in the term
it happens or how it interacts with that term's Flux rolls.

Implemented as a term-start alternative: a Comeback replaces the term's
"+F +F* +F*" progression rather than preceding it. The Fame table gives one
Fame determination per term, and a reset followed by Flux would apply two.
A Comeback that lands above the previous Fame still counts as an increase
for table B, since the term ended with Fame higher than it started.

The alternative reading — reset, then roll the term's Flux as usual — is
defensible and strictly more generous; it is recorded here because the
chart is silent.

Implemented at `chargen/entertainer.go` (`offerComeback`).

### I-21: Fame is not floored at zero (p. 77 chart 03)

Flux is 1D-1D, so a term can reduce Fame, and the chart prints no minimum.
Its Fame descriptor table starts at 0 ("Unknown") and gives no descriptor
below it.

Fame is left unfloored. A career whose Fame falls to zero or below ends at
the next Continue throw, which targets Fame — an unknown Entertainer is out
of work, which is the outcome the chart's own Continue row produces. Adding
a floor at zero would instead keep such a character eligible to continue on
a natural 2.

Implemented at `chargen/entertainer.go` (`setFame`).
