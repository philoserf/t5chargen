package chargen

import (
	"maps"
	"slices"

	"github.com/philoserf/t5chargen/career"
)

// RegistryNames is a test bridge to the careerRegistry key set, sorted.
func RegistryNames() []string {
	return slices.Sorted(maps.Keys(careerRegistry))
}

// RegistryDefinitionNames is a test bridge mapping each careerRegistry key
// to its definition's Name — dispatch keys on the registry key while every
// user-visible label derives from Definition.Name, so the two must match.
func RegistryDefinitionNames() (map[string]string, error) {
	names := make(map[string]string, len(careerRegistry))

	for key, entry := range careerRegistry {
		def, _, err := entry()
		if err != nil {
			return nil, err
		}

		names[key] = def.Name
	}

	return names, nil
}

// LoadUndercoverCareer is a test bridge to the Agent's cover-career
// resolution, so the transcribed Undercover table can be checked against
// the careers the engine can actually read.
func LoadUndercoverCareer(source string) (*career.Definition, error) {
	return loadUndercoverCareer(source)
}

// ScoutDuties exports chart 05's duty labels. They are a Go literal and
// their skill eligibility is a JSON map, so only a test can hold the two
// key sets together.
var ScoutDuties = scoutDuties
