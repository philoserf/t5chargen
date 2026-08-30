# PRD: T5 Character Generator (Go CLI)

2026-08-19. Status: the v1 contract, complete.

This document states what v1 is, once. It carried a dozen dated
amendments recording how each requirement reached its final reading —
those are folded into the requirements they amended, so a reader can
judge the tool against this page without reconstructing the order the
decisions were made in. The history is not lost: `docs/MILESTONE-4.md`
through `-7.md` record what each milestone turned out to be, and git
holds the rest.

## Problem

T5 character generation is a long chart-driven procedure with career-specific rules. Doing it by hand is slow and error-prone. Build a Go CLI that generates rules-accurate T5 characters, as a sibling to `philoserf/world-builder`. Ruleset baseline: Book 1, Print Edition 5.1 (the v5.10 three-book split), as held in the Traveller collection (`~/Documents/Traveller/T5/`) — all page cites are to that artifact.

## User

Mark: solo referee and developer. Secondary: any T5 referee needing NPCs in bulk or players wanting a guided lifepath.

## Goals

1. Generate a complete human character per the Master Chargen Checklist (Book 1 chart E1, p. 72), from homeworld through muster out.
2. Two modes: **interactive** (player makes each choice) and **auto** (tool decides; supports batch NPC generation).
3. Deterministic replay: re-running the engine from the recorded seed and choices reproduces the identical character (see Replay and provenance contract).
4. Output a character record as JSON (canonical) and a Markdown character sheet modeled on the Character Card (Book 1 chart C1, p. 98).
5. Emit a generation record: the full chronological history of the lifepath — every throw, choice, and outcome — embedded in the JSON and renderable as a Markdown transcript.

## Non-goals (v1)

- Custom sophont creation and non-human characteristic variants (chart C2, p. 99; Genetics charts pp. 106–110).
- Psionics, clones, chimeras, robots, artificials.
- Permatic Imperium variants; other editions (Classic, Mongoose, Cepheus).
- Combat, equipment purchasing beyond mustering-out benefits, in-play advancement.

## Functional requirements

Rule citations are to Traveller5 Core Rules Book 1 (Print Edition 5.1) chart pages; prose chapters elaborate each.

**FR1 — Characteristics.** Roll C1–C6 (Str, Dex, End, Int, Edu, Soc for standard humans; charts pp. 54–56). Store as UPP hex string plus named values.

**FR2 — Homeworld.** Determine birthworld/homeworld and grant default homeworld skills (chart B, p. 56; Archive: `Homeworld Skills.pdf`). World generation itself is out of scope, and v1 treats birthworld and homeworld as the same world.

The homeworld input is `"UWP"` or `"UWP TC TC..."` (`--homeworld "A788899-C Ph Pa Ri"`), because the UWP alone does not suffice: chart B awards "one specified skill for each Trade Classification or Remark" (p. 58), and several TCs (Cp/Cs/Cx capitals, Da, Mr, Px, Re, ...) are not derivable from a UWP at all — deriving even the derivable ones needs the Book 2/3 trade-classification definitions, outside the pinned Book 1 ruleset. A bare UWP therefore grants no homeworld skills; `world-builder` output carries TCs and passes them through. Invalid or partial UWPs are rejected with an error, never silently repaired, as are unknown and repeated TCs (each TC appears on a world at most once and grants exactly one specified skill, p. 58). The tool-owned default is Regina A788899-C (Ph Pa Ri) — chart B's own row and the book's worked example (p. 58).

**FR3 — Education.** Pre-career education options: ED5, Trade School, College/University, Service Academy, Apprenticeship, Mentoring — with majors, minors, honors, waivers, and time costs (chart C, p. 60).

Mentoring is transcribed and unreachable for a v1 character: its "C5= Tra" prerequisite and "Tra+2" award both need the Training characteristic, which non-human characteristic variants put outside v1. It is named here because chart C names it, not because a human can take it.

