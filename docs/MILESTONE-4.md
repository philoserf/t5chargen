# Milestone 4 — the rest of the lifepath

Complete. A character now runs from characteristics to muster out: aging
bounds the careers, career changes chain them, all thirteen charts are
implemented, Fame is calculated over the finished record, and muster out
counts up what he leaves with.

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

- All thirteen careers, 84 recorded interpretations, 34 choice points with
  a policy row each, 255 tests over 14 packages, 31 golden fixtures.
- Schema 0.24.0, engine 0.25.0, policy 0.16.0.
- No row in `COVERAGE.md` says `deferred (M4)`.

## Two things need a decision

**Birthdate (FR8) cannot be implemented from an authoritative source.** Its
only cite is the Archive, which the ground rules exclude, and the rulebook
sweep found no birthdate rule in Book 1. Aging does not need one — the
four-year cadence wants years-since-last-check, not a calendar date. The
recommendation is to amend FR8 rather than implement from a forbidden
source, and that is a PRD change rather than an engine one.

**Land Grant and Ship Share values stay uncomputed.** The counts are
recorded and chart M1's per-TC rate is transcribed, but a grant's income
needs the world it sits on and a share needs a ship from chart S — both
out of scope by decision, and the geodesic Land Grant work belongs to the
sibling `philoserf/traveller` repo.

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
