package chargen

// ChoiceID identifies a kind of choice point, so a policy can apply the
// matching POLICY.md rule.
type ChoiceID string

// The choice points the engine currently presents.
const (
	// ChooseCareer selects the next career (chart E1 step D, p. 72).
	ChooseCareer ChoiceID = "select_career"

	// ChooseControllingCharacteristic selects the term's controlling
	// characteristic (p. 65: "The player picks one of these
	// Characteristics ... it governs Risk and Reward for the current
	// Term").
	ChooseControllingCharacteristic ChoiceID = "select_controlling_characteristic"

	// ChooseSkillColumn selects the career skills table column (p. 65:
	// "The character selects a column and rolls 1D for the specific
	// skill").
	ChooseSkillColumn ChoiceID = "select_skill_column"

	// ChooseHobby selects the Citizen Hobby ("Second Success provides a
	// Hobby selected from Citizen Skills and Knowledges", chart 04
	// p. 78).
	ChooseHobby ChoiceID = "select_hobby"
)

// Choice is one choice point presented to a Decider. Options are listed in
// the order the rule presents them (first-listed order in Book 1). Scores,
// when non-nil, are engine-provided decision aids parallel to Options (for
// example the current characteristic values behind a controlling-
// characteristic choice); they are not part of the printed rule and are not
// recorded in the event log.
type Choice struct {
	ID      ChoiceID
	Prompt  string
	Options []string
	Scores  []int
	Cite    string
}

// Decider resolves choice points. Interactive play and the auto-mode
// policy are its two implementations (docs/PRD.md, Decisions): every
// choice in the engine goes through this interface so replay can reapply
// recorded choices.
type Decider interface {
	// Choose returns the index of the selected option.
	Choose(c Choice) int

	// Kind identifies the decider in choice events (docs/PRD.md FR10:
	// "who decided — player or policy").
	Kind() DeciderKind
}
