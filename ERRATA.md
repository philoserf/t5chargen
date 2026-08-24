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

Scope (extended 2026-08-23, with Later Education): levels earned at school
during a suspended term (p. 59, I-90) are education, not career receipts —
for the same reason homeworld grants are not, and with more force, since a
suspended term is not career resolution at all. The career-entry baseline
is raised by whatever the schooling awarded, so a mid-career Apprenticeship
in the skill a later Job or Hobby determination happens to land on does not
demote that determination to Skill-1. Raised rather than reset: a genuine
career receipt of the same skill in an earlier term still demotes.

Implemented at `chargen/careerrun.go` (`careerRun.firstReceiptLevels`,
against the career-entry baseline `entryLevels`; career-generic since the
runner extraction, applying to every registered career) and
`chargen/latereducation.go` (`creditSchooling`), pinned by
`TestLaterEducationIsNotACareerReceipt`.

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

| Chart label  | Master Skill List | Charts                                               |
| ------------ | ----------------- | ---------------------------------------------------- |
| BattleDress  | Battle Dress      | 04 table E; C Available Skills                       |
| Bay Wpns     | Bay Weapons       | 04 table E; C                                        |
| Fwd Obs      | Forward Observer  | 04 table E; C                                        |
| Heavy Wpns   | Heavy Weapons     | 04 table E; C                                        |
| Jump Drive   | Jump Drives       | 04 table E; C                                        |
| Maneuver     | Maneuver Drive    | 04 table E; C                                        |
| Naval Arch   | Naval Architect   | 04 table E; C                                        |
| Navigation   | Navigator         | 03 table C; 04 table E (04 table C prints Navigator) |
| Pilot-ACS    | Spacecraft ACS    | C                                                    |
| Pilot-BCS    | Spacecraft BCS    | C                                                    |
| Power System | Power Systems     | 04 table E; C                                        |
| Slug Thrower | Slug Throwers     | 04 table E; C                                        |
| Submersible  | Sub               | 04 table E; C                                        |
| Wing         | Winged            | 04 table E; C                                        |

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

The p. 66 worked example is the discriminator. It resolves Continue as
"7 +Terms (7 **+0**) = 7" in the character's first term and "7 +Terms
(7 **+1**) = 8" in his second: Terms counts _completed_ terms, and is zero
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

### I-20: A Comeback replaces the term and earns no Talent (p. 77 chart 03)

The chart gives the Comeback as "Reset Fame to 2D; Talent is unchanged.
Comeback is possible any number of times", without saying when in the term
it happens, how it interacts with that term's Flux rolls, or whether a
Comeback that lands above the previous Fame is a Fame increase for table B
("If Fame Increases 2 and Talent+1").

Two decisions, both from the printed clause:

1. **A Comeback replaces the term's "+F +F\* +F\*" progression** rather than
   preceding it. The Fame table gives one Fame determination per term; a
   reset followed by the term's Flux would apply two.
2. **A Comeback term earns neither the Talent nor the two extra skills**,
   even when the 2D lands above the Fame it replaced. "Talent is unchanged"
   is read as governing the Comeback outcome, not merely as a note that the
   reset does not also re-roll Talent the way the pre-Begin 2D set both.
   Table B pairs its two extra skills with that same Talent+1, so denying
   one denies both. The Fame table reaches an increase through "+F +F* +F*";
   a reset is a different path.

The alternative reading of the clause — that it scopes only the reset,
leaving a lucky Comeback to count as an increase — is defensible in
isolation, but it makes the Comeback strictly better than the Flux
progression it replaces: a character sitting at low Fame could reset every
term and harvest Talent and skills from the rebound, which the explicit
"Talent is unchanged" reads as written to prevent.

Revisiting either decision changes generated characters and is an engine
version bump.

Implemented at `chargen/entertainer.go` (`offerComeback`, `resolveFame`).

### I-21: Fame is not floored at zero (p. 77 chart 03)

Flux is 1D-1D, so a term can reduce Fame, and the chart prints no minimum.
Its Fame descriptor table starts at 0 ("Unknown") and gives no descriptor
below it.

Fame is left unfloored, and the character sheet omits a Fame at or below
zero: `statusLine` prints the Fame line only when Fame is positive, so an
Entertainer whose reputation has collapsed shows no Fame rather than a
negative one. The generation record keeps the exact value.

A career whose Fame falls to zero or below fails every subsequent Continue
throw, which targets Fame — an unknown Entertainer is out of work, which is
the outcome the chart's own Continue row produces. The single exception is
the p. 66 Mandatory Continue: a natural 2 requires the character to
continue whatever the target, and `continueRoll` applies that before the
roll-low comparison. A floor at zero would not change this — 2D never
rolls below 2, so targets of 0 and -3 fail identically — so the floor is
omitted as unsupported rather than as load-bearing.

Implemented at `chargen/entertainer.go` (`setFame`).

### I-22: Waivers draw on one pool (p. 76 chart 02; p. 59)

Chart 02 gives the Scholar "Waivers. An adverse die roll or decision (in
Position, Promotion, Research, Publication, Tenure, or Continue) may be
waived. Check Soc (2D); Mod minus previous waivers (successful or not)."
That is the Educational Waiver rule of p. 59 word for word, but neither
text says whose previous waivers are counted.

Implemented as one pool: a waiver spent at university makes the next one
harder in a career and the reverse. Neither rule qualifies "previous
waivers" as educational or career waivers, and the shared decay reads as a
single social-capital budget spent across a life.

The alternative — separate counters per system — would let a character
reset the penalty by changing context, which the unqualified wording does
not support.

A waiver negates the adverse outcome; it does not re-roll it. That is the
existing reading of the education waiver, kept here.

All six named events are offered. Five belong to the Scholar's own
mechanics; Continue belongs to the generic term loop, which offers it
where the definition sets `continue_waiver` (chart 02 is the only chart
printing the Waivers box in v1). A waived Continue failure carries the
career into the next term.

Implemented at `chargen/waiver.go` (`offerWaiver`, `careerWaiver`),
`chargen/careerrun.go` (`continueRoll`), counting
`Character.WaiversAttempted`.

### I-23: Every Scholar has a Major and a Minor (p. 76 chart 02)

Chart 02 reads: "Every Scholar has a Major and a Minor. If no degree (and
an associated Major and Minor) then select any Skill or Knowledge from the
Skills List." The final sentence is singular, but the rule it serves names
both.

A Scholar arriving without a degree selects both, from the whole Master
Skill List, and they cannot be the same (p. 59). The selections live on the
career record and take precedence over any education Major and Minor for
the rest of generation, being the more recent ("A character's current Major
and Minor are the most recent ones selected", p. 59).

This is why chart 02's Academic column prints Major and Minor without the
asterisk the other charts carry: no Scholar can lose the benefit for want
of a Major.

Implemented at `chargen/scholar.go` (`selectAreas`).

### I-24: "Research Success Major +2" awards two levels (p. 76 chart 02)

Table B reads "Per Term 4 / Promoted 1 / Research Success Major +2". The
first two rows are counts of table C rolls, which invites reading the third
as two rolls restricted to the Major.

Implemented as two levels of the Major, awarded directly. The row names the
Major rather than a table, and "+2" is the notation Book 1 uses for a level
award throughout the charts ("Str +1", "Skill-4"). Two restricted rolls
would be written as a count, like the rows above it.

The Skill-15 cap (p. 134) absorbs the compounding over a long career.

Implemented at `chargen/scholar.go` (`researchAndPublication`).

### I-25: The Award-Winning publication margin (p. 76 chart 02)

Chart 02 reads: "If Publication Roll is 4 less than Characteristic, it is
<Award-Winning> and counts as TWO."

Read against the raw characteristic, not the modified target: the chart
says _Characteristic_, and the Publication throw's target is the
characteristic with the opposite-sign Caution or Bravery mod applied. A
Publication roll of Characteristic − 4 or lower is Award-Winning and adds
two publications.

The alternative — measuring against the modified target — would let a
Bravery mod manufacture awards, which the printed word does not support.

The margin is read off a roll that carried the Publication on its own. A
Caution Mod of +5 or more puts the Publication target below
Characteristic − 4, so a _rejected_ roll can sit inside the margin; a
rejection rescued by a Waiver is a plain Publication, since the chart
qualifies the award by the Publication Roll and that roll did not publish.

Implemented at `chargen/scholar.go` (`publish`).

### I-26: The Scholar rank gates read against current Edu (p. 76 chart 02)

Chart 02 gates the ladder on Edu: entry is automatic at Scholar1 for
Edu 8+, "A character with Edu 7 or less is an Amateur Scholar ... cannot be
Promoted", and Tenure needs Edu 10+. The Personal skills column awards
Edu +1, so a Scholar's Edu can cross either threshold mid-career.

The promotion and Tenure gates are tested against the Edu held at the time,
so an Amateur who studies his way to Edu 8 becomes promotable and climbs
from Scholar0 by the ordinary promotion roll. The entry rule is not
re-applied: "automatically Scholar1 to Begin" is scoped to Begin, so a
late-blooming Amateur is not retroactively made a Lecturer.

Implemented at `chargen/scholar.go` (`mayPromote`, `mayApplyForTenure`).

### I-27: The Noble Return and Intrigue modifiers (p. 85 chart 11)

Chart 11's Return & Intrigue box prints its rolls in two columns — "Roll
R&R CC +Mods" under Return, "Roll R&R CC +(opposite sign) Mods" under
Intrigue — and beneath them, one under each column, "Mod= -Successful
Intrigues." and "Mod= +Times Exiled." Whether those are two per-column
modifiers or one combined Mod is not stated.

The p. 73 Career Resolution checklist is the discriminator. It compresses
the same career to:

> NOBLE / To Begin is Automatic if Soc B+ / **Roll Return&Intrigue vs C2 C3
> C4 C5** / Mod minus Intrigues / Mod + Exiles / Determine Skill
> eligibility; take Skills / Roll 7 to Continue

Both modifier lines sit under a single roll line, so they are read as one
combined Mod. The evidence is suggestive rather than decisive: that line
names both rolls at once ("Return&Intrigue"), so a reader could equally
take the two Mod lines as one apiece. It is the best textual signal
available, and it is weighed against the escalation noted below. Implemented as Mod = −Successful Intrigues + Times Exiled, applied to
Return as printed and to Intrigue with the opposite sign:

- Return = CC − Successful Intrigues + Times Exiled
- Intrigue = CC + Successful Intrigues − Times Exiled

which reads sensibly in both directions: a practised schemer intrigues
more easily and is granted return less readily, while exile cuts the other
way — each banishment makes the next Return easier to be granted and
further Intrigue harder to work. (Both rolls are roll-low Checks, so a
higher target is the easier one.)

The reading compounds: Successful Intrigues are never spent, so the
Intrigue target climbs by one per success. A long Noble career reaches a
target above 12, where only the p. 134 automatic failure can stop it (the
pinned seed 3268 is at End+4 = 15 by its fifth term), and an exiled veteran
schemer faces a correspondingly sunken Return. Reading 1 below has no such
escalation; it is recorded here as the cost of the reading taken, not as a
reason to revisit it without a rules ruling.

Readings not taken:

1. Per-column modifiers with the opposite-sign instruction honoured:
   Return = CC − Intrigues, Intrigue = CC − Exiled.
2. Per-column modifiers with the instruction ignored: Return = CC −
   Intrigues, Intrigue = CC + Exiled. Rejected as semantically backwards —
   repeated exile would make further intrigue easier.

The Archive's preliminary Nobles sheet also lists both modifier lines
together under one "Return and Intrigue" row (locate-only; Book 1
governs).

Only one of the two rolls happens in a term, so the opposite-sign
scaffolding protects no in-term tradeoff here — which is why the box reads
ambiguously.

Implemented at `chargen/noble.go` (`nobleMods`).

### I-28: An unmet Noble prerequisite is not a failed attempt (p. 85 chart 11)

Chart 11 gives "To Begin Automatic* ... *if Soc B+" and prints no To Begin
throw. A character below Social Standing B therefore makes no attempt.

Implemented as no throw, no year, and a career_not_begun consequence. P. 65
charges a year for a _failed attempt_ — "Each failed attempt (both Begin or
Retry) takes one year" — and distinguishes attempts from prerequisites:
"Pre-Requisites. Some Careers have requirements before a character may
attempt to Begin." An unqualified character never attempts, so nothing
elapses.

