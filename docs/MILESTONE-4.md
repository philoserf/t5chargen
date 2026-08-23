# Milestone 4 — the rest of the lifepath

The lifepath is implemented end to end. A character runs from
characteristics to muster out: aging bounds the careers, career changes
chain them, all thirteen charts are in, Fame is calculated over the
finished record, and muster out counts up what he leaves with.

**Closed against the spec**, on the second attempt. This document called
the milestone complete once, was corrected to say it was not — two PRD
requirements were unmet, FR7's ship shares and land grants and FR8's
birthdate — and is now closed for the reason that should have applied the
first time: the rules are implemented. The story of the second attempt is
below, under _What the two outstanding rules turned out to be_, and it is
mostly about a rulebook sweep that missed two pages.

Cites are to Book 1, Print Edition 5.1.

## What shipped

Twelve pull requests, in the order they landed.

|                        |                                                                                                                    |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------ |
| Golden regeneration    | `task goldens`, because twenty-five fixtures move repeatedly and hand-copying is where an unreviewed byte slips in |
| Age advance            | one site instead of nine, and the fix that a term ending in death or disability still costs its four years         |
| Sanity                 | chart 05's Scout rule recorded as a pending modifier, since p. 52 withholds the value                              |
| Aging (chart A)        | Life Stages, `2D < Life Stage`, the zero cascade — and death that ends the lifepath                                |
| Career changes (p. 66) | the Reserves, and the one line that held a character to a single career                                            |
| Functionary (chart 13) | Office Politics, which is the term and the Continue both                                                           |
| Craftsman (chart 01)   | Master Points and Masterpieces — the thirteenth career                                                             |
| Fame (chart F)         | calculated, not accumulated                                                                                        |
| Chart M1               | the muster-out benefit vocabulary, typed                                                                           |
| The thirteen table Ds  | plus a guard pinning the chart M2 conflict                                                                         |
| Muster out, benefits   | the rolls and what they award                                                                                      |
| Muster out, the rest   | Automatics, Entitlements, and the Rogue's payoff                                                                   |

Where the plan said ten chunks, twelve shipped: the muster-out data split
at the vocabulary line and the procedure at the automatics line, both
because they were two reviewable things rather than one.

## Where it stands

- All thirteen careers, 86 recorded interpretations, 34 choice points with
  a policy row each, 273 tests over 16 packages, 31 golden fixtures.
- Schema 0.27.0, engine 0.28.0, policy 0.16.0.
- No row in `COVERAGE.md` says `deferred (M4)`, and every PRD requirement
  from FR1 to FR10 is implemented except the three that name milestone 5.

## What the two outstanding rules turned out to be

This section said, for one day, that both were decisions rather than work:
that FR8's birthdate could not be implemented from an authoritative source,
and that Land Grant and Ship Share values were out of scope. Neither held up.
The sweep behind those claims missed pages, and the record here was wrong
long enough to be quoted back in an outside review. Both are implemented
now; what follows is what the rules actually said.

**Book 1 prints the birthdate rule, twice.** p. 58 ("Date of Birth") sets the
default current date at 001-1105 and says to subtract the character's age
from it; p. 263 ("Birthdates") gives the Birth Date Generation table — four
consecutive dice into a 365-day table, rerolling on `RR` — and the worked
example that produces Wonday 044-1075 for an age-30 character. FR8 cites the
Archive, and a bad cite is not the same thing as no source. All 432 cells of
the table are transcribed and checked to cover the year exactly once, and all
three of the page's worked examples are pinned. It consumes dice, so every
golden moved. The Aging schedule is untouched: p. 58 defers the calculation
to the end of character generation, after the last Aging Check has been
thrown, which is why interpretation I-50 survives losing its stated premise.

