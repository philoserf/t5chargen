# Known limitations

What this tool does not do, or does differently from the printed rule.
Each entry says where the reasoning is written down rather than
restating it: [ERRATA.md](ERRATA.md) holds the readings, and
[POLICY.md](POLICY.md) holds what the auto policy decides.

Nothing here is a bug. A bug is a rule this implements wrongly; report
those.

## One rule is not implemented

**Chart 11's `Capital***` cell returns an error.** It awards "World
Knowledge (of world of highest held noble Land Grant)" (p. 85), and
nothing in a character record says which of his grants is highest —
ranking them needs a per-title hex table the book does not print
([I-83](ERRATA.md)). Selecting the cell fails rather than awarding
something invented. It reaches you only if you play a Noble
interactively and pick that cell; the auto policy never does.

## Two rules depart from the page, and records say so

Both are stamped into the record's `errata` field, so a character always
names the readings it was produced under.

- **[I-112]** A World Knowledge counts the terms from age 2 to the age
  career resolution began, not the whole life p. 134's example counts.
  This engine does not know where a character lives once a career has
  him, so it counts only the years it can vouch for. A character who
  genuinely never left home gets less than the printed rule gives.
- **[I-82]** A Land Grant hex on a world the record does not name is
  priced at the no-TC floor, as the book prices its own unnamed
  companion world. "A world with no TCs" and "a world whose TCs are
  unknown" are not the same claim, and this asserts the first knowing
  only the second.

## One skill is awarded whole

**Musician does not expand into Knowledges** ([I-111]). p. 134 makes the
first two receipts of a container skill award a contained Knowledge
instead, and Musician is a container with no instrument list printed
anywhere in Book 1. Rather than invent one, it is awarded as a skill.
Language is excepted by p. 134 itself and behaves the same way.

## The auto policy declines things a player may take

`--auto` applies a fixed, documented decision table
([POLICY.md](POLICY.md)) which tie-breaks by first-listed order. Several
rules are therefore implemented and tested but never reached by an
auto-generated character. Answer the choices yourself — `t5chargen new`
without `--auto` — to reach them.

An auto character never:

- changes careers, so he has exactly one, and **Craftsman and
  Functionary are unreachable** in auto mode — neither can be a first
  career (p. 63);
- resigns from the Reserves, attends Flight School, or changes Branch;
- is commissioned through OTC or NOTC — a commission obliges a term in
  the service (p. 61), which would send every auto character who
  attends College into Soldier;
- takes the Benefits column at muster out, so he leaves with money and
  passages and **no Knighthood**;
- takes Later Education except to climb from a degree he already holds,
  so the ED5 and Trade School end of chart C is not exercised in auto
  mode.

A Citizen can serve into his hundreds. Aging bounds the tail (chart A,
p. 89), but chart 04's fixed "Continue 10-" does not degrade as
characteristics fall. That is the printed rule, not a runaway loop.

## What is out of scope

Set by the PRD's non-goals, not by omission: non-human sophonts and
characteristic variants, psionics, clones and robots, combat, equipment
beyond mustering-out benefits, in-play advancement, and world
generation — supply a UWP or take the default.

[COVERAGE.md](COVERAGE.md) records every rule this implements, rule by
rule, with the page it is on and the test that holds it.

[I-82]: ERRATA.md
[I-83]: ERRATA.md
[I-111]: ERRATA.md
[I-112]: ERRATA.md