Implemented at `chargen/noble.go` (`begin`).

### I-29: A shared Social Standing enters at the lower title (p. 85 chart 11; p. 51)

"Nobles begin with rank equal to their Social Standing" (p. 65), but three
Social Standings carry two titles each: Soc 12 is Baronet and Baron, 14 is
Viscount and Count, 15 is Duke twice.

A character beginning at such a value enters at the lower rung. P. 51 is
explicit for the first of them: "A character elevated to Soc = c (lower
case) is **initially** a Baronet. The next increase in Soc remains C (now
upper case) but the title increases to Baron." A character arriving at that
Social Standing has reached it, however it happened, so the initial title
is the one it confers.

This is what chart 11's Elevation clause allows for — "the next higher
Noble rank and its increase in Social Standing (**if any**)" — and the
title-only steps award no Land Grant under the box A rule the
implementation follows (see I-30 for the rank table's conflicting note).

At the p. 68 characteristic maximum the title still advances while the
Social Standing does not, and again no Land Grant follows.

Implemented at `chargen/noble.go` (`nobleRankFor`, `raiseSoc`).

### I-30: Land Grants follow Soc increases, not titles (p. 85 chart 11)

Chart 11 states the Land Grant rule twice, and the two statements disagree.
Box A: "Land Grants. Each increase in Soc during CharGen awards a Land
Grant." The note under the Noble Rank and Land Grants table: "Nobles
receive Land Grants associated with their fiefs. **Each noble title confers
a Land Grant.**"

They agree everywhere except the three rungs that share a Social Standing
(I-29). Baronet to Baron is a new title with no Soc increase: by box A no
grant, by the table note a grant. The same holds for Viscount to Count and
Duke to Duke, and for a title-only step taken at the p. 68 characteristic
maximum.

Box A governs here. It is the career box's own rule, phrased as a CharGen
procedure ("during CharGen") in the register the rest of box A uses, while
the table note sits in the descriptive paragraph about fiefs and hex
income, which is the muster-out material deferred to milestone 4. Reading
box A as the procedure and the note as the setting description keeps one
rule in force during generation.

Taken literally, box A also awards a Land Grant for _any_ in-career Soc
increase, not only Elevation's: chart 11 table C column 1 line 6 is "C6
+1", which raises a Noble's Soc during the career. That grant is awarded.
Soc increases before the career — homeworld skills, education — are not,
since the character was not yet a Noble and had no fief to be granted.

A Soc increase from table C does **not** move the character along the rank
ladder: "Nobles begin with rank equal to their Social Standing" (p. 65)
scopes the rank/Soc equality to career entry, and chart 11 gives Elevation
as the only route between rungs. A Noble may therefore hold Soc 13 as a
Knight, which is a consequence of the printed rules rather than a
deviation.

Implemented at `chargen/noble.go` (`raiseSoc`, `characteristicRaised`).

### I-31: Wound Badges do not modify promotion (p. 82 chart 08; p. 70)

Chart 08 stars its Officer Promotion and Enlisted Promotion rows and
explains them as "*+Medals and WB Mods". The Imperial Medals table says the
opposite: "Medals (but not Wound Badges) are Mods for Soldier / Spacer /
Marine Promotion" (p. 70).

The p. 70 footnote governs. The p. 66 worked example is the discriminator:
Eneri Dinsha holds one Exemplary Service and one Wound Badge when his first
promotion is computed as "Soc plus Medal Mods (10 **+1**) =11" — the XS
contributes its +1 and the Wound Badge nothing. His second term, holding
one XS, one MCUF and still one Wound Badge, reads "(10 +1+2)": again the
two medals only.

Residual: the worked example is an officer, so it discriminates the Officer
Promotion row directly and the Enlisted Promotion row only by extension of
the footnote's own wording, which names the promotion of all three
services without distinguishing the ladders.

Implemented at `chargen/armedforces.go` (`medalMod`), and in the data as
`medal_mods` on the two promotion rows.

### I-32: The Risk-success badge carries no promotion modifier (p. 82 chart 08)

Chart 08's Risk row awards on success an "XS Exemplary Service Badge", the
same code the Imperial Medals table gives for a Reward roll of 2 through 8.
Read literally, every unharmed term would also add +1 to later promotions.

The Medals table is indexed by "Rew= Successful unmodified **Reward** Roll"
and has no row for a Risk result, so a Risk success awards a decoration
outside the table. It is counted on the record and carries no modifier.
The p. 66 Marine example agrees in spirit, awarding a _Campaign Ribbon_ on
its Risk success rather than a table medal.

The alternative — treating the badge as a table XS worth +1 — is rejected
because it would make the promotion modifier a count of unharmed terms
rather than of decorations, and would let a character promote steadily
while never once succeeding at Reward.

Implemented at `chargen/armedforces.go` (`riskAndReward`), recorded as
`service_badges`.

### I-33: Branch and Operations rolls that run off their tables (p. 82 chart 08)

Chart 08 rolls 1D on an eight-row Branch table with "DM +2 if Edu 10+", and
1D on a nine-row Operations table with "1D+Branch DM plus +2 if Edu 10+".
The Technical branch's DM of +6 with the education modifier reaches 14
against nine rows.

A roll past the end of either table reads its last row; a roll below the
start reads its first. The tables are printed as complete lists with no
wrap or reroll instruction, and the modifiers are plainly meant to push a
character toward the later rows — for Operations, toward Base and its Mod
of zero, which is the quiet posting a highly educated technician would
draw.

Implemented at `career/career.go` (`BranchAt`, `OperationAt`).

### I-34: Branch changes (p. 82 chart 08; p. 66)

Chart 08 says "Officers may not change Branch; Enlisted may select new
Branch on Promotion", and chart 07 repeats it. The p. 66 prose says instead
that "A non-officer character may change (reselect or reroll) Branch at the
end of each Term."

Neither is implemented in v1: the engine sets a branch on entry and keeps
the row for the rest of the career. (Chart 07's Naval Branch table prints
an Officer and an Enlisted side per row, so a Spacer's commission still
moves him across his own row — "for Spacers, Crew becomes Line" (p. 66) —
but the row itself never changes and he is never offered the reroll the
same sentence allows.) The two texts disagree on when an enlisted character
may change, and nothing else in the career turns on the difference until
the branch-change choice exists. Spacer (chart 07) inherited the omission;
recorded here so a later Armed Forces career resolves it rather than
inheriting it silently again.

When it lands, the charts are the narrower and more specific statement and
agree with each other, which argues for "on Promotion"; the prose is the
general rule and argues for "at the end of each Term".

Not implemented; see COVERAGE.md.

### I-35: A Reward success awards only the table medal (p. 82 chart 08)

Chart 08's Risk & Reward grid gives the Reward-success cell as "XS Exemplary
Service, Medal", and the prose beside it as "Success: XS Exemplary Service
and consult Medals table". Read as a conjunction, every Reward success would
award an Exemplary Service _plus_ whatever the Medals table returns.

The Medals table itself is the discriminator: its lines 2 through 8 _are_
"XS Exemplary Service" (p. 70), so the cell names the common case and then
sends the reader to the table for the rest. The p. 66 worked example
confirms it — Eneri Dinsha succeeds at Reward twice, taking line 4 (XS) in
his first term and line 10 (MCUF) in his second, and his card reads
"MCUF-1. XS-1.": two Reward successes, two decorations, not four.

Implemented at `chargen/armedforces.go` (`riskAndReward` calls `awardMedal`
once on a Reward success).

### I-36: A branch is selected on the side the character will serve (p. 81 chart 07; p. 66)

The Naval Branch table prints two sides for each row — row 3 is Line for
an officer and Engineer for a rating, at Mods 1 and 0 (chart 07, p. 81).
A branch is determined on entry, and "Armed Forces characters begin with
enlisted rank" (p. 65), so at that moment every character is enlisted.

The selection therefore offers the enlisted side: an entering Spacer picks
among Crew, Engineer, Gunnery, Technical, and Medical, and the Mod he
weighs is the one his own Risk roll will carry.

Offering the officer side instead has a visible cost: the event log
records a character choosing "Line" and then serving as "Crew".

Either side deduplicates. The chart prints eight rows and fewer distinct
names, so a name has to bind to one row, and the rows it does not bind to
stay reachable only by the roll. On the officer side, rows 2 and 3 (both
Line) collapse into row 1; on the enlisted side, rows 2 (Crew), 4
(Engineer), and 6 (Gunnery) collapse into rows 1, 3, and 5. Neither scheme
escapes that, so which row a name binds to is a decision, not a by-product:

**A name binds to the row that reads the same on both sides.** Enlisted
Engineer appears on row 3, whose officer side is Line, and on row 4, whose
officer side is Engineer; the name binds to row 4. Enlisted Gunnery
appears on rows 5 (officer Gunnery) and 6 (officer Flight); it binds to
row 5. A character who selects a branch and then keeps it — p. 66 lets a
commissioned character "roll for Branch or keep his current Branch" — is
therefore still in the branch he selected, at the Mod he weighed on entry.
Binding to the mixed row instead would quietly move a rating who chose
Engineer at Mod 0 into Line at Mod 1: a Mod he never chose, and, under the
`select_branch` policy, the opposite of the one he was picked for. Crew is
the exception the chart forces — no row reads Crew on both sides, so it
binds to row 1 and a commission does turn Crew into Line, which is exactly
the case p. 66 names.

The rows themselves are unaffected: a commission moves the character
across the row he already holds, which is what "for Spacers, Crew becomes
Line" (p. 66) describes. The other half of that sentence — the option to
reroll Branch rather than keep it — is deferred under I-34.

Implemented at `chargen/armedforces.go` (`chooseBranch`, `sameOnBothSides`).

### I-37: The Peacekeeper column and the Peace Keeper assignment (pp. 82, 86)

Charts 08 and 12 print the skills column as "Peacekeeper" and the
Operations row as "Peace Keeper". Both are transcribed as printed, and the
operation names the column it opens.

This matters because the p. 65 restriction matches assignments to columns
by name: transcribing the column as "Peace Keeper" to make the two agree
would have been a silent edit of the chart, and transcribing them
faithfully without the mapping would have left the column unreachable and
the term quietly short of eligibility.

Implemented at `career/data/soldier.json` and `marine.json`
(`operations[].column`), enforced by the loader.

### I-38: What an Agent may select from a cover career's table (p. 83 chart 09)

Chart 09 sends an Agent undercover and says: "Select (not Roll) one skill
from the skill tables of that Career." Table C of a career is not a list of
skills, though — its cells also raise characteristics, name the character's
Major and Minor, and stand in for whole Master Skill List groups.

The alternatives offered are that career's table C flattened in chart
order, without repeats, and read as follows:

- Plain skill cells are offered as printed.
- Group cells — One Art, One Trade, One Science, Starship Skill, Soldier
  Skill — expand to their Master Skill List members. "Select (not Roll)"
  replaces the 1D that would otherwise have chosen the cell, and the same
  parenthetical replaces the follow-on selection the group cell would have
  triggered.
- Characteristic cells are excluded: they are not skills, and an Agent
  gathering information undercover is not living the cover career's life.
- Major and Minor cells are excluded: they name the _Agent's_ areas, not
  the cover career's, so they offer nothing the cover identity teaches.
- Chart 13's "Any Skill*** from Citizen Life Skills and Knowledges" and
  chart 09's own "Any Knowledge" expand to those lists.

The skill is awarded at one level through the ordinary award path. The
Agent does not enter the cover career: no rank, no automatic skills, and no
first-receipt accounting against it.

Implemented at `chargen/agent.go` (`undercoverSkills`).

### I-39: The Citizen rows roll rather than select (p. 83 chart 09)

Two rows of the Undercover Assignment table print, in place of titles,
"Roll on Citizen Life Skills for Job" and "... for Hobby". These name
chart 04's table E and the roll that reads it, not the Agent's own Job and
Hobby fields, which stay empty: the Agent is undercover, not a Citizen.

The rolled skill is awarded at one level. Chart 04 would award a Job at
Skill-4 and a Hobby at Skill-2, but those are a Citizen's own first
receipts; chart 09's table B allows the Agent "Undercover 1". A "No Skill"
cell awards nothing, as it does for a Citizen.

The chart's own C column is not rolled for these rows, which is the
"if required" of "finally top row C (reroll if >3) if required": a row
offering one entry needs no C roll, and the Functionary row, which offers
none, needs no title at all.

Implemented at `chargen/agent.go` (`undercover`, `undercoverJobTable`).

### I-40: Functionary is transcribed as a reference career (p. 87 chart 13)

The Undercover Assignment table's last row sends an Agent undercover as a
Functionary, whose career chart 13 says "is never a first career" — so the
engine cannot run it until career changes land (docs/PRD.md milestone 4),
and one assignment in eighteen had nowhere to read its skills from.

Chart 13's table C is therefore transcribed as a _reference_ career: loaded
and validated like any other, absent from the available careers and from
the mechanics registry, and read only by chart 09's table. The alternative
— recording the assignment and awarding nothing — would have quietly
starved a row the book prints as playable.

Its box A fields are not transcribed. Chart 13's Continue is "Office
Politics", a form the engine has no target for, and nothing about the
Agent's use of the chart needs it.

Implemented at `career/data/functionary.json` (`reference: true`).

### I-41: A Rogue's Risk failure imprisons rather than injures (p. 84 chart 10)

Every other Risk & Reward chart prints the same Risk-failure text —
"Reduce CC by negative Mods and Flux ... If reduced by 4 or more, then he
is disabled." Chart 10 prints something else entirely: "Prison for (sum of
negative Mods + Flux) years at the start of the next Term (may be zero;
maximum 4). Fame +1 (actually Infamy). Payoff (if any) is halved", and its
Risk success reads "Unharmed" rather than the usual badge.

A Rogue therefore takes no characteristic reduction, no Wound Badge, and
cannot be disabled or killed by his own career. The consequence of being
caught is a sentence, not a wound.

The Flux the sentence names is its own roll: chart 10's Risk failure has
no injury to compute, so the Flux belongs to the sentence. Years are the
negative of the sum, floored at zero and capped at the printed maximum of
four — which is what "may be zero" allows for, a Flux positive enough to
offset the negative Mods leaving the Rogue at liberty.

Implemented at `chargen/rogue.go` (`imprison`).

### I-42: The Rogue's controlling characteristic is chosen before To Begin (p. 84 chart 10)

Chart 10 gives "To Begin CC", and separately "A Rogue selects one
Controlling Characteristic (C1 C2 C3 C4 C5 C6) which is then used
throughout his career (not just in the current Term)." The To Begin check
therefore has no target until the selection is made, so the selection
comes first and the check tests it.

The choice is presented once. Later terms reuse it with no further choice
event, and it is also the Continue target — chart 10's "Continue CC*".

Implemented at `chargen/rogue.go` (`begin`) and `careerRun.chooseCC` via
the `cc_fixed` flag.

### I-43: One term serves a prison sentence (p. 84 chart 10)

A sentence is "Prison for ... years at the start of the next Term (may be
zero; maximum 4)", and a term is four years. A sentence of one to four
years is served by one term, which is spent entirely in prison: the Rogue
masterminds no Scheme and rolls no Risk or Reward, and takes the two
Prison Skills table B allows, "from the Rogue Skills table column 1 or 2
only ... (not Term or Scheme Skills)".

