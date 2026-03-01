package discovery

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/verity-org/integer/internal/config"
)

// ErrVariantFileMissing is returned when a version/type apko YAML file does not exist.
var ErrVariantFileMissing = errors.New("apko config file does not exist")

// DiscoveredImage represents one buildable image: a name × version × type combination.
type DiscoveredImage struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Type     string   `json:"type"`
	File     string   `json:"file"`
	Tags     []string `json:"tags"`
	Registry string   `json:"registry"`
}

// Discover walks the images/ directory and returns every name×version×type
// combination defined across all image.yaml files. _base/ is skipped.
func Discover(imagesDir, registry string) ([]DiscoveredImage, error) {
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		return nil, fmt.Errorf("reading images directory %q: %w", imagesDir, err)
	}

	var results []DiscoveredImage

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_base" {
			continue
		}

		imageDir := filepath.Join(imagesDir, entry.Name())
		defPath := filepath.Join(imageDir, "image.yaml")

		if _, err := os.Stat(defPath); err != nil {
			continue
		}

		def, err := config.LoadImageDefinition(defPath)
		if err != nil {
			return nil, fmt.Errorf("loading image %q: %w", entry.Name(), err)
		}

		imgs, err := expandVersions(def, imageDir, registry)
		if err != nil {
			return nil, fmt.Errorf("expanding versions for %q: %w", entry.Name(), err)
		}

		results = append(results, imgs...)
	}

	return results, nil
}

// expandVersions converts one ImageDefinition into DiscoveredImage entries by
// iterating every version × type combination. File paths are resolved and
// verified to exist. Returns a new slice — never mutates the input.
func expandVersions(def *config.ImageDefinition, imageDir, registry string) ([]DiscoveredImage, error) {
	results := make([]DiscoveredImage, 0, len(def.Versions)*2)

	for _, v := range def.Versions {
		for _, typeName := range v.Types {
			relFile := filepath.Join("versions", v.Version, typeName+".apko.yaml")
			absFile := filepath.Join(imageDir, relFile)

			if _, err := os.Stat(absFile); err != nil {
				return nil, fmt.Errorf("versions/%s/%s.apko.yaml for image %q: %w",
					v.Version, typeName, def.Name, ErrVariantFileMissing)
			}

			tags := deriveTags(v.Tags, typeName)

			results = append(results, DiscoveredImage{
				Name:     def.Name,
				Version:  v.Version,
				Type:     typeName,
				File:     absFile,
				Tags:     tags,
				Registry: registry,
			})
		}
	}

	return results, nil
}

// deriveTags returns the tags for a given type. The "default" type uses the
// base tags unchanged; all other types append "-<type>" to each base tag.
func deriveTags(baseTags []string, typeName string) []string {
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

// WalkApkoFiles returns all apko YAML file paths under imagesDir, excluding
// _base/ and image.yaml files. Used by validate to count checked files.
func WalkApkoFiles(imagesDir string) ([]string, error) {
	var paths []string

	err := filepath.WalkDir(imagesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && d.Name() == "_base" {
			return filepath.SkipDir
		}

		if !d.IsDir() && filepath.Ext(path) == ".yaml" && filepath.Base(path) != "image.yaml" {
			paths = append(paths, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking images directory: %w", err)
	}

	return paths, nil
}
