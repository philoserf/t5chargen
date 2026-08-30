# Beta readiness

The project is technically ready to begin a beta cycle. The alpha already
has strong correctness, provenance, and release discipline. The main gap is
exactly what the alpha release record says: real-world use, compatibility
evidence, and distribution, rather than missing core rules.

At the start of this review, all 17 packages passed under the race detector.
Overall statement coverage was 88.6%, including 92.5% for `chargen`, 86.2%
for the CLI, 90.7% for rendering, and 97.6% for dice.

## Recommended path to beta

### 1. Run an intentional alpha feedback cycle

**Partly done** — the three issue templates exist
(`.github/ISSUE_TEMPLATE/`), and the rules template asks for the version,
record JSON, expected rule with its page, and observed result. Recruiting
testers and running the cycle is the part no amount of tooling does.

Recruit several Traveller5 users, including at least one rules expert and
one person unfamiliar with the project. Ask them to:

- install without maintainer assistance;
- generate automatic and interactive characters;
- render and replay saved records;
- deliberately enter invalid inputs;
- compare at least one character against Book 1;
- submit the record with every bug report.

Create issue templates for rules discrepancies, CLI problems, and
record/replay failures. A rules report should request the tool version,
record JSON, expected rule, page citation, and observed result.

This should be the primary beta gate. More internal implementation work
cannot substitute for it.

### 2. Define record compatibility before another engine release

The project carefully versions records, but deliberately has no upgrade path
for records rejected by newer provenance checks. Before beta, decide and
document:

- whether beta releases promise to render older records;
- whether replay requires the original engine version;
- how users obtain an older executable;
- how long prerelease artifacts remain available;
- what constitutes a breaking schema change.

Add a compatibility corpus containing records from every released version.
Each new release should prove the intended operations against that corpus.
This is more important than pushing raw coverage higher.

### 3. Test the supported installation surface

**Done** — `.github/workflows/ci.yml` gained a `smoke` matrix over macOS
and Linux against both the Go version `go.mod` declares and the current
release; README declares Windows unsupported. Installation from the
tagged module is proved by the release workflow, which builds the
artifacts that way.

CI currently tests one Linux environment and the Go version declared in
`go.mod`. Before beta, add lightweight smoke jobs for:

- macOS and Linux;
- the minimum supported Go version and the current Go version;
- installation from the tagged module, not only repository builds.

Windows is optional, but either test it or explicitly say it is unsupported.
The requirement of Go 1.26.6 is also a meaningful adoption constraint; keep
it only if the code needs it.

### 4. Automate release artifacts when users need easier installation

**Done** — `.github/workflows/release.yml`. A pushed version tag runs the
gate on the tagged commit, builds four platform binaries, checksums them
and opens a draft release. The notes stay manual, which is the one part
of a release a person has to write.

The release procedure explicitly postpones binaries and release automation.
That was sensible for alpha. For beta, signed or checksummed binaries for
macOS and Linux would widen the tester pool and make reports easier to
reproduce.

A release workflow should build from the tag, run smoke tests, produce
checksums, and create the prerelease. Keep the citation check manual because
the source PDF cannot enter CI.

### 5. Harden externally supplied data

**Done** — four fuzz targets over UWP parsing, the eHex digits, render
and replay, run by `task fuzz` rather than by the gate. All four ran
clean at millions of executions. Interactive input and extreme
`batch --count` remain covered by the existing hostile-input tests
rather than by fuzzing.

The highest-value additional testing is fuzzing rather than more example
tests. Good targets are:

- malformed character JSON;
- schema validation;
- `render` and `replay`;
- UWP and trade-classification parsing;
- interactive input;
- extreme `batch --count` values and unusually long-lived characters.

The existing hostile-input tests are a good foundation. Add bounded fuzz
tests and ensure malformed records cannot panic, allocate without practical
bounds, or leave partial output behind.

### 6. Improve beta-facing product ergonomics

**Done** — `t5chargen help` carries examples, troubleshooting, the
report-a-problem link and stability expectations; README gained Stability
and Report a problem sections; CHANGELOG.md starts at the first release.
The `--auto` explanation lives in help text rather than in a prompt,
because a prompt is recorded and replay-compared.

The README explains the product well, but beta users will need:

- `--help` examples and troubleshooting;
- a short "report a problem" link;
- documented stability expectations;
- a concise changelog or release-notes history;
- explicit support-platform language;
- a friendlier explanation of why interactive-only branches are absent
  under `--auto`.

Do not add new Traveller mechanics during this cycle unless alpha users
consistently request the same feature. Stability and usability feedback
should remain distinguishable from scope expansion.

## Small issues to fix now

All three are fixed; the entries stay so the record shows what was found.

- ~~`.githooks/pre-push` incorrectly says the repository has no CI.~~ It now
  describes itself as the earlier of two identical gates.
- ~~`docs/RELEASING.md` hardcodes the first alpha tag in reusable
  commands.~~ The tagging and install blocks take a `version` variable.
- ~~`docs/RELEASE_READINESS.md` says tagging is pending immediately before
  recording that it was tagged.~~ Item 10 records the tag and date.
- Coverage is already sufficient. The lower percentages in `skill`,
  `lifestage`, and data packages are not automatically beta blockers. Review
  uncovered branches by risk rather than imposing a global percentage
  target.

  Confirmed since: measured cross-package rather than per-package
  (`go test -coverpkg=./...`), the module is at 90.3%, and the per-package
  figures above are low only because each package's accessors are exercised
  from `chargen` rather than from its own tests. The uncovered remainder is
  almost entirely `validate*` error branches that a valid chart never
  reaches. One genuine gap was found and closed — `awardMajorOrMinor` had no
  test, and the COVERAGE.md row for it cited a golden that never reached it.

## Suggested beta exit bar

Call the project beta-ready when:

- several independent users have completed the core workflows;
- reported rules discrepancies are resolved or documented;
- released-version records form a compatibility test corpus;
- installation and smoke tests pass on supported platforms;
- malformed external records have fuzz coverage;
- support and compatibility promises are written down;
- the known limitations remain accurate;
- the release is reproducible from its tag.

The core engine does not look like the weak point. The next milestone should
prove that strangers can install it, trust it, preserve their records, and
report failures, rather than adding another large rules milestone.