The alternative — serving a one-year sentence and then scheming for the
remaining three years of the same term — is rejected because the chart
prints one Scheme per Term and one skills allowance per Term, with no way
to apportion either across a partial year.

The Continue throw still happens at the end of a prison term. Nothing
suspends it, and a Rogue may leave the career from prison.

Implemented at `chargen/rogue.go` (`prisonTerm`).

### I-44: The Payoff formula (p. 84 chart 10)

Chart 10 gives "Payoff= V x (1+CC-R+Mods)", where "V= Value of Scheme,
CC= Controlling Characteristic, R= Reward Die Roll, Mods= Mods for
Reward".

Read as: V is the scheme row's printed value, CC the characteristic's
current value, R the raw total of the Reward throw, and Mods the
Reward-side modifiers with the sign they were applied — the opposite of
the Risk side, as the chart's own "Roll R&R CC +(opposite sign) Mods"
instructs.

A multiplier at or below zero pays nothing rather than costing the Rogue
money: "No Reward" is what the chart prints for a failure, so a success
cannot be worse than one.

"Payoff (if any) is halved" where the Risk also failed, by integer
division. The two Ship Share rows multiply and halve their shares the same
way, since the chart gives them as Values like any other.

Implemented at `chargen/rogue.go` (`payoff`).

### I-45: A term ending in disability still elapses its four years (p. 66; chart 05 p. 79)

Book 1 fixes the term at four years — "the 4-year Term" (p. 66) — and
tells a disabled character to "Muster Out at Term end with Double
Benefits" (p. 69; charts 02, 05, 06, 07, 08, 09, 12).

Read as: the term completes. "Term end" is the printed moment the
character musters out at, so the term he musters out of is a term he
served, and it costs the four years every term costs.

The alternative — that the career stops at the injury and the remaining
years are never served — would make a disabled character younger at
muster out than a healthy one who served the same number of terms, and
would leave "Term end" naming a moment that never arrives.

This is recorded because it changes generated output. The engine advanced
the clock inside the Continue throw, which a disabled character never
reaches, so such a term silently elapsed no time at all.

Implemented at `chargen/careerrun.go` (`term`).

### I-46: A term ending in death still elapses its four years (p. 66; p. 65; p. 69)

"If the Controlling Characteristic is reduced to zero or less, the
Character is dead" (p. 65). Book 1's dedicated paragraph on the subject,
"Dying During Character Generation" (p. 69), adds only that "all efforts
in this particular character creation process are lost" — it fixes no age
for the death, and the engine's finest unit of time is the year.

Read as: the term elapses in full, for the same reason as I-45 — the term
is T5's atom of career time, and the engine has no sub-term clock to
report a death partway through one.

The rival reading is defensible and is named here rather than dismissed:
a character who dies did not finish the term, so the term might cost
nothing, and p. 69's "all efforts ... are lost" can be read as saying the
record is discarded and its age never asked for. It is rejected because
it reports a character who dies in his first term as dying at 18 — the
instant he entered the career — which is a worse answer than rounding to
the term the book already treats as indivisible.

Implemented at `chargen/careerrun.go` (`term`).

### I-47: Sanity is recorded as a pending modifier, never generated (p. 52; chart 05 p. 79)

Two pages govern, and they pull in different directions.

Page 52 establishes the characteristic and withholds it: "Every character
has this obscure (and usually unreferenced) characteristic called Sanity.
Characters do not generate Sanity until it is first called for by a
situation, encounter, or stimulus." Generation is "All sophonts roll
Sanity with 2D", and recording is deliberately absent — "Sanity is not
normally indicated in references to a character ... When necessary, it is
stated independently as CS= N or San= N."

