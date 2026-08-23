# Milestone 4 — the rest of the lifepath

Milestone 3 closed with every career reachable from a standing start
implemented. What remains between a career record and a finished character
is this milestone: aging, career changes, the last two careers, fame, and
muster out.

Cites are to Book 1, Print Edition 5.1. Every page named here was read
before this plan was written; nothing below is from memory.

## Three findings that shape the work

**Sanity is on p. 52, not chart A.** Chart 05 gives the Scout "reduce San
= -1 for each TWO Terms served", deferred in COVERAGE on the grounds that
chart A would define San. Chart A (p. 89) never mentions it; the CS Sanity
section on p. 52 does, along with 2D generation, the Check, and the
zero-is-insanity clause. Two COVERAGE rows are wrong as a result: the one
citing "chart A defers it", and the one bundling Sanity with the PRD's
Psionics non-goal, which does not mention Sanity at all.

**Printed p. 71 is a second chart, M2 Muster Out Tables** — a consolidated
reprint of all thirteen career table Ds that contradicts six career pages
(Citizen shifted a row, Scholar and Functionary each gain a twelfth row,
the Entertainer differs in both row count and DM divisor, and Rogue and
Noble read "+Total Terms" against M2's "+Terms").

**`Character.Age` is write-only in the engine.** The only read anywhere is
the renderer printing it. No throw depends on age today, which is what
makes the age restructure aging requires provable rather than merely
tested.

## The authority question

Where chart M2 disagrees with a career page, **the career page governs**.
The tiebreaker is printed prose on p. 67, not a preference: "Each career
is fully described on its own comprehensive page. Once the career is
selected, turn to that page and resolve it according to the rules on that
page." M2 is a convenience reprint.

Each career's table D is therefore transcribed into that career's own JSON
file, alongside the rest of its chart, and a loader guard test asserts the
divergence set against M2 is exactly the six documented conflicts — so a
seventh, or a silently resolved one, fails the build rather than passing
unnoticed.

The counter-argument is real and belongs in the record: p. 68 _defines_
"+Terms" as terms in that career, with a worked example, while "+Total
Terms" is defined nowhere. The uniform rule is still preferred, because
honouring the career page for rows and abandoning it for DMs invites
exactly the per-table arguing the rule exists to prevent.

## Sequence

Aging first; muster out last. The middle ordering puts fame **after**
Craftsman, which is a change from this document's first draft: chart F
multiplies Craftsman Masterpieces, so fame written earlier would be
written against counters that do not exist, and p. 68's "one additional
roll if Fame 19+" makes fame an input to muster out. Fame late satisfies
both.

1. **Golden regeneration.** Twenty-five fixtures move repeatedly this
   milestone and regeneration is manual today. A `-update` flag on the
   existing comparisons, reusing the same serialization expression so a
   formatting divergence is impossible by construction.
2. **Centralize age advance.** `Age += TermYears` lives inside
   `continueRoll`, which is skipped when a character dies or is disabled,
   so those terms elapse zero years. Both paths always end the career, so
   appending the missing advance renumbers nothing: exactly six fixtures
   move, and any other movement is a bug.
3. **Sanity.** Record the modifier; do not roll the value. p. 52 forbids
   generating it before it is called for, and rolling would consume stream
   for a mechanic v1 never uses.
4. **Aging** (chart A, p. 89). Life Stages, the `2D < Life Stage` check
   the character wants to fail, the reduction, and the zero cascade. Needs
   a strict-less-than dice helper: the package has `<=`, `<=`-with-
   auto-failure, and `>=`, but no `<`. This is where age stops being
   write-only.
5. **Career changes** (p. 66) and the Reserves. The plumbing is three
   levels deep, and `character.go`'s `if err != nil || began` is the single
   line confining a character to one career.
6. **Functionary** (chart 13). Its skills table is already transcribed as
   a reference career for the Agent's Undercover table, so only box A and
   Office Politics remain.
7. **Craftsman** (chart 01). Untranscribed entirely: the 9D Master Points
   throw and its box-versus-prose conflict, QREBS, Masterpieces, and the
   "New Trade" cells.
8. **Fame** (chart F, p. 91). Fame is calculated, not accumulated — the
   chart's own example is "Rogue with one Failed Scheme ... has Fame = 1 x
   3 = 3". `Character.Fame` becomes derived; `CareerRecord.Fame` stays,
   because it is the Entertainer's tracked value and its Continue target,
   and chart F defers to it.
9. **Muster out, data.** Split at the vocabulary line: chart M1 and the
   typed benefit vocabulary first, then the thirteen table Ds transcribed
   against it. A reviewer of the transcription then has no design question
   to adjudicate.
10. **Muster out, procedure.** One step, post-lifepath — p. 68's rule that
    Functionary terms join an _earlier_ career's DM means a career's roll
    depends on careers served after it, and the worked example runs three
    careers and musters out once.

## Deferred, and why

- **Birthdate** (FR8) cannot be implemented from an authoritative source.
  Its only cite is the Archive, which the ground rules exclude, and the
  rulebook sweep found no birthdate rule in Book 1. Aging does not need
  it: the four-year cadence wants years-since-last-check, not a calendar
  date. FR8 should be amended rather than implemented from the Archive.
- **Land Grant hexes and ship selection.** Grants and shares are recorded
  and valued; the geodesic hex economics and chart S ship selection stay
  with the sibling `philoserf/traveller` repo.
- **Reserve resignation.** The default policy would never resign, so the
  Check would be dead stream in every generated character. It becomes a
  real choice in interactive mode.
- **Clone Aging**, a Relict/Guest/Med mechanic outside the human core
  lifepath.

FR4 also cites career changes as "Archive: `Changing Careers.pdf`"; the
rule is in Book 1 on p. 66 and the cite should move to the printed page.

## Not in this milestone

Interactive mode, batch generation, the replay verifier, and the formal
JSON Schema are milestone 5. The schema's deferral is dated in
docs/PRD.md and turns on the record shape settling, which muster out is
the last thing to disturb.
