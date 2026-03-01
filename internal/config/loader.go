package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadIntegerConfig reads and parses the global integer.yaml.
func LoadIntegerConfig(path string) (*IntegerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading integer config %q: %w", path, err)
	}
	var cfg IntegerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing integer config %q: %w", path, err)
	}
	return &cfg, nil
}

// LoadImageDefinition reads and parses an images/<name>/image.yaml file.
func LoadImageDefinition(path string) (*ImageDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading image definition %q: %w", path, err)
	}
	var def ImageDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parsing image definition %q: %w", path, err)
	}
	return &def, nil
}