Chart 05 then spends it: "Because of the long-term isolation that a Scout
must endure, reduce San= -1 for each TWO Terms served" (p. 79). It is the
only career chart that prints such a rule, and it presupposes a value that
p. 52 says does not yet exist.

Read as: the reduction is recorded, not applied. The record carries
`sanity_mod` — what a Sanity value will owe the moment something calls for
one — and no Sanity value is generated.

The rival reading is to roll 2D at the moment chart 05 demands it and
apply the reduction immediately. It is rejected on both pages: p. 52 says
plainly that generation waits for a situation, and character generation is
not one, since nothing in the lifepath ever reads Sanity. It would also
consume two faces of the seeded stream for a value v1 never uses, moving
every downstream throw for no outcome.

Partial terms owe nothing. The chart charges per two terms, so three terms
cost what two cost.

**Open where career changes make it reachable (milestone 4).** "for each
TWO Terms served" is charged here against the terms of a single career
record. Should a character serve as a Scout, leave, and return, three
terms followed by one would owe -1 under this reading but -2 if the two
stints were summed first. Nothing in the book settles it, and until career
changes land no character can serve two Scout stints.

The genetic-die rule — "Sanity Is Genetic. The first die of Sanity is
recorded as the genetic D" (p. 52) — is out of reach for the same reason:
there is no roll from which to take a first die.

Implemented at `chargen/careerrun.go` (`recordSanityMod`), with the count
of terms per point as a chart fact in `career/data/scout.json`.

### I-48: The Retirement stage runs to 73, not the printed 71 (p. 89 chart A)

Chart A states its structure three times, and one statement disagrees with
the other two.

The prose: "After Infancy, each Life Stage is two terms (8 years; this may
differ for non-humans)." The lifespan: "Humans have a 2-year infancy and
nine stages of 8 years each. The traditional lifespan for humans is 74
years." Both put Retirement, stage 9, at 66-73. The Stages of Life table
prints "9 Retirement 66-71".

Read as: the arithmetic governs. Two statements agree against one, the
lifespan of 74 is arithmetically impossible under the printed range (2 +
8×8 + 6 = 72), and every other row of the table matches the arithmetic
exactly.

Nothing mechanical turns on it. Retirement is the last stage, so a
character past 71 stays in it either way, and the Aging Check reads the
stage number rather than the range. It is recorded because the table is
transcribed as printed, so a reader comparing the data to the derivation
will find the divergence and should find this entry with it.
`TestPrintedRangesDivergeExactlyOnce` pins the divergence set to this one
row, so a later transcription cannot quietly resolve it or add another.

Implemented at `lifestage/lifestage.go` (`Of`, `FirstYearOf`), with the
printed columns kept in `lifestage/data/lifestages.json`.

### I-49: Three characteristics at zero means three or more (p. 89 chart A)

The chart escalates by count: one characteristic reduced to zero is reset
to 1; two bring "a major illness ... four weeks in rest and recuperation";
three bring "an extremely major illness ... four months", and "the second
time three characteristics are reduced to 0, the character dies". It stops
at three.

Four is reachable. Mental Aging adds Intelligence to the three Physical
characteristics at Life Stage 9, so a single Aging Check pass rolls four,
and all four can zero at once.

Read as: three or more. The chart's sequence is plainly an escalation, and
reading "three" strictly would make four characteristics at zero less
severe than three — indeed harmless, since no clause would match.

### I-50: Aging Checks fall on absolute ages, and illnesses cost no game years (p. 89 chart A)

"Once Aging begins, it occurs every four years on the character's
birthday."

This entry first justified itself on the claim that Book 1 prints no
birthdate rule and FR8's only cite is the Archive the ground rules exclude.
**That claim was false.** Book 1 prints the rule twice — p. 58 ("Date of
Birth") sets the default current date at 001-1105 and says to subtract age
from it, and p. 263 ("Birthdates") gives the Birth Date Generation table
that fixes the day of the year. FR8 was miscited, which is not the same
thing as unsourced, and the sweep that reported no rule missed both pages.

The reading survives its own bad premise, on the second argument, which was
always the load-bearing one. Read as: the checks fall at ages 34, 38, 42 and
so on — the four-year cadence anchored to the age Physical Aging begins at.
Anchoring to absolute ages rather than to elapsed time matters because a
failed career entry costs a single year (p. 65), which would otherwise knock
a character permanently off the four-year grid and change how many checks a
lifetime holds.

A birthdate does not disturb that. p. 58 puts the calculation at the end of
character generation — "Until Character Generation is complete, Birthdate
calculation may be deferred" — so the day of the year is known only after
the last Aging Check has already been thrown, and could not have scheduled
them even if the engine had wanted it to.

Neither illness costs game years. Four weeks and four months are both
shorter than the year, which is the finest unit the engine tracks. They
are recorded on the character and in the transcript, and charged nothing.

Implemented at `chargen/aging.go` (`ageEffects`, `illness`).

### I-51: Death ends career resolution (p. 89 chart A; p. 65; p. 69)

Both deaths in character generation — a controlling characteristic reduced
to zero by injury (p. 65), and the second extremely major illness (p. 89) —
now stop the lifepath.

This is recorded because the engine did not do it before, and the omission
was invisible until aging made it load-bearing. Aging kills, but
generation went on serving terms afterwards: sweeps produced characters
who died in their nineties and were still accumulating skills at 401. The
engine has always set the dead flag and carried on.

The record is still returned rather than discarded, which is the narrower
reading of p. 69's "the Character is dead (and all efforts in this
particular character creation process are lost)". A generator that reports
how a character died is more useful than one that returns nothing, and the
sentence is as easily read as guidance for play as an instruction to the
tool. That question stays open in COVERAGE.

Implemented at `chargen/careerrun.go` (`term`) and `chargen/character.go`
(`runCareer`).

### I-52: A character killed by aging may record an age past the year he died (p. 89 chart A; p. 66)

The term is four years (p. 66) and the engine elapses them together, then
resolves the Aging Checks the span crossed. A character who dies on a
birthday inside the term therefore keeps the whole term's years: he can
die at 86 and end the record at 87.

Read as: the term still costs its four years, consistent with I-45 and
I-46. The alternative — truncating the clock at the fatal birthday — is
more accurate about the age and was measured before being rejected: 203 of
531 aging deaths over 3000 seeds overstate the age by one to three years.
It is rejected because it would move the years-elapsed event after the
Aging Checks it precedes, renumbering every subsequent event in every
record for a cosmetic gain, and because the engine already charges whole
terms for deaths within them.

What the record does instead is state the year plainly. Aging is the one
death in character generation whose year the rules pin exactly — an injury
can fall anywhere in a term, an Aging Check falls on a named birthday — so
the death consequence carries the age at death, and the transcript prints
"DEAD at 86" rather than leaving a reader to infer it from a `age` field
that says 87.

Implemented at `chargen/aging.go` (`illness`) and `render/render.go`
(`consequenceDeadText`).

### I-53: A failed Continue ends Career Resolution; only a voluntary change chains careers (p. 66)

Two sentences on the same page decide how many careers a character can
hold. "At the end of the 4-year Term, the Character must successfully roll
(2D) to Continue (or less) in the career. Failure ends Career Resolution:
the character must begin adventuring." And: "A character may avoid the
Continue roll (and its possibility of Mandatory Continue) by voluntarily
ending his service in the current career and selecting a different career
for which he is eligible."

Read as: a failed Continue ends everything, and the voluntary change is
the only way to a second career. The rival reading — that a failed
Continue merely ends _that_ career and another may be selected — is
rejected by the first sentence naming Career Resolution rather than the
career, and because it would make the voluntary change pointless: the
change exists precisely to avoid the Continue roll, and there would be
nothing to avoid if failing it cost the same.

**No cap on the number of careers is stated anywhere.** Book 1 prints
none, and none is invented here. The practical bound is the one the rules
supply: each new career must pass a To Begin, aging accumulates, and death
ends resolution.

### I-54: A career left may be re-entered; a career failed may not (p. 66; p. 65)

"a different career for which he is eligible" (p. 66) is read as different
from the current one, which is what the sentence is distinguishing. Nothing
forbids returning to a career served earlier, and chart 10's "select any
previous career" for a Rogue's Scheme presupposes that careers accumulate.
A character may therefore serve, leave, and later return.

A career whose To Begin failed is different: "If both Begin and Retry
fail, this career may not be used" (p. 65). That is read as lifetime
rather than scoped to the selection round in progress, since the sentence
scopes itself to the career rather than to the attempt. The engine reads
it off the record — a career holding a `began:false` entry is excluded —
so the exclusion survives a career change without separate bookkeeping.

### I-55: Resigning from the Reserves is deferred (p. 67)

"A character who leaves a military, naval, or marine career is
automatically in the Reserves until retirement at Life Stage 9, at which
point he or she receives a Reserve Pension. A character in the Reserves
maintains his or her last held rank as a Reserve Rank." That much is
recorded: which careers enrol a leaver is a chart fact in the career data,
and the rank is the last one held, there being "no process for promotion
or advancement while in the Reserves".

"A character may resign from the Reserves (Check Continue) and forego its
benefits and responsibilities" is deferred to interactive mode. The
default policy would never resign — resigning forfeits a pension and
gains nothing the engine models — so the Check would consume two faces of
the seeded stream in every Armed Forces character and shift every
subsequent throw for an outcome that never varies. It becomes a real
decision when a player is answering.

Activation — "A member of the Reserves is subject to activation for the
needs of the service" — is play, not generation.

### I-56: "Continue Office Politics" is not a throw (p. 87 chart 13)

Box A prints three lines: "To Begin Total Terms x3 / Office Politics C2 C3
C4 C5 / Continue Office Politics". Every other career's Continue line names
a target — a number, a characteristic, a tracked value. Chart 13's names a
procedure.

Read as: the Functionary rolls no Continue throw. The Office Politics box
already decides it, in the terms the page uses for the Risk: "Risk Failure:
Functionary career ends. The character may not Continue. Risk Success:
Functionary may continue in the career." A separate Continue throw would
be a second chance at a question the Risk has answered, and the box's
Failure column says "Cannot Continue" rather than "career ends", which is
the language of a Continue outcome.

This is recorded because the engine's Continue machinery assumes a target,
and a Functionary reaching it with none would throw against zero and end
every career after one term. The definition therefore declares Office
Politics as its Continue form, and the term loop skips the throw.

### I-57: The Auto Skill column pairs with F0, F2, and F3 (p. 87 chart 13)

