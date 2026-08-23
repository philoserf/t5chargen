# Milestone 4 — the rest of the lifepath

Milestone 3 closed with every career reachable from a standing start
implemented. What remains between a career record and a finished character
is this milestone: aging, muster out, career changes, and fame.

Cites are to Book 1, Print Edition 5.1. Every page named here was read
before this plan was written; nothing below is from memory.

## The four workstreams

### 1. Aging — chart A, p. 89

The chart is complete and unambiguous, which makes it the right place to
start.

- Life Stages: nine stages of eight years after a two-year infancy, so
  Young Adult begins at 18 and Retirement at 66.
- Physical Aging (Str, Dex, End) begins at 34, Life Stage 5; Mental Aging
  (Int) at 66, Life Stage 9. Both resolve as an Aging Check every four
  years on the character's birthday.
- The Aging Check is `2D < Life Stage`, rolled per applicable
  characteristic, and the character wants to **fail** it. Note the strict
  less-than: it is not the roll-low Check the rest of the engine uses, so
  it wants its own dice helper rather than a target fudged by one.
- An effect reduces the characteristic by one. One characteristic at zero
  resets to 1; two, a major illness and four weeks; three, an extremely
  major illness and four months — and the **second** time three reach
  zero, the character dies.

Doing this first is worth stating plainly: careers currently run until a
natural 12 fails a Continue throw, which has produced characters aged 234
and 402 in sweeps. Aging is the printed bound on that, and every golden
in the repo will move when it lands.

Deferred within this workstream: Clone Aging, which is a Relict/Guest/Med
mechanic outside the human core lifepath.

### 2. Muster out — chart M1, p. 70, plus each career's table D

Chart M1 was transcribed for its Medals table already; the rest of the
page is sighted and recorded in COVERAGE. It carries the Automatics list,
the Financial and Non-Financial benefit split, the Life Stage 9
Entitlements, the Armed Forces retirement rates, and the Forbidden
Knowledge table.

Every career has its own table D, none of which is transcribed yet — that
is thirteen tables of money and benefits, plus chart 11's third "Proxy"
column, which no other chart has.

This workstream also owes several debts already recorded against it:

- **Double Benefits** on mustering out disabled, promised by charts 02,
  05, 06, 08, 09, 12 and the p. 65 injury rule.
- The Scout's **Land Grant** economics and the Noble's **Land Grant
  hexes**, both of which need worlds, and chart 11's "Capital" skills cell
  which reads the world of the highest-held grant.
- The Rogue's **Payoff** and the Merchant's **Ship Shares**, both counted
  today with no way to spend them.

### 3. Career changes — p. 66

"A character may avoid the Continue roll (and its possibility of Mandatory
Continue) by voluntarily ending his service in the current career and
selecting a different career for which he is eligible. The decision to
change careers is irrevocable and must be made before attempting begin the
new career." A Functionary or Noble cannot change; no character may change
_to_ Citizen.

This unlocks the two careers milestone 3 could not reach, and both are
waiting on exactly this:

- **Craftsman** (chart 01) — "Automatic\* — \*if TWO skill-6 and
  Craftsman-1", which a character leaving education at 18 essentially
  never satisfies. Its Masterpiece machinery, the 9D Master Points throw,
  the box-versus-prose conflict between "9D < Master Points" and "Roll 9D
  for Master Points or less", and QREBS allocation are all banked in
  COVERAGE.
- **Functionary** (chart 13) — "is never a first career". Its skills table
  is already transcribed as a reference career for the Agent's Undercover
  table, so only its box A and Office Politics remain.

It also makes reachable a pile of rules that are currently unreachable
prose: the Rogue's "select any previous career" for a Scheme, the
Reserves, and the Service Academy's Officer1 entry linkage.

### 4. Fame — chart F, p. 91

**Fame is calculated, not accumulated.** This is the finding that most
changes the shape of the work, and it contradicts what the engine does
today. Chart F: "Current Fame for an individual is based on a variety of
accomplishments. For example, Rogue with one Failed Scheme (and no other
applicable factors) has Fame = 1 x 3 = 3."

The eligibility table gives a multiplier per career: Craftsman
Masterpieces x3 and Perfect Masterpieces x5; Scholar = Rank and =
Publications; Scout Discoveries **x4**; Merchant = Rank, Ship Owner = 1D;
the Armed Forces by Officer Rank with medals multiplied (XS x0, WB x1,
MCUF x1, MCG x2, SEH x3, \*SEH\* x4) and **enlisted service earning
none**; Agent = number of Commendations; Rogue Successful Schemes x2 and
Failed Schemes **x3**; Imperial Noble Soc x1.5 and +1 per Exile. The
Entertainer is "detailed under Career", so chart 03's tracked Fame is
already correct. Fame stacks to 20, beyond which only the highest applies.

Three careers currently write a running `Character.Fame` counter: the
Scout adds 1 per Discovery where chart F says x4, and the Rogue adds 1 per
failed Scheme where chart F says x3. The Entertainer sets it outright,
which chart F endorses.

The likely reconciliation — to be confirmed against the page when the work
starts, not assumed now — is that a career chart's "Fame +1" records _one
occurrence_ and chart F supplies the multiplier, which is exactly how the
chart's own worked example reads. Under that reading the counter becomes a
derived value and the per-career increments become occurrence counts the
records already hold.

Two smaller pieces belong here: the Fame Flux Event, which any character
may invoke once, and the Agent's Commendation value, `N = CC - Reward
roll`, which the engine counts but does not yet compute.

## Sequencing

1. **Aging.** Self-contained, bounds every career, and every golden moves
   once — better to absorb that before muster out adds more.
2. **Fame.** Replaces the running counter with chart F's calculation, and
   resolves a collision three careers have each deferred to it.
3. **Career changes.** Unlocks Craftsman and Functionary and turns several
   pieces of unreachable prose into reachable rules.
4. **Craftsman and Functionary**, which only exist once (3) lands.
5. **Muster out.** Largest and last: thirteen benefit tables plus chart M1,
   and it consumes what the earlier steps produce — Land Grants, Ship
   Shares, Payoffs, Double Benefits, retirement by rank and terms.

## Open questions to resolve before the relevant chunk

- **Sanity.** Chart 05 gives the Scout "reduce San = -1 for each TWO Terms
  served", deferred on the grounds that chart A would define San. Chart A
  as printed does not mention it. Locate San before the aging chunk and
  either implement or re-record the deferral honestly against the right
  page.
- **Birthdate** (FR8) is unimplemented and belongs with aging, since aging
  resolves "on the character's birthday".
- **The Continue bound.** Once aging lands, the long-career tail should
  disappear on its own. If it does not, the 2..11 validation on fixed
  Continue targets and the unbounded characteristic and value targets
  deserve a second look together.

## Not in this milestone

Interactive mode, batch generation, the replay verifier, and the formal
JSON Schema are milestone 5. The schema's deferral is dated in
docs/PRD.md and turns on the record shape settling, which muster out is
the last thing to disturb.