**FR4 — Careers.** All 13 careers (charts pp. 75–88): Craftsman, Scholar, Entertainer, Citizen, Scout, Merchant, Spacer, Soldier, Agent, Rogue, Noble, Marine, Functionary. Per career: To Begin throw with failure/retry handling, term resolution, Risk/Reward, Rank/Commission/Promotion, career-specific mechanics (Entertainer Fame and Big Break, Craftsman Masterpiece, Agent undercover career and Commendations, Rogue Schemes, Noble exile, military Branch/Operations/Schools/Medals), skill eligibility, Continue throws, and career changes (Archive: `Changing Careers.pdf`).

**FR5 — Skills.** Award skills per term against the Master Skill List (chart MS, p. 132), with first-receipt rules (e.g. Job Skill-4 on first receipt, Skill-1 thereafter) and the Skill/Knowledge distinction: p. 134's Knowledge-Knowledge-Skill progression, under which a container skill's first two receipts award one of its contained Knowledges instead, so five receipts of Fighter leave Fighter-3 and a Knowledge-2. Knowledges cap at 6. The contained lists are the skill descriptions' own, pp. 135-167 — a `KNOWLEDGES` sidebar for Driver, Fighter, Flyer, Gunner, Heavy Weapons and Seafarer, a titled box for Engineer, and inline enumerations for Animals and Pilot — not chart MS, whose note calls its list "advisory" and which prints no contained Knowledges for any container.

Two containers are exceptions, recorded as limits rather than as scope: Musician prints no instrument list anywhere and is awarded whole (I-111), and Language is excepted by p. 134 itself.

**FR6 — Aging.** Aging checks each term at life stage 5+ (age 34 for humans; chart A, p. 89).

**FR7 — Muster out.** Fame determination, automatics, entitlements, money/benefit throws per career (charts M1-M2, pp. 70-71), including land grants (p. 88) and ship shares (chart S, p. 90).

Land grants and ship shares are two different things because the book gives two different things. A **Land Grant has a printed value** — "Cr10,000 per TC annually (equal to Cr5,000 if there are no TCs)" (p. 88) — and it is computable: p. 88 sites the first hex of any grant on the character's homeworld, whose Trade Classifications the record already carries for chart B, and p. 41 adds one companion hex per mainworld hex. That is the pair p. 88's worked example prices. A hex on a world the record does not name is priced at the no-TC floor, as the book prices its own unnamed companion world (deviation I-82). A **Ship Share has no printed value**: Book 1 attaches no credit figure to one anywhere, and chart S prices ships in shares instead — one share buys 50 tons (I-84). So FR7 is read as grant income in credits per year and share value in tons of hull. Redeeming shares for an actual ship is not part of muster out: the Fame ordering forecloses it (I-64) and p. 90 makes the purchase an act of play.

**FR8 — Character record.** Track name, birthdate, homeworld, career history term by term, ranks, medals, fame, money, benefits, and full skill list. The JSON record is the source of truth; the Markdown sheet is a render of it.

The birthdate is printed on two pages of Book 1: p. 58 ("Date of Birth") sets the default current date at 001-1105 and subtracts the character's age at muster out from it, and p. 263 ("Birthdates") gives the Birth Date Generation table — four consecutive dice, rerolling on RR — with pp. 262-263 giving the calendar the day is named on. One clause is deliberately not implemented: p. 263's Alternative Birthdate Option takes the player's real birthday, an input from outside the record that the replay contract cannot accept (I-85).

**FR9 — Dice engine.** xD throws, Flux (1D−1D), and target-number throws as a distinct, unit-tested package; all rolls consumed from a seeded stream and logged for replay.

