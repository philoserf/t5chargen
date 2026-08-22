# POLICY.md — auto-mode default policy

Version: **0.5.0** (`policy_version` in every character record). Changing any
rule here is a policy version bump (docs/PRD.md, Replay and provenance
contract). 0.2.0 added the homeworld choice points; 0.3.0 the education
choice points; 0.4.0 the Scout career choice points (chart 05, p. 79) and
the Exploration fallback in `select_skill_column`.

The auto policy is total (it can decide every valid choice point),
deterministic, and tie-breaks by first-listed order in Book 1 (docs/PRD.md,
CLI sketch). The engine presents each choice's options in the order the rule
lists them; the policy returns an index.

## Decision table

| Choice point                        | Rule                                                                    | Rationale                                                                                                                                                                                                                                                            |
| ----------------------------------- | ----------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `select_career`                     | First-listed available career.                                          | Tie-break rule; with a `--career` force the list holds only the forced career.                                                                                                                                                                                       |
| `select_controlling_characteristic` | Highest-valued available characteristic; ties break to first-listed.    | Maximizes the roll-low success chance of the term's Citizen Life (or, later, Risk/Reward) throw.                                                                                                                                                                     |
| `select_skill_column`               | The first present of **General**, then **Exploration**, then **Business**; first-listed otherwise. | The only Citizen column (chart 04 table C, p. 78) where every row yields a plain skill regardless of education (Academic rows are lost without a Major/Minor) or caste; Personal would trade skills for characteristic bumps, which is a poor default for bulk NPCs. Business is Merchant's equivalent column (chart 06, p. 80), appended so the Citizen and Scout picks are unchanged. |
| `select_hobby`                      | First-listed table E entry (currently `ACV`).                           | Pure tie-break: the book gives no ranking for the 100+ alternatives. A smarter hobby heuristic is a future policy version.                                                                                                                                           |
| `select_homeworld` | The assigned homeworld (the supplied `--homeworld`, or the tool-owned default, Regina). | Assignment per docs/PRD.md FR2; random chart-B selection is an interactive/milestone-5 concern. |
| `select_art` | First-listed ("Actor"). | Tie-break rule; the book gives no ranking. |
| `select_trade` | First-listed ("Biologics"). | Tie-break rule; the book gives no ranking. |
| `select_education` | The college track: University, then College, then ED5; None otherwise. | Maximizes Edu gain for bulk NPCs. Service Academy is excluded because its Officer1 graduation links to milestone-3 military careers; Trade School/Apprenticeship trade Edu growth for narrow skills. |
| `select_service` | First-listed (Army). | Tie-break rule; unreachable while the policy never picks Service Academy. |
| `select_major` / `select_minor` | First-listed from the institution's Available Skills column (minor list excludes the major). | Tie-break rule; the book gives no ranking. |
| `select_check_characteristic` | Highest-valued of the stated characteristics; ties break to first-listed. | Maximizes the roll-low check chance, same rule as the controlling characteristic. |
| `attempt_honors` | Always attempt. | "Failure has no effect" (p. 59) — pure upside. |
| `attempt_waiver` | Always attempt. | The immediate stake (admission or reinstatement) outweighs the cost: each attempt worsens future waiver odds by Mod -1 (p. 59). |
| `select_begin_track` | First-listed. | The highest berth the chart offers: chart 06 lists "To Begin 4th Officer" first, which enters at officer rank M1 and so opens the Officer Promotion ladder. Failing it costs a year and the career (interpretation I-15). |
| `attempt_advancement` | Always attempt. | A commission or promotion attempt has no stated cost, and rank carries Auto Skills, an extra skill per rank gained (chart 06 table B), and a muster-out DM. |
| `select_skill` | First-listed. | Tie-break rule. Reached when a chart cell names more than one Master Skill List entry — the chart 04 table E "Grav" and "Spacecraft" cells (ERRATA.md I-10, I-11), whose options are listed in Master Skill List order — by the Apprenticeship award, which the policy never selects; and by the "One Art", "One Trade", "One Science", and "Starship Skill" cells, whose alternatives are the corresponding Master Skill List groups. |
| `select_duty` | Explorer Duty. | The Scout career's point, and the larger skill eligibility (chart 05 table B: Explorer 8 vs Courier 4); Courier's safety is an interactive-play trade-off. |
| `select_risk_mod` | No Mod. | Neutral default: Caution improves Risk but worsens Reward and vice versa (p. 65); a smarter Caution/Bravery heuristic is a future policy version. |
| `attempt_retry` | Always attempt. | The I-8 Reward retry has no stated cost. |

## Known limitations (0.4.0)

- Every auto-generated Citizen's hobby is the first-listed table E entry
  (excluding the determined Job, per ERRATA I-3).
- Careers run until the Continue roll fails — an unbounded geometric
  process, so long careers produce old characters until milestone 4 aging
  lands. Rule-accurate; note the Skill-15 cap (p. 134) bounds individual
  skill levels, not career length.
