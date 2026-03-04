package config

// ApplyType returns the tags for a given build type. The "default" type uses
// the base tags unchanged; all other types append "-<type>" to each base tag.
// Returns an empty slice when baseTags is empty.
func ApplyType(baseTags []string, typeName string) []string {
	tags := make([]string, len(baseTags))
	if typeName == "default" {
		copy(tags, baseTags)
		return tags
	}
	for i, t := range baseTags {
		tags[i] = t + "-" + typeName
	}
	return tags
}
