package chargen

// The Service Academy's Officer1 graduation (chart C p. 60): "C5=8 BA
// Officer1". The first two are applied where every graduation is; the
// third is a link forward into a career the character has not entered yet,
// which is why it lives here rather than with the schooling.

import "strings"

// academyOfficer reports whether the character graduated the Service
// Academy of the named service, which is what earns him an officer's
// commission on entry (interpretation I-94).
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
		if record.Program == serviceAcademy && record.Graduated &&
			record.Service == service && strings.Contains(record.Degree, officer1) {
			return true
		}
	}

	return false
}

// The chart C row and the token its Graduation column carries.
const (
	serviceAcademy = "Service Academy"
	officer1       = "Officer1"
)
