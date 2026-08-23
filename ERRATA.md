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
birthday." The engine has no birthdate: FR8's is cited to the Archive,
which the ground rules exclude, and Book 1 prints no birthdate rule.

Read as: the checks fall at ages 34, 38, 42 and so on — the four-year
cadence anchored to the age Physical Aging begins at. Anchoring to
absolute ages rather than to elapsed time matters because a failed career
entry costs a single year (p. 65), which would otherwise knock a character
permanently off the four-year grid and change how many checks a lifetime
holds.

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