The Functionary Ranks table prints a rank ladder beside an Auto Skill
column with three entries: Bureaucrat, Admin, Bureaucrat. Which ranks they
belong to is a matter of vertical alignment, and the table's row spacing is
uneven — text extraction in reading order pairs the second Bureaucrat with
F3, and a layout-preserving extraction places it below F4.

Settled from the glyph coordinates rather than by eye or by pattern. Each
Auto Skill shares an exact baseline with its rank: F0 Clerk and Bureaucrat
at y=290.511, F2 Senior Supervisor and Admin at y=308.646, F3 Manager and
Bureaucrat at y=322.911. F4 Senior Manager sits at y=330.246 with nothing
beside it.

Worth recording because the result is not the pattern a reader expects. F0,
F2, F4 — every other rank — would be the natural reading, and it is wrong.

### I-58: A Noble cannot become a Functionary, by two rules that agree (p. 87 chart 13; p. 66)

Chart 13 states it directly: "Note that a Noble may not become a
Functionary." p. 66 states it more broadly: "A Functionary or Noble cannot
change to a new career."

The engine enforces the broad rule, which subsumes the narrow one — a
Noble cannot leave for any career, so he cannot leave for this one. No
separate Noble-to-Functionary check exists, and a test holds the two
together so that relaxing the broad rule cannot silently repeal the narrow
one.

The redundancy is worth noting rather than resolving: chart 13 says it
because a reader on that page has no reason to have read p. 66.

### I-59: A failed Risk does not stop the Reward roll (p. 66; p. 87 chart 13)

Chart 13's Office Politics box prints six lines, and run together they read
as though the Reward were nested inside a Risk success:

    Roll for Risk against CC. No Mods are used for Office Politics.
    Risk Failure: Functionary career ends. The character may not Continue.
    Risk Success: Functionary may continue in the career.
    Roll for Reward against CC
    Reward Failure: Functionary is not promoted.
    Reward Success: Functionary is promoted one rank.

Read as: Risk and Reward are two rolls, both made every term, as the box's
own two-row table shows — one row for Risk with its Failure and Success
columns, one for Reward with its own.

The discriminator is the worked example on p. 66, where a failed Risk is
followed by a Reward roll anyway: "Risk must roll (10 -2 -1 +2) =9 or less
on 2D; he rolls 11 and fails ... Reward changes the sign on the Mods and
must roll (10 +2 +1 -2) = 11 or less on 2D; he rolls 9 and succeeds again.
He will receive a Medal." That is the generic Risk & Reward procedure every
career shares, and chart 13 prints its Office Politics in the same
two-row form.

The consequence is worth stating plainly, because it looks wrong at first:
a Functionary can be **promoted in the term he loses his job**. The Risk
asks whether he keeps the position and the Reward whether he did well, and
office politics is exactly the setting where both can be true at once — as
the Armed Forces version already is, where Eneri is wounded and decorated
in the same term. The rank is not cosmetic: muster out reads it ("Automatic:
Directorship if Rank F6+", chart 13 D).

Implemented at `chargen/functionary.go` (`resolveTerm`), whose doc comment
keeps the page's line breaks so the quote cannot be misread back into the
nested form.

### I-60: An automatic-but-conditional entry gates eligibility, it does not fail (p. 75 chart 01; p. 65)

Chart 01's box A reads "To Begin Automatic\*", footnoted "\*if TWO skill-6
and Craftsman-1". Entry is automatic — there is no throw printed — but only
for a character who already qualifies.

Read as: a character who does not qualify is not offered the career. p. 65
attaches its two costs to attempts — "Each failed attempt (both Begin or
Retry) takes one year" and "If both Begin and Retry fail, this career may
not be used" — and an automatic entry has no attempt to fail. Treating the
unmet condition as a failed To Begin would charge a year for a throw never
made, and would burn the career for life on a condition the character may
satisfy two terms later.

The practical effect is that Craftsman is unreachable at eighteen without
being barred: a character leaving education has neither the craft nor two
skills at level 6, and acquires both only over several terms in another
career. That is a different mechanism from chart 13's, which states the
bar as a rule ("Functionary is never a first career"), and chart 01 states
no such rule. p. 63 mentions both together — "Craftsman (1) and Functionary
(13) are unavailable as initial careers" — but scopes that to the 2D
random-selection system, where 1 and 13 are simply unrollable.

Implemented at `chargen/craftsman.go` (`meetsPrerequisite`) and
`chargen/careerrun.go` (`eligibleCareers`).

### I-61: Master Points count all four Controlling Characteristics (p. 75 chart 01)

The Creating A Masterpiece box lists what a Craftsman totals:

    Controlling Characteristics
    Craftsman Skill
    Up to FIVE Skills at level 6+ (or Knowledges at level-6) (but not
    languages)
    Must total at least 40 Master Points

"Controlling Characteristics" is plural, and box A names four of them —
"Masterpiece C1 C2 C3 C4". But the Passion text says "The Controlling
Characteristic governs creating the current Masterpiece", singular, and the
career rotates them.

Read as: all four are counted, and the rotating one governs which
characteristic the term's Masterpiece is made under. The arithmetic decides
it. A character entering on the chart's own minimum — Craftsman-1 and two
skills at 6 — totals about 7 + 1 + 12 = 20 under the singular reading, half
the 40 the box demands, so no new Craftsman could ever attempt a Masterpiece
and the career would be inert. Under the plural reading the same character
totals about 28 + 1 + 12 = 41, just over the bar. The entry condition and
the creation floor were plainly written to meet.

"Up to FIVE" is read as the best five available, which is what any Craftsman
would choose. Craftsman itself is counted on its own line and not again
among them.

Implemented at `chargen/craftsman.go` (`masterPoints`).

### I-62: The creation throw is roll-low inclusive, and QREBS is deferred (p. 75 chart 01)

The page states the throw twice and the two disagree by one. The box prints
"9D < Master Points"; the prose prints "Roll 9D for Master Points or less
for success in creation".

Read as: the prose, "or less". It is the roll-low Check form every other
throw in character generation uses (interpretation I-17), and a strict
less-than would be the second in the whole book after the Aging Check
(I-50), introduced here by a glyph rather than by words. The p. 134
automatic failure applies as usual, which matters: a Craftsman with 54 or
more Master Points would otherwise be unable to fail.

**QREBS is deferred.** "Allocate the Master Points to QREBS (for the ranges
-5 to +5, -5 = 1 point; +5 = 11 points). If all QREBS values are set at the
Maximum, excess Master Points can be allocated equally in excess of +5."
Nothing in character generation reads the result: chart F counts
Masterpieces and Perfect Masterpieces for Fame, and muster out prices them
by Master Points. The allocation is a decision about an object, made when
the object matters, so the five qualities and their ranges are transcribed
— Quality on 1 to 10, Reliability, Ease, Bulk and Safety on -5 to +5, read
from the column positions — and the allocation is left to play.

Vintage appreciation is deferred with it: "A Masterpiece increases in value
about 1% per year (simple interest), but are subject to Flux (in percent)
when sold" prices a sale, and character generation makes none.

### I-63: Fame stacks to 20 as a cap, not a cliff (p. 91 chart F)

"Fame Stacks. A character's Fame is the sum of all Fame points received to
20; beyond 20, only the highest Fame applies."

Read as: the sum applies up to 20, and a single accomplishment worth more
than 20 carries past it. So Fame is the greater of the capped sum and the
largest single source.

The rival reading takes "beyond 20" to describe the total: once the sum
passes 20 it collapses to the largest single source. The discriminating
case is two accomplishments of 12. This reading gives 20; the rival gives
12 — less Fame than the same character had after the first of them alone.
No reading of a rule titled "Fame Stacks" should let an accomplishment
subtract, and `TestStackIsMonotonic` holds that property over the whole
range rather than trusting the argument.

**What "the highest Fame" is a unit of.** The clause names a largest
single source, and the chart offers two candidates for what a source is:
one eligibility line's total, or one occurrence within it.

Read as: the occurrence. The chart defines the unit itself — "xN = N Fame
points per occurrence" — so a Scout's six Discoveries are six Fame points
of 4, not one of 24, and "the sum of all Fame points received" sums those.
The eligibility rows that instead name a value outright — "=Rank",
"=Publications", "Soc x1.5", the Entertainer's tracked Fame — each supply
a single point of that size, since there are no occurrences to divide them
into.

The rival reading takes the whole line as the unit, and the discriminating
case is a Rogue with twelve failed schemes: 36 under the rival, 20 under
this one. Two things decide it. The footnote defines the unit in so many
words, and it is the only sentence on the page that does. And under the
rival the "to 20" limit almost never binds — any xN line with a handful of
occurrences clears it — which leaves half the sentence idle and makes a
Rogue famous throughout All Reality for a dozen botched heists.

Under this reading both clauses have work. Sweeping the auto policy across
every first career and 800 seeds, 49 characters pass 20 and the highest
reaches 40, all of them on a single direct-value line — chart 03's
Entertainer, whose Fame the career tracks and chart F takes whole. The
limit binds on the xN lines; the escape carries the outright values.

**Where the Fame Flux Event applies.** "Any character may choose ... to
add Flux to Fame." The Flux is added to the Fame the accomplishments stack
to, not stacked alongside them as another source. Stacking it would put it
under the "only the highest Fame applies" clause, where a negative Flux is
absorbed whenever one eligibility dominates the total — a Scout with a
single 16-point source keeps 16 whatever he rolls — and a symmetric gamble
would only ever pay. `TestFameFluxCanLose` holds the property.

Implemented at `fame/fame.go` (`Stack`) and `chargen/fame.go`
(`computeFame`).

### I-64: "Merchant Ship Owner = 1D" is deferred (p. 91 chart F; p. 68)

Chart F gives a Merchant Fame for his rank and again for owning a ship.
Ownership is not established during career resolution: a Merchant
accumulates Ship Shares, and what they buy is settled at muster out, where
chart S (p. 90) prices ships in shares.

The ordering forecloses it. Muster out reads Fame — "He is allowed one
additional roll if Fame 19+" (p. 68) — so Fame must be known before muster
out runs, and ownership is not known until it has. Rolling 1D for a
Merchant who merely holds shares would credit Fame for a ship he may never
own.

Deferred rather than guessed. The eligibility is recorded here so that
whoever implements ship purchase knows Fame is waiting on it.

### I-65: "Armed Forces Enlisted = no Fame" is read flat (p. 91 chart F)

Chart F's Armed Forces block lists "Army / Marine / Navy: Officer Rank *"
above six decorations with their multipliers, and footnotes the block
"*Armed Forces Enlisted = no Fame."