**Land Grant income is printed and computable from what the record already
holds.** p. 88: "An unimproved Land Grant generates income based on the Trade
Classifications of the world and equal to Cr10,000 per TC annually (equal to
Cr5,000 if there are no TCs)," and "The first hex in any grant is on the
Noble's homeworld." Every character carries his homeworld's Trade
Classifications, so the stated blocker — a grant needs a world we do not have
— was false for the one hex the book sites on a world we do. p. 41 adds a
companion hex per mainworld hex, and p. 88's own example prices exactly that
pair: Cr20,000 for Sir Richard's two-TC homeworld hex, Cr5,000 for the
companion minor world. The rate has been sitting in `benefit/data/benefits.json`
as `credits_per_tc` since chart M1 landed, read by one test and no production
code — the same transcribed-but-unwired shape this milestone had already hit
three times, and the reason a gate for it landed alongside.

**Ship Shares genuinely have no credit value, and that is the finding.** Book
1 attaches no Cr figure to a share anywhere; one share buys 50 tons of ship
from chart S (p. 90), so a 200-ton Free Trader takes four. FR7 asks for a
value the book declines to give, and the honest close is to transcribe chart
S and record what the shares reach rather than invent a price. Transcribing
it turned up a second thing: a character's shares arrive from two places, his
career Rewards and the Benefits column, and nothing added them together.
Redemption itself stays out — I-64 forecloses ownership at muster out on
Fame-ordering grounds, and p. 90 says shares "may be saved for some future
use".

## What closing it took

Five more pull requests after the twelve above.

|                    |                                                                                      |
| ------------------ | ------------------------------------------------------------------------------------ |
| The corrections    | the false "Book 1 prints no birthdate rule", in four documents and an interpretation |
| Land Grant income  | priced per grant, and the Scout finally recorded the grants chart 05 always gave him |
| Chart S            | the seven ships, and the two share ledgers pooled                                    |
| The birthdate      | 432 cells, four dice, and the last step of the checklist                             |
| The dead-data gate | 204 chart fields, 17 unread, every one of them accounted for on purpose              |

Three things are worth keeping from how it went.

**A negative finding needs the same evidence as a positive one.** "Book 1
prints no birthdate rule" was recorded in four documents and built into an
interpretation, and it was simply not checked. It cost a wrong milestone
record, a wrong ERRATA premise, and an outside reviewer's time.

**The recurring defect had a shape, and the shape had a gate.** Four fields
were transcribed, validated at load, and read by nothing. The gate that now
catches them caught five more of my own from the chunk before it, and then
caught me again on the first change I made after it landed.

**Two guesses failed and both were worth the failure.** A bound on the
duplicate-benefit reroll hung the suite before it was right, and an invariant
on chart C's knowledge-only rows failed on real data because the flag turned
out to be orthogonal to the columns rather than a substitute for them. The
second left a genuine open question, recorded in `COVERAGE.md` rather than
quietly excused: whether education may award a bare container skill.

The per-title hex table on p. 88 (Gentleman one hex through Emperor 256) is
deliberately not applied — it keys hexes on title where I-30 already chose
box A's per-Soc-increase count, and reconciling the two reopens a settled
interpretation.

## What the auto policy will not show you

Three limitations are the documented tie-break working as written, not
defects, and each is recorded in `POLICY.md`:

- No auto-generated character changes careers, so every one has a single
  career and Craftsman and Functionary are unreachable. Their fixtures are
  generated with a test Decider and carry `policy_version: "none"`.
- No auto-generated character takes the Benefits column at muster out, so
  none receives a Knighthood or a characteristic improvement. They muster
  out with money and passages.
- Chart 04's fixed "Continue 10-" does not degrade as characteristics
  fall, so a Citizen can still serve into his hundreds until aging kills
  him. Faithful to the page; the real bound is voluntary muster out.

## Still deferred

Later Education (it suspends a career term, which interactive mode is the
place for); chart 11's Capital cell and the Rogue selecting a previous
career as a Scheme, both of which need scope this milestone declined;
Clone Aging and the Caste clause, outside the human core lifepath; QREBS
allocation and Vintage Masterpiece appreciation, which price a sale that
generation never makes.

## Not in this milestone

Interactive mode, batch generation, the replay verifier, and the formal
JSON Schema are milestone 5. The schema's deferral turned on the record
shape settling, and muster out was the last thing to disturb it — so that
is now unblocked.
