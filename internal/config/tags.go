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

// ForEachType iterates every type in v, calling fn with the type name and its
// computed tags. Pairs that produce empty tag lists are skipped. fn may return
// an error to abort iteration early; that error is returned to the caller.
func ForEachType(v *VersionDef, fn func(typeName string, tags []string) error) error {
	for _, typeName := range v.Types {
		tags := ApplyType(v.Tags, typeName)
		if len(tags) == 0 {
			continue
		}
		if err := fn(typeName, tags); err != nil {
			return err
		}
	}
	return nil
}