Read as: an enlisted character earns no Fame from that career at all —
neither from rank, which he has none of, nor from his decorations.

The rival reading is defensible and turns on where the asterisk sits: it
is printed on the three "Officer Rank *" lines and nowhere else, so it
could be scoping only those, leaving the medal lines to apply to anyone.
That reading has something going for it — the chart's own Marine Sergeant
Brett Bozeman, with Wound Badge-4 after four terms, is memorable enough
that a reader expects him to be known for it.

It is rejected because the footnote says what it says. "Enlisted = no
Fame" is a flat statement, and reading it as narrower than its own words,
on the strength of a typographical detail, is the kind of inference this
file exists to avoid making silently.

Implemented at `chargen/fame.go` (`armedForcesFame`).

### I-66: "Imperial Noble Soc x1.5" rounds down (p. 91 chart F)

Fame points are whole numbers everywhere else on the chart — "xN = N Fame
points per occurrence" — and this is the only eligibility that can produce
a half. An odd Social Standing gives one: Soc 11 yields 16.5.

Read as: round down, which is how every other division in the rules
resolves (the muster-out DMs "+Fame/3" and "+Fame/2", the Scout's Sanity
"per TWO Terms"). Nothing on the page says otherwise, and rounding up
would make an odd Soc worth more per point than an even one.

Implemented at `chargen/fame.go` (`nobleFame`).

### I-67: A Rogue's failed scheme is a failed Reward, and his infamy is separate (p. 91 chart F; p. 84 chart 10)

Chart F prices "Rogue Successful Schemes x2" and "Rogue Failed Schemes
x3", and chart 10 has two rolls that could decide which a scheme was: the
Risk, whose failure imprisons, and the Reward, whose failure pays nothing.

Read as: the Reward decides. Chart 10's own skill eligibility table uses
exactly these words — "Failed Scheme 1 / Successful Scheme 4" — for the
Reward outcome, and reading the same two terms differently on the facing
page would be perverse.

That leaves chart 10's "Fame +1 (actually Infamy)", which it awards on a
Risk failure and chart F does not enumerate. It is recorded as the career's
own tracked Fame — the same field chart 03 uses, and which chart F reads
where it says "Entertainer detailed under Career" — and chart F adds it to
the total. The two are not the same event: a Rogue can be caught on a
scheme that still pays, and walk free from one that pays nothing.

Implemented at `chargen/rogue.go` (`schemeTerm`, `imprison`) and
`chargen/fame.go` (`rogueFame`).

### I-68: A career's own Fame contributes nothing when it is negative (p. 91 chart F; p. 77 chart 03)

Chart F reads the Entertainer's Fame off his career — "Entertainer detailed
under Career" — and chart 03 keeps that Fame as a running level: 2D at the
start, then "+F +F\* +F\*" every term, and 2D again on a Comeback. Flux is
1D-1D, so the level can end below zero. Seed 144 forced to Entertainer ends
at Fame -2.

Read as: the career contributes nothing. Chart F's eligibility column
prices accomplishments, and an Entertainer nobody has heard of has none;
"Fame is the level of recognition or respect society ... holds for an
individual", and no level of it is less than none.

The rival reading takes the number as printed and subtracts it, which
credits an obscure career with unmaking the Fame of every other one — a
Scout's Discoveries erased by an Entertainer's bad run. It also silently
suppresses "If NO other eligibility, 1D", because a negative entry is an
entry: seed 144 finished at Fame 0, Unknown, with no Fame line on the
sheet, where a career worth exactly zero would have rolled 1D.

The clamp is at the chart F reading, not on chart 03's own value: the
Entertainer's record keeps the negative level, which is his Continue
target and the number a Comeback measures against.

Implemented at `chargen/fame.go` (`singleLineFame`).

### I-69: "Merchant =Rank" reads the printed rank number (p. 91 chart F; p. 80 chart 06)

Chart 06 prints the Merchant ladder as RX Temp, R0 Spacehand, R1 Steward
Apprentice, R2 Drive Helper, then M1 Fourth Officer through M6 Senior
Captain, and notes "R-Ranks are Ratings (or Enlisted). M-Ranks are
Officers." The numbering restarts at the commission.

Read as: "=Rank" is the number printed beside the title, so a Drive Helper
is Rank 2 and a Fourth Officer is Rank 1.

The consequence is worth stating plainly: a Rating who takes his Officer
Commission loses Fame by it, R2's two points becoming M1's one. That is
what the chart says, and no other numbering is printed for the ladder —
chart 06's own muster-out DM says "+ Officer Rank", which likewise counts
the M-number. Recorded here because a non-monotonic Fame looks like a bug
at the implementation site.

Implemented at `chargen/fame.go` (`rankNumber`).

### I-70: Charts 12 and 01 print "Directorate" and "Director" where the rest print "Directorship" (pp. 86, 75; p. 71)

Chart 12's table D row 9 awards a "Directorate", and chart 01's row 11 a
"Director". Chart M1's Non-Financial list names a "Directorship", as do
charts 11 and 13, and p. 68 glosses only that spelling: "A Directorship is
an appointment to the Board of Directors of a large corporation."

Read as: all three are the same benefit. Nothing in the book defines a
Directorate or a Director, and a benefit that appears in one cell with no
rules attached is a spelling rather than a mechanic.

Each word is transcribed as printed alongside the resolved kind, so the
cell says both what the page says and what the engine does with it. Both
appear twice — chart 12 and chart 01 agree with chart M2's reprint of them
— so they are consistent spellings, not slips in one place.

### I-71: Where chart M2 disagrees with a career page, the career page governs (p. 71; p. 67)

Printed page 71 is chart M2, a consolidated reprint of all thirteen
muster-out tables. It disagrees with six of the pages it reprints:

- **02 Scholar** — M2 appends a twelfth row, Cr60,000 / TAS Fellow.
- **03 Entertainer** — M2 stops at twelve rows where the career page has
  thirteen, and its Money DM divides Fame by 5 where the career page
  divides by 3.
- **04 Citizen** — M2 has twelve rows: a new first row, Low Psg / Low Psg,
  with the career page's eleven shifted down beneath it.
- **10 Rogue** and **11 Noble** — M2 reads "+Terms" where both career
  pages read "+Total Terms". p. 68 defines "+Terms" as terms in that
  career; "+Total Terms" it never defines.
- **13 Functionary** — M2 appends a twelfth row, Pension x2 / Knighthood.

Read as: the career page governs, uniformly. The tiebreaker is printed
prose rather than preference — "Each career is fully described on its own
comprehensive page. Once the career is selected, turn to that page and
resolve it according to the rules on that page" (p. 67) — and M2 is a
convenience reprint of pages that instruction sends the reader to.

The counter-argument is real and points the other way on two of the six:
p. 68 defines "+Terms" with a worked example, and defines "+Total Terms"
nowhere, so M2's wording is the one the book explains. It is set aside
because a rule that holds the career page for rows and abandons it for DMs
is two rules, and the first future conflict would reopen the argument. One
rule, explainable in a sentence, revisable in one place.

Both readings are transcribed. `TestM2DivergesExactlyWhereRecorded` pins
the divergence set at exactly these six and fails if a later transcription
resolves one or adds a seventh; `TestTheSevenTablesThatAgreeCarryNoReprint`
holds the other half.

### I-72: Fame is a table D benefit though chart M1 lists it only as an Automatic (p. 70; pp. 76-83)

Chart M1 splits benefits into Financial and Non-Financial columns, and
Fame appears in neither. It appears instead among the Automatics, where
its eligibility is "any".

But four career tables award it as an ordinary benefit cell: chart 02 and
chart 03 at "Fame +1", chart 05 and chart 09 at "Fame +2".

Read as: a benefit like the others, added to the vocabulary. A cell has to
resolve to something, the chart names it, and chart F prices Fame for
every career already. The alternative — treating those four cells as
naming the Automatic — would make them award a character something he has
by default, which is no award at all.

The omission is chart M1's, and the vocabulary records it so a reader
comparing the two lists finds the note rather than the gap.

### I-73: The Fame-19+ muster-out roll goes to the first career served (p. 68)

"He is allowed one additional roll if Fame 19+", and "A character with a
roll allowed by Fame-19+ may select which career-dictated table to use."

The choice is the character's, and the engine has to make it for him. Read
as: the first career served. The page gives no basis for preferring one
table over another — the tables differ, but not in a way a rule ranks —
and first-listed is what the auto policy falls back on when it cannot weigh
(docs/PRD.md, CLI sketch), so putting the roll anywhere else would be a
preference dressed as a rule.

The roll is a character's rather than a career's, so it is added once
across the whole muster out rather than once per career.

### I-74: A duplicate benefit is rerolled until it differs, or until the table has nothing else (p. 69)

"A result that duplicates a previous (unwanted or unusable) benefit may be
rerolled until a different benefit is received, for example: Wafer Jack,
TAS Member, Knighthood."

Read as written: until, not once. The engine rerolls while the result
repeats something already held.

What the rule does not say is what to do when it cannot be satisfied, and
the tables make that reachable. The throw is 1D plus a DM, clamped to the
last row, so it can only land on a span of six rows or fewer; a character
whose DM carries that whole span onto rows holding the same duplicate has
no different benefit to receive, and "until" would never end. The loop
therefore stops when every row the dice can reach repeats something held —
the one case where the duplicate stands.

The question is what the reachable rows offer, not whether they differ
from each other: a span of three different held duplicates differs plenty
and still has nothing to give. Getting that wrong is not a rounding error
but a hang, which is how it was found.

Recorded because the engine rerolled exactly once first, which is a
different rule, and because a Citizen sweep never shows the difference: it
produces single rerolls in quantity and consecutive ones not at all. Chart
02's Benefits column runs twelve deep on seed 72.

### I-75: "+ Officer Rank" counts the number beside the rank, on either ladder (p. 68; charts 06, 08, 10, 12)

Charts 06, 08, 10 and 12 head a muster-out DM column "+ Officer Rank", and
their ladders number the enlisted ranks from 1 alongside the officers':
chart 08 runs S1-S6 beside O1-O7, and chart 06 "R-Ranks are Ratings (or
Enlisted). M-Ranks are Officers."

Read as: the number printed beside the character's rank, whichever ladder
he stands on. An enlisted S4 counts 4.

