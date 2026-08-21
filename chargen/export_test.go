package chargen

import "slices"

// RegistryNames is a test bridge to the careerRegistry key set.
func RegistryNames() []string {
	names := make([]string, 0, len(careerRegistry))
	for name := range careerRegistry {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}