**FR10 — Generation record.** An ordered event log of the entire generation: each checklist step entered, each throw (dice, target, modifiers, result), each choice (who decided — player or policy — and what the alternatives were), each consequence (skill awarded, rank gained, characteristic change, year elapsed). Stored as an `events` array in the character JSON. Events carry monotonic sequence numbers; consequence events reference the sequence number of the throw or choice that caused them, or of the step that established the state where no throw or choice did (interpretation I-87); throw events record the dice expression, individual dice, target, modifiers, and rule citation. Serves three purposes: **audit** (verify any character against Book 1 by walking the log), **replay** (verification data for goal 3 — see the contract below), and **narrative** (render as a readable lifepath transcript — the character's biography in game terms).

## Replay and provenance contract

- Every character JSON carries: `schema_version`, `ruleset` (pinned: Book 1 Print Edition 5.1), `engine_version`, `policy_version`, `rng` (algorithm + seed), an `inputs` block, and any applied `ERRATA.md` deviations. Old characters stay auditable after embedded tables change.
- RNG: Go `math/rand/v2` PCG, named in the record. Changing algorithm or policy is a version bump, since either changes seeded output.
- Replay re-runs the engine from the recorded seed, inputs and choice events, recomputing every throw; the stored event log is verification data, not input. `t5chargen replay character.json` exits non-zero at the first mismatch, reporting the diverging event's sequence number.

The seed and the choice events alone are not sufficient, which is why records carry `inputs`. Two of the engine's inputs were recorded nowhere: the `--career` force and `--current-year`. The force matters because a recorded choice is an **index**, and forcing a career holds the first career's option list to a single entry — so a replay that did not know a record was forced would offer the full eligible list and read the recorded index against it. Eleven of the fourteen golden records are forced. The name and homeworld are inputs too, but they were already stored as character state.

`policy_version` is deliberately **not** verified on replay: recorded choices are reapplied and the policy is never consulted, so a record made under one POLICY.md version replays under any other — which is also what makes the two fixtures carrying `"none"` replayable at all. The same follows for **who decided**: the replay decider is not the original, so the only decider it can name in a re-run's choice events is the recorded one, which then matches itself. Replay therefore attests that the recorded choices rebuild the recorded character — not that the decider the record names would have made them. A record whose `decider` fields or `policy_version` were altered replays clean, and that is inherent to replaying from the record rather than a gap in the verifier.

`t5chargen replay --ignore-provenance` re-runs a record whose versions do not match and reports where the generation disagrees. `engine_version` moves roughly once per feature, so a record can be orphaned hours after it is written, on the branch that wrote it — and the record it orphans may be the one that cannot be produced any other way, a record generated interactively carrying `policy_version: "none"` whose re-run from its seed under the auto policy yields a different character. The flag waives one check and no others: the verification is unchanged, a tampered record still fails, and the provenance identifiers are excluded from the record comparison only so that the mismatch the caller already acknowledged does not come back as the answer. The applied deviations are not among them — that list is generated content rather than an identifier, so a record written before a deviation was stamped differs there, and being told so is the point. It writes nothing: there is no upgrade path for an orphaned record, and there cannot honestly be one, because a different engine is what the version was reporting.

## JSON conventions

Characteristics stored numeric with the UPP hex string derived and stored alongside; money as integer credits; dates as Imperial calendar day/year with age in years. Skills and Knowledges are distinct entries. Derived values are stored and recomputed on replay.

The schema is `docs/character.schema.json`, draft 2020-12, with `docs/character.minimal.json` and `docs/character.complete.json` beside it, versioned by `schema_version`. It is precise about the envelope — every field's type, the vocabularies of event, consequence and benefit kinds, and the rule that an event carries exactly the payload its kind names — and deliberately loose about which payload fields each consequence kind uses, because `omitempty` makes those sets ragged and fifty-five branches in the schema would be a second copy of the code. That last rule is pinned in `docs` instead.

`schema_version` tracks **the shape of the records the engine writes**, not the precision of the document describing it. A constraint that narrows the schema to what the engine already produced is a clarification and does not bump; one that would invalidate a record the current engine writes is a bump. The case that settled it was the `upp` pattern: added after the clamp that made an unrepresentable UPP impossible, so every record engine 0.41.0 wrote was already inside it, and the only records it newly rejects are ones an older engine should not have written. Records carry the version, so the test is about them and not about the file.

Validation is a hand-written checker over the subset of JSON Schema the document uses, rather than a library: it is a few hundred lines against six third-party modules, in a repo that has none. The risk in writing one is that a validator with a bug passes everything, so it is not trusted on the fixtures passing — every rule the schema states has a record that must fail because of it, and each of the checker's keywords is mutation-tested.

## CLI sketch

```
t5chargen new [--seed N] [--auto] [--career scout] [--homeworld A867A69-F] [--name X] [-o file]
t5chargen batch --count 20 --auto [--career ...] [-o dir|file.jsonl]
t5chargen render [--history] character.json
t5chargen replay [--ignore-provenance] character.json
t5chargen version
```

`version` reports the build and the three versions a record stamps, so a bug report can be matched against any record the binary wrote. It is read from the build info the toolchain embeds, which means it names what the binary is rather than what a build flag was told to say.

Flags precede the filename. Go's `flag` package stops parsing at the first non-flag argument, so `render char.json --history` leaves `--history` standing as a second positional argument and exits with a usage error.

Interactive mode walks the E1 checklist step by step; auto mode applies the fixed default policy. `new` writes JSON to stdout unless `-o`; `batch` emits JSONL (or one file per character with `-o dir`), requires `--auto`, and derives each member's seed from the base seed + index, recorded in each record. `--career` forces the first career only. Existing files are never overwritten without `--force`. Interrupted interactive sessions produce no output file.

The auto policy is **total** (it can decide every valid choice point: education, career entry and retry, Risk/Reward, skills, continuation, career change, benefits), deterministic, and tie-breaks by first-listed order in Book 1. The full decision table lives in the tool repo's `POLICY.md`; `policy_version` identifies it in every record.

## Architecture notes

- Repo: new `philoserf/t5chargen` (or similar) — code does not live in this collection repo, per the world-builder precedent.
- Packages: `dice`, `chargen` (engine; consumes a `Decider` interface for all choice points), `career` (data-driven career definitions), `render`, `cmd/t5chargen`.
- Data/logic boundary: tables, thresholds, and labels are embedded data files; orchestration and career-specific exceptional mechanics are typed Go code. No rules language.
- Testing: golden character fixtures per career, replay round-trips, schema validation, and property tests on the dice engine. No throughput targets — a solo CLI emitting text records.

## Milestones

1. Dice engine + characteristics + character record/render, with the generation event log wired in from the start (end-to-end walking skeleton, one career: Citizen).
2. Education and homeworld skills.
3. All 13 careers with career-specific mechanics. Exit criterion: a living `COVERAGE.md` in the tool repo mapping every E1 step and career rule to its page cite, implementation, and golden test — no career is "done" until its uncommon branches are listed there under one of that document's statuses.
4. Aging, career changes, muster out, fame.
5. Interactive mode polish; batch mode; replay verification.
6. The rules milestone 5 left: the Rogue's previous-career Scheme (chart 10), and OTC and NOTC (p. 61).
7. The deferrals that were still real once milestone 6 closed: resigning from the Reserves (p. 67), Flight School (pp. 60-61), the Branch changes of I-34, the Scholar's rank titles, and the Skill/Knowledge distinction, which milestone 6 had scoped out on a premise that proved wrong.

_All seven are complete._ `docs/MILESTONE-6.md` and `docs/MILESTONE-7.md` record what each turned out to be, against what was planned here. Nothing this document asks for is outstanding but chart 11's `Capital***` cell, which `docs/COVERAGE.md` carries as its one accepted exception.

## Decisions (2026-08-19)

- **Names**: blank by default, `--name` flag to supply one. No generator in v1 — T5 core has no name-gen rules, and names are setting-dependent.
- **Auto-mode policy**: one fixed, documented default policy in v1. The engine takes a `Decider` interface (interactive and auto modes are its two implementations), so alternative policies are a later addition, not a rework.
- **Errata**: freeze on Print Edition 5.1 as held in the Traveller collection; every citation must be verifiable against those PDFs. Deliberate deviations (e.g. where the printed rule is defective) are recorded in the tool repo's `ERRATA.md`, never applied silently.

## Sources

- `~/Documents/Traveller/T5/Traveller5 Core Rules Book 1 Characters and Combat.pdf` (authoritative; chart pages cited above).
- `~/Documents/Traveller/T5/Archive/` chargen extracts (`Character Generation Checklist.pdf`, `CharGen Careers.pdf`, `CharGen Charts.pdf`, `Master Skills List.pdf`, etc.) for quick topic location only — they derive from the 2008 Preliminary CD-ROM edition, so any rule implemented from them must be verified against Book 1 Print Edition 5.1.
