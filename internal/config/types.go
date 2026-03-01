package config

// IntegerConfig represents the integer.yaml global configuration.
type IntegerConfig struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Target     TargetSpec   `yaml:"target"`
	Defaults   DefaultsSpec `yaml:"defaults"`
}

// TargetSpec describes the registry where built images are published.
type TargetSpec struct {
	Registry string `yaml:"registry"`
}

// DefaultsSpec holds project-wide defaults applied to all images.
type DefaultsSpec struct {
	Archs []string `yaml:"archs"`
}

// ImageDefinition represents an images/<name>/image.yaml file.
// Each image has one or more version streams (e.g. Node 20 LTS, Node 22 LTS).
// Within each version, one or more types are built (default, dev, fips, jre, …).
// File paths follow the convention: versions/<Version>/<type>.apko.yaml.
type ImageDefinition struct {
	APIVersion  string       `yaml:"apiVersion"`
	Kind        string       `yaml:"kind"`
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	EOLProduct  string       `yaml:"eol-product,omitempty"` // endoflife.date product slug
	Upstream    UpstreamSpec `yaml:"upstream,omitempty"`
	Versions    []VersionDef `yaml:"versions"`
}

// UpstreamSpec references the primary Wolfi package for the image.
type UpstreamSpec struct {
	Package string `yaml:"package"`
}

// VersionDef declares one version stream for an image (e.g. "22" for Node 22 LTS,
// "3.12" for Python 3.12). File paths are derived by convention:
//
//	versions/<Version>/<type>.apko.yaml
//
// Tags for the "default" type are used as-is; every other type gets the type
// name appended as a suffix (e.g. "22" → "22-dev", "22-fips").
type VersionDef struct {
	Version string   `yaml:"version"`          // "22", "3.12", "17", …
	EOL     string   `yaml:"eol,omitempty"`    // "2027-04-30" — from endoflife.date
	Latest  bool     `yaml:"latest,omitempty"` // true → carries the "latest" tag
	Tags    []string `yaml:"tags"`             // base tags for the default type
	Types   []string `yaml:"types"`            // ["default", "dev", "fips"]
}