The rival reading — an enlisted character has no Officer Rank, so no DM —
is the more natural reading of the four words, and it is wrong. Knighthood
sits at row 10 on chart 08, row 11 on chart 07 and row 10 on chart 12, and
the Benefits throw is 1D. Without a DM a non-officer cannot reach those
rows at all, which would make p. 68's "In the Spacer, Soldier, and Marine
careers, Knighthood is only available to Officers. A non-officer receives
Soc +1" a rule for a case the dice cannot produce. The clause names exactly
the three dual-ladder careers, and their enlisted ladders run to 6 — enough
to reach the Knighthood row and no further. That is a system, not a
coincidence.

Chart F shows what the book does when it means to exclude the enlisted: it
footnotes the block "*Armed Forces Enlisted = no Fame" (interpretation
I-65). The muster-out box carries no such footnote. And I-68 already reads
this DM the same way — "chart 06's own muster-out DM says '+ Officer Rank',
which likewise counts the M-number" — with the non-monotonicity that
follows stated there plainly.

Implemented at `chargen/musterout.go` (`dmValue`), pinned by
`TestKnighthoodFollowsItsThreeClauses`, whose non-officer sweep fails if
the DM is withheld from the enlisted.

### I-76: A passage is a benefit, not its cash value, until it is cashed out (pp. 68-69)

p. 68 prices the non-money benefits — "StarPass ... has a value of
Cr250,000", a High Passage Cr10,000 — and p. 69 describes cashing out.

Read as: the price is what the benefit fetches if the character sells it,
not money he already holds. The record carries the passage; `credits` is
what muster out paid. Adding the price to the money would both spend a
ticket the character still holds and count the award twice, once in the
credits and once in the benefits list.

This is recorded because the engine did the other thing first, and the
result was visible: a Craftsman showed Cr575,000 of which Cr250,000 was an
unsold StarPass that the same sheet listed among his benefits.

Cashing out is its own step, and only for Entitlements: "Any Entitlement
can be cashed out for a lump sum" of five years' payments (p. 69).

Implemented at `chargen/musterout.go` (`award`).

### I-77: A dead character does not muster out (p. 67; p. 69)

"Mustering Out counts up the character's belongings (at least the major
ones), the money, and the abilities that a character has accumulated
through several years of career and notes them as assets for the
adventuring situations to come" (p. 67).

Read as: a character who died in generation has no adventuring situations
to come, and takes nothing. p. 69 says it flatly — "the Character is dead
(and all efforts in this particular character creation process are lost)".

The engine still returns the record, which is the narrower reading of that
sentence recorded at I-51: a generator that reports how a character died is
more useful than one that returns nothing. But it stops adding to it. Fame
is computed for the dead and muster out is not, which is the distinction
the two pages draw: Fame is what a character was known for, and benefits
are what he takes with him.

### I-78: Chart M1 lists the Medal automatic for the Spacer and Soldier only (p. 70; p. 86)

Chart M1's Automatics column reads "Medal — Spacer or Soldier". Chart 12
awards Marines medals from the same Imperial Medals table on the same page,
and p. 67's prose is careerless: "A character may have received heroism
medals, campaign ribbons, and wound badges."

Read as printed. The automatic is what chart M1 says it is, and the Marine
keeps the decorations his own chart gave him — they are on his record
either way, and chart F prices them for him. What he does not get is the
line in the Automatics list.

The reading costs nothing and the alternative would be an invention. It is
recorded because a reader comparing a decorated Marine's sheet against a
decorated Soldier's will find the difference and want to know whether it
was noticed.

### I-79: The Automatics are recorded, not logged (p. 70; docs/PRD.md FR10)

"When a character ends character generation he may find that he already own
some specific awards or items" (p. 67). The Automatics are read off the
finished record — a Fighter-1 means a personal weapon, three Discoveries
mean a TAS Life Membership — and nothing is rolled or chosen for them.

FR10 requires a consequence to name the throw or the choice that caused it,
and an Automatic has neither. So they are recorded on the character and
rendered on the sheet, like the UPP and the Life Stage, rather than emitted
as events with nothing to hang from.

The Entitlements are different, and are logged: p. 69 offers a decision on
each of them — "Any Entitlement can be cashed out for a lump sum equal to
five years of payments" — and that decision is what the record hangs from.

### I-80: Reserve years are calendar years, not one stretch per service (p. 69)

"A character who leaves a military, naval, or marine career is
automatically in the Reserves until retirement at Life Stage 9" (p. 67),
and "the Reserve Pension is paid for years actually served as a Reservist,
but only upon reaching Life Stage 9" (p. 69).

Read as: the years from the first service he left until Life Stage 9,
counted once. A character who leaves the Soldiers at 34 and the Spacers at
50 has been a Reservist for the thirty-two years since he was 34, not for
the forty-eight that summing each service's stretch would give him. "Years
actually served" is what forecloses the other reading: forty-eight years
cannot be served inside thirty-two.

Recorded because the engine did the summing first, and the overlap is only
reachable through a career change — no auto-generated character leaves two
services, so nothing in the default sweep would have shown it.

### I-81: A Pension x2 doubles the pension of the career it was rolled on (p. 68)

"Pension x 2 doubles the Pension the character receives from the career.
Each doubling is of the original Pension: the first x2 doubles the Pension,
the second x2 triples the pension, the third x2 quadruples the original
Pension."

Read as: from _the_ career, singular and definite — the career whose table
D the benefit was rolled on. A Pension x2 taken on chart 13 doubles the
Functionary's pension. It does not reach the Reserve Pension, which no
Functionary table can grant, nor a Professor's.

The rival reading takes "the Pension" as whatever pension the character
holds, which is the same thing for the common case of one pension and
wrong for the character p. 69 describes holding two: "a character may
receive duplicate Entitlements (for example, a Reserve and a Functionary
pension)". A benefit rolled on one career's table doubling another
career's pension has nothing in the text behind it.

### I-82: A Land Grant hex on a world the record does not name is priced at the no-TC floor (p. 88; p. 79)

"An unimproved Land Grant generates income based on the Trade
Classifications of the world and equal to Cr10,000 per TC annually (equal to
Cr5,000 if there are no TCs)" (p. 88).

Only one hex has a world this engine knows. p. 88 puts it there — "The first
hex in any grant is on the Noble's homeworld" — and the homeworld's Trade
Classifications are on the record because chart B needs them (docs/PRD.md
FR2). Every other hex sits on a world nobody has generated: p. 88's
subsequent hexes are "randomly allocated", p. 41's companion hex is "on
another world in the system", and p. 79 puts a Scout's grant on "a
non-Mainworld within the Imperium".

Read as: a hex whose world is not named earns the no-TC rate. This is a
deviation and worth naming as one, because "a world with no TCs" and "a
world whose TCs are unknown" are not the same claim, and the engine is
asserting the first where it only knows the second.

What justifies it is the book pricing its own unnamed hex exactly that way.
p. 88's worked example: "recently knighted Sir Richard of Hefry (Trade
Classifications Ni Va) has a Land Grant of one Terrain Hex on Hefry
producing an income of Cr20,000 annually, and a companion Land Grant on a
minor world (no Trade Classifications) elsewhere in the system producing
Cr5,000 annually." The companion world is not generated, named, or rolled
for; it is simply taken to have no Trade Classifications. The engine does
what the example does.

The alternative was to leave grant income uncomputed, which is what this
repo did until now on the stated ground that "a grant's income needs the
world it sits on". That was wrong about the homeworld hex, and the earlier
note is corrected in docs/MILESTONE-4.md.

Implemented at `chargen/entitlement.go` (`landGrantIncome`, `hexIncome`),
with the per-career hex layout in `benefit/data/benefits.json`.

### I-83: The per-title Land Grant hex table is not applied (p. 88; chart 11 p. 85)

p. 88's NOBLE LAND GRANTS table gives each title a hex count — a Gentleman
one hex, a Knight one on the mainworld and one elsewhere, a Baron four and
four, an Emperor 256 and 256 — under the heading "Each title confers its own
Land Grant: a Knight raised to Baronet receives it in addition to his
Knighthood."

That table counts grants by title. I-30 already read chart 11's box A
against the same page's rank-table note and chose box A: "Each increase in
Soc during CharGen awards a Land Grant." The engine counts grants by Soc
increase, and the two rules cannot both drive the same number.

Read as: the count stays with I-30 and the hex table is not applied. Each
grant is priced as one homeworld hex plus one companion (p. 88, p. 41),
which is the shape of the page's own worked example and of its two smallest
titles.

Recorded rather than left implicit because the table is right there beside
the income rule this engine does implement, and its absence would otherwise
read as an oversight. Applying it means reopening I-30, which is a decision
about the count and not about the money.

### I-84: Book 1 attaches no credit value to a Ship Share (p. 90; p. 69; p. 68)

Chart S prices ships in shares and never prices a share: "one Share
acquires 50 tons of the ship (thus, a 200-ton Free Trader requires 4 Ship
Shares to acquire full control)." p. 69 adds only that a share "may be
redeemed upon Mustering Out, or it may be retained and redeemed at some
later date."

Read as: the omission is deliberate, not a gap for the engine to fill.
p. 68 prices the benefits it means to price, to the credit — a StarPass at
Cr250,000, a Low Passage at Cr1,000 — and gives a share no figure on any of
the four pages that discuss one. Chart S's own MCr column prices the ship,
not the share, and the page converts between the two currencies nowhere.

So a share is recorded as a share and valued in tons of ship. PRD FR7 asks
for "ship shares and land grants" among the things muster out settles; what
Book 1 supplies for a share is a redemption rate, and FR7 is amended to say
so rather than the engine inventing a price the rules withhold.

Redeeming the shares for an actual ship stays unimplemented, on two
grounds. I-64 already forecloses it at muster out: Fame is read there — "He
is allowed one additional roll if Fame 19+" (p. 68) — so Fame must be known
before muster out runs, and a Merchant's ship is not known until it has.
And p. 90 makes the purchase an act of play rather than of character
generation: the shares "become available immediately, or the eligibility
may be saved for some future use", and "Characters may pool their ship
shares ... the majority share holder determines the ship type selected",
which is a decision among players and not one the generator can make.

Implemented at `ship` (chart S) and `render` (`shipSharesLine`), which
reports the count and the largest ship it reaches.

### I-85: The Alternative Birthdate Option is not implemented (p. 263)

"Alternative Birthdate Option. Use the Player's actual Birth Date to
determine the day of the year for the Character's Birthdate. Dagin's
birthday is March 6: his Imperial calendar birthdate is (Jan=31)+(Feb=28)+6
= Wonday 065."

