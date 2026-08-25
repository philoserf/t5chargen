package interactive

// What a player is shown besides the questions.
//
// The engine records everything it does, and a Decider that also watches
// (chargen.Watcher) is handed each event as it happens. That is enough to
// follow the run: the checklist step it is in, the characteristics as they
// are rolled, the years as they pass. None of it is new information — it
// is the record, shown while it is being written rather than after.
//
// The shape follows the Master Chargen Checklist (chart E1, p. 72), which
// is what a player has beside him: lettered steps, in order, with the
// character's own numbers where the step needs them.

import (
	"fmt"
	"strings"

	"github.com/philoserf/t5chargen/chargen"
)

// characteristicOrder is the order chart A rolls them and the UPP prints
// them (p. 56).
var characteristicOrder = []string{"Str", "Dex", "End", "Int", "Edu", "Soc"}

// session is what the front end knows about the run so far.
type session struct {
	step  string
	age   int
	stats map[string]int

	skills    map[string]int
	told      bool // the characteristics have been shown
	abandoned bool // the player left rather than finished
}

// newSession starts with a character not yet rolled.
func newSession() *session {
	return &session{age: chargen.StartAge, stats: map[string]int{}, skills: map[string]int{}}
}

// Watch follows the run. It is called for every event the engine records,
// in order, and changes nothing: the front end reads the record as it is
// written.
func (d *Decider) Watch(event chargen.Event) {
	switch event.Kind {
	case chargen.EventStep:
		d.enterStep(event.Step.Name)
	case chargen.EventConsequence:
		d.session.apply(event.Consequence)
		d.announceCharacteristics()
	case chargen.EventThrow, chargen.EventChoice:
	}
}

// enterStep announces a step of the checklist as the engine reaches it.
func (d *Decider) enterStep(name string) {
	d.session.step = name

	fmt.Fprintf(d.out, "\n%s\n", Rule(name))
}

// apply folds one consequence into what the front end knows.
//
// Only the handful a header needs: the characteristics, the years, and the
// skills. Everything else the engine records is still recorded — it is
// simply not something a player needs on screen while answering.
//
//nolint:exhaustive // Deliberately partial: a header needs these and the rest are the record's business.
func (s *session) apply(c *chargen.ConsequenceEvent) {
	switch c.Kind {
	case chargen.ConsequenceCharacteristicSet,
		chargen.ConsequenceCharacteristicChange,
		chargen.ConsequenceAgingEffect,
		chargen.ConsequenceCharacteristicReset:
		// The reset belongs with the rest: "If one Characteristic is
		// reduced to 0, it is reset to 1" (p. 89 chart A) lands as its
		// own consequence after the aging effect that zeroed it, and a
		// header that stopped at the effect would show a 0 the
		// character does not have.
		if c.Characteristic != "" {
			s.stats[c.Characteristic] = c.Value
		}
	case chargen.ConsequenceYearsElapsed:
		s.age += c.Value
	case chargen.ConsequenceSkillAwarded:
		if c.Skill != "" {
			s.skills[c.Skill] = c.Value
		}
	default:
	}
}

// rolled reports whether the characteristics are known yet.
func (s *session) rolled() bool { return len(s.stats) == len(characteristicOrder) }

// characteristics renders step A's own line: the six, and the UPP they
// make ("Create the UPP", chart E1 step A).
func (s *session) characteristics() string {
	var b strings.Builder

	for _, name := range characteristicOrder {
		fmt.Fprintf(&b, "%s %-3d", name, s.stats[name])
	}

	return strings.TrimRight(b.String(), " ") + "   UPP " + s.upp()
}

// upp renders the six as the extended-hex string the record carries.
func (s *session) upp() string {
	return chargen.Characteristics{
		Str: s.stats["Str"], Dex: s.stats["Dex"], End: s.stats["End"],
		Int: s.stats["Int"], Edu: s.stats["Edu"], Soc: s.stats["Soc"],
	}.UPP()
}

// status is the line above each question: where the run has got to, and
// the character's own numbers.
//
// The step lags by one prompt at a term boundary, because the offer to
// suspend a term is put before the term's step is entered — which is
// deliberate, since a suspended term never becomes one (p. 59).
func (s *session) status() string {
	parts := []string{fmt.Sprintf("age %d", s.age)}

	if s.rolled() {
		parts = append(parts, "UPP "+s.upp())
	}

	if n := len(s.skills); n > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", n, plural(n, "skill")))
	}

	if s.step != "" {
		parts = append(parts, s.step)
	}

	return strings.Join(parts, " · ")
}

// plural is the crudest possible pluralisation, and enough: the only
// word it is asked for is "skill".
func plural(n int, word string) string {
	if n == 1 {
		return word
	}

	return word + "s"
}

// Rule draws a step heading, so the checklist a player is following is
// visible in the session rather than only in the record afterwards.
// Exported so a caller can close the session in the same shape.
func Rule(name string) string {
	const width = 64

	line := "── " + name + " "
	if pad := width - len([]rune(line)); pad > 0 {
		line += strings.Repeat("─", pad)
	}

	return line
}
