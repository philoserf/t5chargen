# POLICY.md — auto-mode default policy

Version: **0.1.0** (`policy_version` in every character record). Changing any
rule here is a policy version bump (docs/PRD.md, Replay and provenance
contract).

The auto policy is total (it can decide every valid choice point),
deterministic, and tie-breaks by first-listed order in Book 1 (docs/PRD.md,
CLI sketch). The engine presents each choice's options in the order the rule
lists them; the policy returns an index.

## Decision table

| Choice point                        | Rule                                                                    | Rationale                                                                                                                                                                                                                                                            |
| ----------------------------------- | ----------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `select_career`                     | First-listed available career.                                          | Tie-break rule; with a `--career` force the list holds only the forced career.                                                                                                                                                                                       |
| `select_controlling_characteristic` | Highest-valued available characteristic; ties break to first-listed.    | Maximizes the roll-low success chance of the term's Citizen Life (or, later, Risk/Reward) throw.                                                                                                                                                                     |
| `select_skill_column`               | The **General** column; first-listed if a career has no General column. | The only Citizen column (chart 04 table C, p. 78) where every row yields a plain skill regardless of education (Academic rows are lost without a Major/Minor) or caste; Personal would trade skills for characteristic bumps, which is a poor default for bulk NPCs. |
| `select_hobby`                      | First-listed table E entry (currently `ACV`).                           | Pure tie-break: the book gives no ranking for the 100+ alternatives. A smarter hobby heuristic is a future policy version.                                                                                                                                           |

## Known limitations (0.1.0)

- Every auto-generated Citizen's hobby is the first-listed table E entry.
- Careers run until the Continue roll fails; with aging deferred to
  milestone 4, long careers produce old characters. Rule-accurate, and
  bounded by the Skill-15 cap (p. 134).
