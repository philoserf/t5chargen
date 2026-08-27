package chargen

// ChoiceID identifies a kind of choice point, so a policy can apply the
// matching POLICY.md rule.
type ChoiceID string

// The choice points the engine currently presents.
const (
	// ChooseCareer selects the next career (chart E1 step D, p. 72).
	ChooseCareer ChoiceID = "select_career"

	// ChooseCareerChange offers the p. 66 decision at the end of a term:
	// "A character may avoid the Continue roll (and its possibility of
	// Mandatory Continue) by voluntarily ending his service in the
	// current career and selecting a different career for which he is
	// eligible".
	ChooseCareerChange ChoiceID = "change_career"

	// ChooseCashOut offers p. 69's alternative to an annual payment:
	// "Any Entitlement can be cashed out for a lump sum equal to five
	// years of payments".
	ChooseCashOut ChoiceID = "cash_out_entitlement"

	// ChooseBenefitColumn is p. 68's per-roll decision: "Which Column?
	// Character may select either the Money column or Benefits column
	// for each roll." Chart 11 adds a third, Power.
	ChooseBenefitColumn ChoiceID = "select_benefit_column"

	// ChooseBenefitDM offers as much of a table D DM as the character
	// wants: "The DMs on the Benefits Tables are optional ... They may be
	// partially applied (any value may be selected from 0 to the total
	// allowed value)" (p. 68).
	ChooseBenefitDM ChoiceID = "apply_benefit_dm"

	// ChooseFameFlux offers chart F's once-per-character gamble: "Any
	// character may choose (once during Character Generation or after
	// adventuring begins) to add Flux to Fame" (p. 91).
	ChooseFameFlux ChoiceID = "invoke_fame_flux"

	// ChooseAssociatedCareer names the prior career a Functionary
	// position belongs to: "The Functionary character must identify with
	// which prior career his position is associated" (chart 13 p. 87).
	// Muster out reads it — a later Functionary's terms join the earlier
	// career's benefit DM (p. 68).
	ChooseAssociatedCareer ChoiceID = "select_associated_career"

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

	// ChooseOfficerTraining volunteers for OTC or NOTC, or declines
	// both: "A character attending College or University may also
	// volunteer to participate in OTC ... or NOTC" (p. 61).
	ChooseOfficerTraining ChoiceID = "volunteer_officer_training"

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

	// ChooseSchemeFlux accepts or adjusts the Rogue's Scheme roll: "Flux
	// may be modified (after roll) plus or minus 1" (chart 10, p. 84).
	ChooseSchemeFlux ChoiceID = "adjust_scheme_flux"

	// ChooseSchemeCareer takes a career already served as the Scheme in
	// place of rolling for one: "A Rogue may select for his Scheme
	// (rather than roll) any previous career" (chart 10, p. 84). Offered
	// only where there is a previous career to take.
	ChooseSchemeCareer ChoiceID = "select_scheme_career"

	// ChooseResignReserves offers the resignation p. 67 allows a
	// character enrolled on leaving a military, naval or marine career:
	// "A character may resign from the Reserves (Check Continue) and
	// forego its benefits and responsibilities." Offered before the
	// Check, so declining throws nothing.
	ChooseResignReserves ChoiceID = "resign_reserves"

	// ChooseFlightSchool offers chart C's Flight School to a character
	// in the first year of his first Armed Forces term: "Service
	// Academy Honors Graduates may attend Flight School" (p. 60);
	// "College or University Honors Graduates who participated in OTC
	// or NOTC may attend Flight School" (p. 61).
	ChooseFlightSchool ChoiceID = "attend_flight_school"

	// ChooseBranchChange offers the branch change an advancement
	// entitles a character to: "Enlisted may select a new Branch upon
	// Promotion" (charts 07, 08, 12), and "A character who receives a
	// Commission may roll for Branch or keep his current Branch"
	// (p. 66).
	ChooseBranchChange ChoiceID = "change_branch"

	// ChooseKnowledge selects which of a container skill's Knowledges a
	// receipt awards: "A character who receives a Skill may always
	// choose one of its contained Knowledges instead" (p. 134), and the
	// first two receipts award one whether he likes it or not.
	ChooseKnowledge ChoiceID = "select_knowledge"

	// ChooseBranch selects the Armed Forces branch where the character
	// passes the chart's Branch check ("Select Branch Soc", chart 08
	// p. 82); a failed check rolls one instead.
	ChooseBranch ChoiceID = "select_branch"

	// ChooseElevationFlux accepts or declines the Noble's one-shot Flux on
	// an Elevation roll ("Once in the Noble Career after a successful
	// Intrigue, invoke Flux as a Mod on Elevation roll", chart 11 p. 85).
	ChooseElevationFlux ChoiceID = "invoke_elevation_flux"

	// ChooseCareerWaiver accepts or declines a career waiver ("An adverse
	// die roll or decision ... may be waived", chart 02 p. 76). Distinct
	// from ChooseWaiver, the Educational Waiver, because the two draw on
	// one pool but warrant different policies.
	ChooseCareerWaiver ChoiceID = "attempt_career_waiver"

	// ChooseTenure accepts or declines a Tenure application ("Scholar
	// with Edu 10+ may apply for Tenure upon reaching Scholar3", chart 02
	// p. 76).
	ChooseTenure ChoiceID = "attempt_tenure"

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

	// ChooseLaterEducation offers the p. 59 suspension: "At the beginning
	// of any term, the character may apply for any Educational Institution
	// or Training, and if accepted substitutes that process for the entire
	// term".
	ChooseLaterEducation ChoiceID = "apply_later_education"
)

