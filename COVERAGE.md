# COVERAGE.md — rule coverage map

The docs/PRD.md milestone-3 exit criterion: every Master Chargen Checklist
step (chart E1, p. 72) and career rule maps to its page cite, its
implementation site, and its test — no career is done until its uncommon
branches are listed here as covered or explicitly deferred.

Statuses: **covered** · **deferred (M#)** with the owning milestone ·
**interpretation** rows also name their ERRATA.md entry. Cites are to
Book 1, Print Edition 5.1.

## Foundations

| Rule                                         | Cite             | Implementation                   | Test                                          | Status                                                       |
| -------------------------------------------- | ---------------- | -------------------------------- | --------------------------------------------- | ------------------------------------------------------------ |
| xD / xD±k rolls                              | pp. 18–19        | `dice.Roll`/`RollMod`            | `dice/dice_test.go` properties, golden stream | covered                                                      |
| Flux, Good/Bad Flux                          | pp. 19, 261      | `dice.Flux`/`GoodFlux`/`BadFlux` | exhaustive 36-combination tables              | covered                                                      |
| Roll-low target throws                       | pp. 120, 122     | `dice.Throw`                     | `dice/throw_test.go`                          | covered                                                      |
| Spectacular success/failure flags            | p. 127           | `dice.Throw` flags               | `TestResolveThrow`                            | covered (override semantics applied at call sites; none yet) |
| Many Dice procedures                         | p. 260           | —                                | —                                             | deferred (needs ≥11D; chargen never rolls it)                |
| D/2 half-die                                 | p. 19            | —                                | —                                             | deferred ("rarely used")                                     |
| eHex 0–33, I/O omitted                       | p. 22            | `ehex`                           | exhaustive round-trip                         | covered                                                      |
| Seeded stream + replay contract              | docs/PRD.md      | `dice.New`                       | golden sequence pin                           | covered (replay verifier itself is M5)                       |
| Skill maximum 15                             | p. 134           | `Character.awardSkill`           | `checkSkills` sweep                           | covered (Knowledge-6 cap deferred, M3 MS list)               |
| Characteristic maximum 15                    | p. 68            | `awardCharacteristicAndLog`      | `TestCharacteristicMaximum`                   | covered                                                      |
| Event log: steps/throws/choices/consequences | docs/PRD.md FR10 | `chargen/event.go`               | seq/payload/isolation/JSON-shape tests        | covered                                                      |

## E1 step A — Generate Characteristics

| Rule                                  | Cite                     | Implementation            | Test                          | Status                         |
| ------------------------------------- | ------------------------ | ------------------------- | ----------------------------- | ------------------------------ |
| 2D ×6 in roll order Str…Soc           | chart A p. 56            | `RollCharacteristics`     | stream-order test, golden UPP | covered                        |
| Psi and Sanity deferred, never rolled | chart A p. 56; non-goals | `RollCharacteristics` doc | golden stream                 | covered (v1 excludes psionics) |
| UPP in SDEIES eHex                    | pp. 22, 47–48            | `Characteristics.UPP`     | `TestUPP`, golden             | covered                        |

## E1 step B — Determine A Homeworld

| Rule                                             | Cite                      | Implementation                            | Test                              | Status                                     |
| ------------------------------------------------ | ------------------------- | ----------------------------------------- | --------------------------------- | ------------------------------------------ |
| Assigned/selected homeworld; tool default Regina | p. 58; chart B p. 56; FR2 | `runHomeworld`, `world.Default`           | `TestHomeworldDefault`            | covered                                    |
| Random selection from the chart B world list     | chart B p. 56             | —                                         | —                                 | deferred (M5 interactive)                  |
| One skill at level 1 per TC                      | p. 58                     | `grantTC`                                 | homeworld tests                   | covered                                    |
| Ri One Art / In The Trades selections            | chart B p. 56             | `grantSelection`                          | `TestHomeworldDefault`/`Supplied` | covered                                    |
| Ds Deep Space double grant                       | chart B p. 56             | data + `grantTC`                          | `TestHomeworldSupplied`           | covered                                    |
| UWP validation, never repaired                   | FR2                       | `world.ValidateUWP`, `Homeworld.Validate` | `TestValidateUWP`, error tests    | covered                                    |
| TCs not derivable from UWP                       | p. 58; FR2 note           | docs/PRD.md FR2                           | —                                 | covered (derivation out of pinned ruleset) |

## E1 step C — Education and Training

| Rule                                                                               | Cite          | Implementation                            | Test                             | Status                                                                                   |
| ---------------------------------------------------------------------------------- | ------------- | ----------------------------------------- | -------------------------------- | ---------------------------------------------------------------------------------------- |
| Optional education; prerequisites are minimums                                     | pp. 57, 59    | `chooseProgram`                           | invariants sweep                 | covered                                                                                  |
| ED5, Trade School, Apprenticeship, College, University, Service Academy            | chart C p. 60 | `chargen/education.go` + `education` data | program/matrix locks, sweeps     | covered                                                                                  |
| Mentor                                                                             | chart C p. 60 | data only                                 | `TestPrograms`                   | deferred (needs non-human Tra, v1 non-goal)                                              |
| Training Course, Masters, Professors, Medical, Law, ANM, OTC/NOTC, Flight, Command | chart C p. 60 | data only (`implemented: false`)          | `TestPrograms`                   | deferred (post-BA / career-integrated, M3+)                                              |
| Later Education (suspend a career term)                                            | p. 59         | —                                         | —                                | deferred (M4, with career changes)                                                       |
| Major/Minor selection after admission; cannot match                                | p. 59         | `selectMajors`                            | goldens; review fix              | covered                                                                                  |
| Admission check; failure costs one year                                            | p. 59         | `apply`                                   | goldens                          | covered                                                                                  |
| Pass/Fail per year; Major+1/pass, Minor+1/2 passes                                 | pp. 59–60     | `passFailYear`, `awardPass`               | invariants, goldens              | covered; failed year elapsing = interpretation I-5                                       |
| Waivers: Soc check, −1 per prior attempt                                           | p. 59         | `waiver`                                  | pinned seed 1                    | covered for Application + Pass/Fail; Prerequisite + Honors waivers deferred (M5)         |
| Honors: optional roll, Major+1, no effect on failure                               | p. 59         | `honors`                                  | goldens; fresh Int-or-Edu choice | covered                                                                                  |
| Language at double rate                                                            | p. 59         | `majorRate`                               | `TestLanguageDoubleRate`         | covered                                                                                  |
| Graduation Edu=N / +1 if already there                                             | chart C p. 60 | `graduate`                                | `checkGraduationEdu`             | covered                                                                                  |
| Human Tra checks at Edu/2                                                          | pp. 55, 60    | `checkValue`                              | —                                | interpretation I-6 (policy-unreachable; direct test with Apprenticeship decider pending) |
| Apprenticeship unrestricted skill list                                             | chart C p. 60 | `awardApprenticeship`                     | —                                | interpretation I-7 (policy-unreachable)                                                  |

## E1 step D — Careers (generic)

| Rule                                                  | Cite                      | Implementation               | Test                                                | Status                                  |
| ----------------------------------------------------- | ------------------------- | ---------------------------- | --------------------------------------------------- | --------------------------------------- |
| Career selection; forced first career                 | chart D p. 64; CLI sketch | `runCareer`, registry        | `TestRegistryMatchesAvailable`, forced-career tests | covered for Citizen                     |
| CC rotation (no reuse until all used)                 | pp. 64–65                 | `careerRun.chooseCC`         | rotation-window sweep                               | covered                                 |
| Per-term skills table (column choice + 1D)            | p. 65                     | `careerRun.termSkills`       | goldens, sweeps                                     | covered                                 |
| Trade/Art/Science cells                               | p. 78 table C             | error (`errNotImplemented`)  | direct data test                                    | deferred (M3); unblocked — the MS list supplies the 10 Trades, 6 Arts, and 13 Sciences |
| Continue roll; mandatory on exactly 2                 | p. 66                     | `careerRun.continueRoll`     | pinned seed 3                                       | covered                                 |
| 4-year terms; age accounting                          | p. 66; p. 59 (age 18)     | `continueRoll`, `StartAge`   | `checkEducationAge`                                 | covered                                 |
| Generic Risk/Reward (Caution/Bravery, injury, reward) | pp. 64–65                 | —                            | —                                                   | deferred (M3, first non-Citizen career) |
| To Begin throws with retry                            | pp. 64–65                 | `careerMechanics.begin` seam | —                                                   | deferred (M3; Citizen is automatic)     |
| Rank / Commission / Promotion                         | p. 65                     | —                            | —                                                   | deferred (M3 military/Merchant/Scholar) |
| Career changes                                        | p. 66                     | —                            | —                                                   | deferred (M4)                           |

## Career 04 — Citizen (chart 04, p. 78)

| Rule                                              | Cite                  | Implementation           | Test                     | Status                                                                                    |
| ------------------------------------------------- | --------------------- | ------------------------ | ------------------------ | ----------------------------------------------------------------------------------------- |
| Begin automatic; no transfer into Citizen         | p. 72 panel 04; p. 66 | `citizenMechanics.begin` | goldens                  | covered (no-transfer becomes testable with M4 career changes)                             |
| Citizen Life 2D ≤ CC, no mods                     | p. 65; chart 04       | `resolveTerm`            | sweeps, goldens          | covered                                                                                   |
| Success ladder: Job-4, Hobby-2, alternating +1    | chart 04              | `awardCitizenLife`       | ladder sweep             | covered                                                                                   |
| Job: table E 3-dice roll, rerolled A faces logged | p. 78                 | `determineJob`           | goldens, event integrity | covered                                                                                   |
| "No Skill" cell → retry next success              | p. 78                 | `determineJob`           | —                        | interpretation I-1 (`job_undetermined` event; low-probability branch, no pinned seed yet) |
| First-receipt vs career baseline                  | p. 78                 | `firstReceiptLevels`     | review-verified seed 7   | covered; interpretation I-2                                                               |
| Hobby excludes the Job                            | p. 78                 | `determineHobby`         | `checkHobby`             | covered; interpretation I-3                                                               |
| No rank                                           | p. 65                 | (nothing to do)          | —                        | covered                                                                                   |
| Muster out row (chart 04 D)                       | pp. 67, 70–71, 78     | —                        | —                        | deferred (M4)                                                                             |

## Career 05 — Scout (chart 05, p. 79)

| Rule | Cite | Implementation | Test | Status |
| --- | --- | --- | --- | --- |
| To Begin C1/C2/C3; failure costs a year, career unusable | chart 05; p. 65 | `scoutMechanics.begin` | begin-failure + fallback sweeps | covered (no Begin retry under I-8 reading 1) |
| Begin-failure fallback to remaining careers | p. 65 | `runCareer` loop | `TestScoutBeginFallback` | covered (empty options = legal no-career end) |
| Duty choice: Courier avoids R&R; eligibility 4/8 | p. 79; chart 05 B | `resolveTerm`, `SkillEligibility` | invariants sweep | covered (policy: Explorer) |
| Caution/Bravery/No Mod selection | p. 65; chart 05 | `chooseRiskMod` | Bravery decider tests | covered (policy: No Mod) |
| Risk vs CC+Mods; failure injury: negative mods + Flux, no increase | chart 05 | `injury` (+`Log.Flux`) | injury sweeps | covered (Scout clamp; generic p. 65 Flux-compensation variant lands with the next R&R career) |
| Wound Badge on reduction | p. 65 | `injury` | badge-accounting sweep | covered |
| Disabled at 4+: term completes, no Continue, career ends | chart 05; p. 65 | `injury`, `term` | `TestScoutInjuryOutcomes` | covered (Double Benefits deferred, M4 muster out) |
| Dead at CC ≤ 0: term and generation end at the injury | p. 65 | `injury`, `term` | `TestScoutInjuryOutcomes` | covered |
| Reward vs CC−Mods; success = Discovery, Fame +1 | chart 05 | `reward`, `discovery` | fame==discoveries sweep | covered (Land Grant values deferred, M4) |
| Retry R&R C5 | chart 05 | `retryReward` | policy-always path in sweeps | interpretation I-8 (three readings recorded) |
| Continue vs Int | chart 05 | `careerRun.continueRoll` | Continue-Int cite sweep | covered |
| Sanity −1 per two terms | chart 05 | — | — | deferred (San untracked; chart A defers it) |
| Muster-out table D; Land Grant economics | chart 05; pp. 70–71 | — | — | deferred (M4) |
| "Starship Skill" cells | chart 05 table C | `EntryStarship` (errors if selected) | data test | deferred (M3); unblocked — the MS list now supplies the seven Starship Skills |
| No rank | p. 65 | (nothing to do) | — | covered |

## Career 06 — Merchant (chart 06, p. 80)

| Rule | Cite | Implementation | Test | Status |
| --- | --- | --- | --- | --- |
| Three entry tracks; each enters its own rank | chart 06 A | `merchantMechanics.begin`, `BeginTracks` | `TestMerchantBeginTracks` | covered |
| "To Begin Temp Auto" needs no throw | chart 06 A | `begin` | `TestMerchantTempIsAutomatic` | covered |
| Entry failure costs a year, career not begun | chart 06 A; p. 65 | `begin` | `TestMerchantBeginFailure` | interpretation I-15 (no fall-through to Temp) |
| Risk & Reward C1 C2 C3 C4; Caution/Bravery/No Mod | chart 06; p. 65 | `riskAndReward`, `chooseRiskMod` | golden seed 17 | covered |
| Risk failure: injury, Wound Badge, disabled at 4+, dead at zero | chart 06; p. 65 | `careerRun.injury` (shared) | Scout injury suite | covered (Double Benefits deferred, M4) |
| Reward success: escalating Ship Shares | chart 06 | `awardShipShares` | `TestMerchantShipShareEscalation` | covered; seventh receipt is I-14, economics deferred (M4) |
| Officer Commission (Int) from a rating; lands at M1 | chart 06 A | `advance`, `attempt` | `TestMerchantCommission` | covered |
| Rating Promotion (Dex, +3 if Int 8+) | chart 06 A | `advance` | `TestMerchantRatingPromotion` | covered |
| Officer Promotion (Terms x2, +3 if Int 8+) | chart 06 A | `advancementTarget` | `TestMerchantOfficerPromotionTarget` | interpretation I-12 (completed terms) |
| Which rows each class may attempt per term | chart 06 | `advance` (entry-class snapshot) | `TestMerchantCommission` | covered — a commissioned Temp does not also attempt Officer Promotion |
| Top of a ladder bars the attempt | chart 06 | `eligibleForAdvancement` | `TestMerchantRanksAreCharted` | interpretation I-13 |
| Automatic Skills by rank | chart 06 B | `enterRank` | golden seed 17 | covered (ordinary receipts, p. 66) |
| Per Term 4 + 1 per rank gained | chart 06 B | `resolveTerm` via `termOutcome.skillRolls` | `TestMerchantAdvancementEarnsSkills` | covered |
| Continue vs Str | chart 06 A | `careerRun.continueRoll` | golden seed 17 | covered |
| Disability and the rest of the term | chart 06 | `term` via `endCareer` | Scout precedent | interpretation I-16 |
| Muster-out table D; ship-share economics | chart 06 D | — | — | deferred (M4) |

## Cross-cutting — throw resolution

| Rule | Cite | Implementation | Test | Status |
| --- | --- | --- | --- | --- |
| Checks fail on the highest possible roll (2D on 12) | p. 134-135; chart 10 | `dice.Check` | `TestCheckAutomaticFailure` | interpretation I-17 — applied at every chargen throw; guarantees careers terminate |
| "One Art" / "One Trade" / "One Science" / "Starship Skill" cells | charts 04-06 table C; p. 132 | `careerRun.awardFromGroup` | Merchant golden | covered (alternatives are the chart MS groups) |

## Careers 01–05 remainder, 07–13

All deferred (M3): Craftsman, Scholar, Entertainer, Spacer, Soldier,
Agent, Rogue, Noble, Marine, Functionary (charts pp. 75–88). Functionary
additionally needs career changes (M4): it "is never a first career".
Each career's section is added here with its chunk, uncommon branches
enumerated, before it is called done.

## Master Skill List (chart MS, p. 132)

| Rule | Cite | Implementation | Test | Status |
| --- | --- | --- | --- | --- |
| The 64 skills; list is closed ("no others available") | p. 132 | `skill/data/master_skill_list.json` | `TestListLoads`, `TestSkillGroups` | covered (count checked against the printed 64) |
| Knowledges by parent skill; Talents, Personals, Intuitions | p. 132 | same | `TestLookupKinds` | covered (Knowledges/Talents advisory per the page note) |
| Default Skills (usable at level-0 by any character) | p. 132; p. 133 | `Entry.Default` | `TestDefaultSkills` | transcribed; the level-0 default rule itself is deferred (task resolution, out of chargen scope) |
| Chart labels resolve to list names | p. 132 vs charts | `skill.Resolve` + loader checks | `TestResolveCanonical`, `TestAwardedSkillsAreCanonical` | interpretation I-9 |
| Grav under three parents | p. 132 | qualified names; `resolveSkillName` | `TestQualifiedKnowledges`, `TestAmbiguousChartCellsResolveByChoice` | interpretation I-10 |
| "Spacecraft" cell | p. 78 vs p. 132 | `labels`; `resolveSkillName` | `TestAmbiguousChartCellsResolveByChoice` | interpretation I-11 |
| Knowledge-Knowledge-Skill progression on receipt | p. 133 | — | — | deferred (M3 follow-up: award semantics) |
| Knowledge-6 / World-6 / Career-6 maximums | p. 134 | — | — | deferred (M3 follow-up; the Skill-15 cap is applied, Talent-15 is moot until talents are awarded) |
| Career: <Name> and World: <Name> knowledges | p. 134 | — | — | deferred (M4: needs terms-served and residence accounting) |
| Sciences specialization beyond level 6 | p. 134 | — | — | deferred (M4) |
| Education provides only contained Knowledges for the ten container skills | p. 134 | `knowledge_only` rows | education matrix tests | covered (data); award semantics with the progression above |

## E1 step E — Muster Out, and later steps

Muster out (pp. 67, 70–71), aging (chart A p. 89; FR6), fame (chart F
p. 91; FR7), career changes, birthdate (FR8): all deferred (M4).
Batch, replay verification, interactive mode: deferred (M5).

## Cross-cutting interpretations

I-1 … I-17 in ERRATA.md, each referenced from its row above. The I-4
skill-name residual (MSL-exact strings, the Navigator/Navigation split) is
closed by I-9: every skill name in every embedded chart is validated
against the Master Skill List at load time.
