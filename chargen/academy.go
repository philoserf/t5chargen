package chargen

// The Service Academy's Officer1 graduation (chart C p. 60): "C5=8 BA
// Officer1". The first two are applied where every graduation is; the
// third is a link forward into a career the character has not entered yet,
// which is why it lives here rather than with the schooling.

import "strings"

// academyOfficer reports whether the character holds an Officer1
// commission into the named service, which is what earns him an officer's
// entry rank (interpretation I-94).
//
// Keyed on the Degree rather than on the Service Academy row, because
// three chart C rows confer one: the Academy's "BA Officer1" and, from
// p. 61, OTC's "Army Officer1" and NOTC's "Navy Officer1 or Marine
// Officer1". The obligation each carries is the same.
//
// Graduation, not attendance: Officer1 is printed in chart C's Graduation
// column, so a cadet who failed out carries nothing forward. And the
// service must match — an Academy trains for the force it names, and its
// graduate joining a different one enters as any other recruit does.
func (c *Character) academyOfficer(service string) bool {
	if service == "" {
		return false
	}

	for _, record := range c.Education {
		if record.Graduated && record.Service == service &&
			strings.Contains(record.Degree, officer1) {
			return true
		}
	}

	return false
}

// academyCommission names the service a character was commissioned into,
// or "" where he holds no commission.
//
// It is academyOfficer asked the other way round: entryRank knows which
// career it is resolving and asks whether this character is its officer;
// career selection has no career yet and needs to be told which one the
// commission obliges him to enter (p. 62, interpretation I-99).
func (c *Character) academyCommission() string {
	for _, record := range c.Education {
		if record.Graduated && record.Service != "" &&
			strings.Contains(record.Degree, officer1) {
			return record.Service
		}
	}

	return ""
}

// The chart C row that names a service by asking, and the token every
// commissioning row's Graduation column carries.
const (
	serviceAcademy = "Service Academy"
	officer1       = "Officer1"
)