// Choice is one choice point presented to a Decider. Options are listed in
// the order the rule presents them (first-listed order in Book 1). Scores,
// when non-nil, are engine-provided decision aids parallel to Options — the
// current characteristic values behind a controlling-characteristic choice,
// or what refusing a waiver would cost (POLICY.md). They are not part of
// the printed rule and are not recorded in the event log, so a policy can
// weigh a stake without reading the prompt text and rewording a prompt
// cannot change a generated character.
type Choice struct {
	ID      ChoiceID
	Prompt  string
	Options []string
	Scores  []int
	Cite    string

	// ScoreLabel names what Scores mean, for a decider that shows them to
	// a person. "1" against a program means "you qualify", and nobody
	// could guess that from the digit; unlabelled scores stay between the
	// engine and the policy.
	//
	// A label applies to every option in the list, so only label a Score
	// that means the same thing for each of them. Where the array is
	// really one flag about the choice, padded to length — the waiver
	// stake is — labelling it reads as a claim about each option in turn,
	// and the second option's padding reads as the opposite of the truth.
	ScoreLabel string

	// Nth and Of place a choice in a run of identical ones: the term's
	// skill selections are the same question asked several times, and a
	// player answering the fifth cannot otherwise tell it from the first.
	// Engine-provided decision data like Scores — not part of the printed
	// rule, and not recorded, so a front end may show them and replay
	// never sees them.
	Nth, Of int
}

// Decider resolves choice points. Interactive play and the auto-mode
// policy are its two implementations (docs/PRD.md, Decisions): every
// choice in the engine goes through this interface so replay can reapply
// recorded choices.
type Decider interface {
	// Choose returns the index of the selected option. An error refuses the
	// choice outright and ends generation: an interactive session the player
	// abandoned (docs/PRD.md, CLI sketch: "Interrupted interactive sessions
	// produce no output file"), or a replay whose recorded choice no longer
	// matches the choice the engine presents. Both are distinct from an
	// out-of-range index, which is a decider that answered wrongly rather
	// than one that declined to answer.
	Choose(c Choice) (int, error)

	// Kind identifies the decider in choice events (docs/PRD.md FR10:
	// "who decided — player or policy").
	Kind() DeciderKind
}
