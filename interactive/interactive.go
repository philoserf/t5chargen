// Package interactive resolves choice points by asking a person, the
// other of the two Decider implementations docs/PRD.md names: "Two modes:
// interactive (player makes each choice) and auto (tool decides)".
//
// It writes numbered prompts to a Writer and reads answers from a Reader,
// so the whole of it is testable with a scripted script and no terminal.
// A terminal front end is a view over this, not a replacement for it: the
// answers it produces are indices into the same option lists the auto
// policy indexes, which is what makes an interactive record replayable.
package interactive

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/philoserf/t5chargen/chargen"
)

// ErrAbandoned reports a session the player left rather than finished.
// docs/PRD.md's CLI sketch says an interrupted interactive session
// "produce[s] no output file", and this is how the engine is told: it is
// distinct from an out-of-range answer, which is a decider that replied
// wrongly rather than one that stopped replying.
var ErrAbandoned = errors.New("interactive session abandoned")

// pageSize is how many options are listed before the prompt asks for a
// filter instead. The longest list the engine presents is the Master Skill
// List at over a hundred entries, which is not a menu.
const pageSize = 12

// The single-character answers that are not option numbers.
const (
	quitAnswer = "q"
	listAnswer = "?"
)

// Decider asks a person to resolve each choice point.
type Decider struct {
	in  *bufio.Scanner
	out io.Writer
}

// New returns a Decider reading answers from in and writing prompts to
// out.
func New(in io.Reader, out io.Writer) *Decider {
	return &Decider{in: bufio.NewScanner(in), out: out}
}

// Kind identifies the player in choice events (docs/PRD.md FR10: "who
// decided — player or policy").
func (*Decider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// Choose puts the choice to the player and returns the index answered.
//
// A choice with one option is answered without asking. The engine presents
// some choices that way as a seam rather than a decision — an assigned
// homeworld is the standing example — and asking a player to confirm the
// only thing he may do is not a question.
func (d *Decider) Choose(c chargen.Choice) (int, error) {
	if len(c.Options) == 1 {
		fmt.Fprintf(d.out, "\n%s\n  %s\n", c.Prompt, c.Options[0])

		return 0, nil
	}

	d.present(c, "")

	for {
		answer, err := d.read()
		if err != nil {
			return 0, err
		}

		switch {
		case answer == "" || answer == listAnswer:
			d.present(c, "")
		case strings.EqualFold(answer, quitAnswer):
			return 0, fmt.Errorf("%w at %q", ErrAbandoned, c.ID)
		default:
			if index, ok := parseIndex(answer, len(c.Options)); ok {
				return index, nil
			}

			d.filter(c, answer)
		}
	}
}

// read takes one line of input, treating the end of it as abandonment: a
// player who closed the session did not answer, and the engine must not be
// told that he did.
func (d *Decider) read() (string, error) {
	fmt.Fprint(d.out, "> ")

	if !d.in.Scan() {
		if err := d.in.Err(); err != nil {
			return "", fmt.Errorf("reading the answer: %w", err)
		}

		return "", fmt.Errorf("%w: input ended", ErrAbandoned)
	}

	return strings.TrimSpace(d.in.Text()), nil
}

// parseIndex reads an answer as a 1-based option number.
func parseIndex(answer string, options int) (int, bool) {
	n, err := strconv.Atoi(answer)
	if err != nil || n < 1 || n > options {
		return 0, false
	}

	return n - 1, true
}

// filter shows the options matching what the player typed, or says that
// none do. Nothing is selected by matching alone, however few match: the
// numbers stay the option's own, so an answer means the same thing whether
// it was typed against the whole list or a filtered view of it.
func (d *Decider) filter(c chargen.Choice, term string) {
	if d.present(c, term) == 0 {
		fmt.Fprintf(d.out, "  nothing matches %q. Type %s to list the options again.\n", term, listAnswer)
	}
}

// present writes the prompt and the options matching term, and reports how
// many matched. An empty term matches everything.
func (d *Decider) present(c chargen.Choice, term string) int {
	fmt.Fprintf(d.out, "\n%s\n", c.Prompt)

	if c.Cite != "" {
		fmt.Fprintf(d.out, "  %s\n", c.Cite)
	}

	shown, matched := 0, 0

	for i, option := range c.Options {
		if term != "" && !strings.Contains(strings.ToLower(option), strings.ToLower(term)) {
			continue
		}

		matched++

		if shown >= pageSize {
			continue
		}

		shown++

		fmt.Fprintf(d.out, "  %3d  %s%s\n", i+1, option, score(c, i))
	}

	if matched > shown {
		fmt.Fprintf(d.out, "  ... %d more. Type part of a name to narrow the list.\n", matched-shown)
	}

	fmt.Fprintf(d.out, "  (a number to choose, text to search, %s to list, %s to abandon)\n", listAnswer, quitAnswer)

	return matched
}

// score renders the engine's decision aid for an option, where it offers
// one and says what it means. Scores are not part of the printed rule and
// are not recorded (chargen.Choice); they are shown because the engine has
// already worked out a number the player would otherwise be reaching for.
//
// An unlabelled score is not shown. "1" against an education programme
// means "you meet its prerequisite", and a player cannot be expected to
// guess that from the digit; a number nobody can read is worse than no
// number, because it looks like information.
func score(c chargen.Choice, i int) string {
	if i >= len(c.Scores) || c.ScoreLabel == "" {
		return ""
	}

	return fmt.Sprintf("  [%s %d]", c.ScoreLabel, c.Scores[i])
}