Not implemented, and not for want of clarity — the rule is plain and its
two worked examples both check out against the calendar. It takes an input
the engine cannot accept. A player's birthday is a fact about a person
outside the record, and the determinism contract (docs/PRD.md) requires
every value to come from the seed or from a recorded choice, so a birthdate
sourced from the world outside would not replay.

The printed table is implemented and is the rule's own default. The
alternative remains available to a referee, who can simply write the day on
the sheet; nothing downstream reads the birthdate.

### I-86: Every character gets a birthdate, the dead included (p. 263; p. 58)

"Every character has a birthdate, used to track chronological age, to help
produce an understanding of the passage of time, and as a trigger to
acquiring experience" (p. 263).

Read as written: every character. This differs from muster out, which a
character who died in generation does not reach (interpretation I-77), and
the difference is in the two rules. Muster out "counts up the character's
belongings ... as assets for the adventuring situations to come" and a dead
character has none; a birthdate is a fact about when he was born, and dying
does not unmake it.

The arithmetic still works for him. Birth year is the current year less the
age, and a character killed by aging can record an age past the year he
died (interpretation I-52) — his birthdate is computed from the age the
record carries, which is the only age there is.

The step therefore runs outside the `if !c.Dead` that guards muster out,
and every golden fixture carries a birthdate including the two whose
characters died.

### I-87: A consequence may name the step that established the state, where no throw or choice produced it (docs/PRD.md FR10)

FR10 required that "consequence events reference the sequence number of the
throw or choice that caused them". Three consequences cannot satisfy it,
because at the moment they are emitted there is no throw:

- `chargen/rogue.go` — a Rogue serving a sentence. The sentence was set by a
  Risk failure in an **earlier term**; this term merely serves it.
- `chargen/craftsman.go` — "If a Craftsman cannot show at least 40 Master
  Points, he cannot attempt a Masterpiece (treat as Failure)" (chart 01
  p. 75). No throw is made, and the shortfall is points accumulated across
  every term served.
- `chargen/careerrun.go` — chart 01's New Trade cell when every Trade is
  already held (p. 75). The exhaustion is the sum of every prior award.

Each follows from accumulated state rather than from dice. Read as: the
cause may be the step that established the state, and FR10 is amended to say
so rather than being left literally false while the engine did otherwise in
sixteen records out of every sixty Rogues.

The alternative was to thread the antecedent throw forward, and it was
rejected on the merits rather than on cost. The antecedents are all in
earlier terms, so a consequence would point backwards across a term
boundary at a throw whose own consequences were settled long before — which
tells a reader walking the log less than the step directly above it does,
not more.

What the amendment does not license is a dangling cause. A consequence
naming sequence zero refers to no event at all, and
`TestEveryConsequenceNamesItsCause` fails on one.

Implemented at `chargen/rogue.go` (`prisonTerm`), `chargen/craftsman.go`
(`attempt`) and `chargen/careerrun.go` (`awardNewTrade`).

### I-88: A suspended term costs the term's four years, not the program's duration (p. 59)

"At the beginning of any term, the character may apply for any Educational
Institution or Training, and if accepted substitutes that process for the
entire term" (p. 59).

The two readings differ whenever the program is shorter than a term. Trade
School is one Pass/Fail year; a term is four. Read as the program's
duration, a character could take Trade School for one year and serve the
remaining three, getting the schooling almost free. Read as the term, the
whole four-year slot is given over.

The term is the object the sentence replaces — "substitutes that process
**for the entire term**" — so the term is what is spent. A one-year Trade
School costs four years, and a College that fails its Pass/Fail in the
second year is not refunded the other two: the term was given over, not
rented by the year.

The process charges its own years as it goes, exactly as it does before a
career, and what remains of the term is charged against the choice that
spent it. Nothing is charged twice.

Implemented at `chargen/latereducation.go` (`attendMidCareer`), pinned by
`TestLaterEducationSubstitutesTheTerm` on Trade School, where the two
readings differ by three years.

### I-89: A refused applicant serves the term, having already lost a year to the refusal (p. 59)

Substitution is conditional: "and **if accepted** substitutes that process
for the entire term" (p. 59). A character who applies and is refused has not
been accepted, so nothing is substituted and the term is served.

He is still out the year the application cost: "A failure disallows
admission and consumes one year" (p. 59). That sentence is about the
application, not about the term, so the two compose — the cycle costs five
years, one for the refusal and four for the term he then serves.

That is harsh, and it is what the two sentences say. The alternative, losing
the term to a school that never admitted him, is worse and has no textual
support at all: the term is substituted only on acceptance.

It also matters mechanically, and more than the text alone suggests. A
refused application means a term is served, which means a Continue throw is
made — so a character who applies every term is not thereby immune to his
career ending, which is half of why the lifepath terminates (see I-90).

The other reading does not merely mispay the years: it hangs. Mutating the
engine so a refusal suspends the term anyway makes generation loop forever
on a decider that always applies. Apprenticeship has no prerequisite and
takes no time (chart C Duration), so a refused Apprenticeship under that
reading would cost nothing and consume the term, and nothing would ever
advance the clock toward the aging that has to end the lifepath. The
printed reading is also the terminating one.

Implemented at `chargen/latereducation.go` (`attendMidCareer`, the
unadmitted return), pinned by `TestLaterEducationTerminates`.

### I-90: Suspending a term suspends the Continue throw with the rest of resolution (p. 59)

"Characters may **suspend career resolution** to return to school or
training" (p. 59). The Continue throw is part of career resolution, so a
suspended term throws no Continue, just as it runs no Risk/Reward and awards
no career skills.

The consequence is worth stating plainly: a character at school cannot have
his career end that term. The rule hands the player a way to sit out a term
he would rather not throw for, and that is a real effect of the printed
sentence rather than an oversight in reading it.

A suspended term is also not a term served. No TermRecord is appended, so it
counts toward neither the muster-out benefit rolls nor the pensions, both of
which count terms (`len(record.Terms)`). A character is not paid for service
he spent at school.

The lifepath still terminates, but only because the engine reads death at
the right place, and this entry first claimed otherwise. The years pass
whether they are spent serving or studying, so aging arrives on schedule and
kills (chart A p. 89); and an application may be refused, which serves the
term and throws its Continue after all (I-89). Both were true as written.
What was missing was the check: a character who died at school was never
noticed, and since aging stops checking once Dead is set, a lifepath that
outlived its own death could not end at all. Seed 111 hung. The sweep that
was said to pin the claim ran over five seeds and did not reach it — five
seeds is an anecdote, not a sweep, and `TestLaterEducationTerminates` now
runs 150.

That death has to be read where the term loop resumes, and not only where a
term was served. Aging kills at school as readily as in service, and the
refused applicant's lost year passes too, so the loop checks once the offer
is resolved, whichever way it went: a corpse is not offered school again,
and does not serve the term he applied out of. The check is load-bearing
rather than tidy — once a character is dead, aging stops checking, so a
loop that survives the death has nothing left that can ever end it.

Implemented at `chargen/careerrun.go` (`term`) and
`chargen/latereducation.go`, pinned by `TestLaterEducationSuspendsResolution`,
`TestLaterEducationIsNotATermServed` and
`TestLaterEducationDeathEndsTheLifepath`.

### I-91: An assigned school is attended inside the term and costs no extra years (p. 59; charts 07, 08, 12; chart C p. 60)

"Some schools are attended during career resolution (assigned as part of
career resolution)" (p. 59). That sentence follows Later Education and
describes a different thing: not a term given over to schooling, but a
school the career hands the character while he serves.

Chart C gives ANM School and Command College a Duration of 1 year each, and
the question is whether that year is added to the term or is one of its
four. It is one of its four. ANM School arrives as an Operations
assignment, and the Operations rolls are what the term is spent doing;
Command College is sited by its own footnote as "in Year 1 of next Term",
which places it inside a term rather than beside one. Neither sentence
describes a character who serves four years and studies a fifth.

So an assigned school runs with the education process's year-charging
suppressed, and the term charges its four as always. The contrast with
Later Education is exact and deliberate: that one substitutes "for the
entire term" and costs all four (I-88), because there the term is what is
being replaced.

One consequence of siting the school inside a term: Command College waits
for a term that is actually served. "In Year 1 of next Term (if Continue)"
names a term, and a term suspended for Later Education is "not a term
served" (I-90) — there is no Year 1 of service to hold the college in. So
an officer who takes the footnote and then suspends the following term
attends Command College in the next term he serves, rather than losing it.

Implemented at `chargen/assignedschool.go` (`attendAssignedSchool`, setting
`eduRun.withinTerm`) and `chargen/armedforces.go` (the `commandCollege`
flag, read in `resolveTerm` and so never reached by a suspended term),
pinned by `TestAnAssignedSchoolCostsNoExtraYears`.

### I-92: An assigned school selects no Major or Minor (p. 59; chart C p. 60)

"The character attending an Educational Institution must select a Major and
a Minor from the appropriate Skill and Knowledge list" (p. 59).

Neither assigned school takes one. Chart C files ANM School and Command
College under Military rather than under the Educational Institutions the
sentence names, and — the stronger reason — what the two rows provide is
stated outright: "Knowledge-2 from School=ANM" and "2x Skill-1". Neither
reads a Major or a Minor, so a Major selected at an assigned school would
change nothing about the character.

Presenting two choices that cannot affect the outcome would put noise in
the record and two more indices in the replay stream for no rule. The
Major and Minor a character already carries are untouched: "A character's
current Major and Minor are the most recent ones selected" (p. 59) speaks
of institutions that ask for them, and an assigned school does not ask.

Implemented at `chargen/assignedschool.go` (`attendAssignedSchool`, which
does not call `selectMajors`).

### I-93: ANM School is resolved once per term and after the four Operations rolls (charts 07, 08, 12; p. 66)

"Rolls 4 times per Term for Operations; select the highest Mod of the
four" (chart 08), and "Resolve ANM School using Education".

Two things the charts leave open.

**When.** The four rolls are one block: they determine the term's
assignments and the Mod applied to Risk and Reward (p. 66). Resolving the
school between them would interleave its Admission and Pass/Fail throws
with the assignment rolls, splitting a block the rule treats as a unit and
making the transcript harder to walk. The school is resolved once the four
are known.

**How many times.** The assignment can come up more than once in the four
rolls. It is resolved once regardless. Chart C gives ANM School a Duration
of one year and the term has four; a character sent four times would spend
the whole term at a school the chart sizes at a quarter of it, and the
Operations block describes one term's assignment pattern rather than four
separate postings.

Implemented at `chargen/armedforces.go` (`operations`), pinned by
`TestANMSchoolIsResolvedAsEducation`.
