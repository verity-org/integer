package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/verity-org/integer/internal/config"
)

// Catalog is the top-level structure published to the reports branch and
// consumed by the verity website to render the "Zero-CVE Rebuilds" section.
type Catalog struct {
	GeneratedAt string  `json:"generatedAt"`
	Registry    string  `json:"registry"`
	Images      []Image `json:"images"`
}

// Image represents a single named image with all its version streams.
type Image struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Versions    []Version `json:"versions"`
}

// Version represents one version stream (e.g. "1.26" for Go 1.26).
type Version struct {
	Version  string    `json:"version"`
	Latest   bool      `json:"latest,omitempty"`
	EOL      string    `json:"eol,omitempty"`
	Variants []Variant `json:"variants"`
}

// Variant represents one built type (default, dev, fips) within a version.
type Variant struct {
	Type    string   `json:"type"`
	Tags    []string `json:"tags"`
	Ref     string   `json:"ref"`    // primary published ref (registry/name:tag)
	Digest  string   `json:"digest"` // empty if build report unavailable
	BuiltAt string   `json:"builtAt"`
	Status  string   `json:"status"` // "success" | "failure" | "unknown"
}

// buildReport matches the JSON written by .github/scripts/push-reports.sh.
type buildReport struct {
	Digest  string   `json:"digest"`
	Status  string   `json:"status"`
	BuiltAt string   `json:"built_at"`
	Tags    []string `json:"tags"`
}

// Generate walks imagesDir, merges build reports from reportsDir, and returns
// a Catalog. reportsDir may be empty (reports omitted, all variants "unknown");
// a non-empty reportsDir that does not exist is an error.
func Generate(imagesDir, reportsDir, registry string) (*Catalog, error) {
	if reportsDir != "" {
		if _, err := os.Stat(reportsDir); err != nil {
			return nil, fmt.Errorf("reports dir %q: %w", reportsDir, err)
		}
	}

	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		return nil, fmt.Errorf("reading images dir %q: %w", imagesDir, err)
	}

	var images []Image

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_base" {
			continue
		}

		defPath := filepath.Join(imagesDir, entry.Name(), "image.yaml")
		if _, err := os.Stat(defPath); err != nil {
			continue
		}

		def, err := config.LoadImageDefinition(defPath)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", defPath, err)
		}

		img := Image{
			Name:        def.Name,
			Description: def.Description,
		}

		for _, v := range def.Versions {
			ver := Version{
				Version: v.Version,
				Latest:  v.Latest,
				EOL:     v.EOL,
			}

			if err := config.ForEachType(&v, func(typeName string, tags []string) error {
				ref := fmt.Sprintf("%s/%s:%s", registry, def.Name, tags[0])
				variant := Variant{
					Type:   typeName,
					Tags:   tags,
					Ref:    ref,
					Status: "unknown",
				}
				if reportsDir != "" {
					reportPath := filepath.Join(reportsDir, def.Name, v.Version, typeName, "latest.json")
					if report, err := loadReport(reportPath); err == nil {
						variant.Digest = report.Digest
						variant.BuiltAt = report.BuiltAt
						variant.Status = report.Status
					}
				}
				ver.Variants = append(ver.Variants, variant)
				return nil
			}); err != nil {
				return nil, err
			}

			img.Versions = append(img.Versions, ver)
		}

		images = append(images, img)
	}

	return &Catalog{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Registry:    registry,
		Images:      images,
	}, nil
}

func loadReport(path string) (*buildReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r buildReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
