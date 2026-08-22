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

	// ChooseHomeworld selects the homeworld (chart E1 step B, p. 72;
	// "determined by selection, assignment, or random rolls", p. 58).
	ChooseHomeworld ChoiceID = "select_homeworld"

	// ChooseArt and ChooseTrade resolve the chart B "One Art (Choose
	// One)" and "The Trades (Choose One)" grants (p. 56).
	ChooseArt   ChoiceID = "select_art"
	ChooseTrade ChoiceID = "select_trade"

	// ChooseEducation selects the pre-career program, or None (chart C
	// p. 60; "Consider acquiring an advanced education", p. 57 step C).
	ChooseEducation ChoiceID = "select_education"

	// ChooseService selects the Service Academy service (chart C p. 60).
	ChooseService ChoiceID = "select_service"

	// ChooseMajor and ChooseMinor select the education Major and Minor
	// ("they cannot be the same", p. 59).
	ChooseMajor ChoiceID = "select_major"
	ChooseMinor ChoiceID = "select_minor"

	// ChooseCheck selects among a check's stated characteristics ("Check
	// one of the stated Characteristics", p. 59).
	ChooseCheck ChoiceID = "select_check_characteristic"

	// ChooseHonors accepts or declines the optional Honors roll (p. 59).
	ChooseHonors ChoiceID = "attempt_honors"

	// ChooseWaiver accepts or declines an Educational Waiver attempt
	// (p. 59).
	ChooseWaiver ChoiceID = "attempt_waiver"

	// ChooseSkill selects a skill from an open list: the Apprenticeship
	// Skill+4 (chart C p. 60), and the Master Skill List entries a single
	// chart cell covers (chart 04 table E "Grav" and "Spacecraft";
	// ERRATA.md I-10, I-11).
	ChooseSkill ChoiceID = "select_skill"

	// ChooseDuty selects the Scout term's duty (chart 05 table B; "A
	// Scout may avoid the Risk and Reward rolls by volunteering for
	// Courier Duty", p. 79).
	ChooseDuty ChoiceID = "select_duty"

	// ChooseRiskMod selects Caution, Bravery, or No Mod for Risk &
	// Reward (p. 65; chart 05 p. 79).
	ChooseRiskMod ChoiceID = "select_risk_mod"

	// ChooseRetry accepts or declines a career Retry row (chart 05
	// "Retry R&R C5"; interpretation I-8, ERRATA.md).
	ChooseRetry ChoiceID = "attempt_retry"

	// ChooseBeginTrack selects the entry path of a career offering
	// several (chart 06: "To Begin 4th Officer / Spacehand / Temp",
	// p. 80).
	ChooseBeginTrack ChoiceID = "select_begin_track"

	// ChooseSpecialty selects the Entertainer's art (chart 03 "Select A
	// Specialty", p. 77).
	ChooseSpecialty ChoiceID = "select_specialty"

	// ChooseOptionalFlux accepts or declines one of the two optional Fame
	// Flux rolls ("the first is required; the second and third are
	// optional", chart 03).
	ChooseOptionalFlux ChoiceID = "attempt_optional_flux"

	// ChooseComeback accepts or declines an Entertainer Comeback
	// ("Comeback is possible any number of times", chart 03).
	ChooseComeback ChoiceID = "attempt_comeback"

	// ChooseAdvancement accepts or declines a commission or promotion
	// attempt ("Temp may attempt Officer Commission and Rating
	// Promotion", chart 06 p. 80).
	ChooseAdvancement ChoiceID = "attempt_advancement"
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
