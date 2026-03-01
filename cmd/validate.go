package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/verity-org/integer/internal/config"
	"github.com/verity-org/integer/internal/discovery"
)

// ErrValidationFailed is returned when one or more image configs fail validation.
var ErrValidationFailed = errors.New("validation failed")

// ValidateCommand schema-validates all YAML configs and verifies that every
// referenced apko file exists on disk.
var ValidateCommand = &cli.Command{
	Name:  "validate",
	Usage: "Schema-validate all image configs and verify referenced files exist",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to integer.yaml",
			Value:   "integer.yaml",
		},
		&cli.StringFlag{
			Name:  "images-dir",
			Usage: "Path to the images/ directory",
			Value: "images",
		},
	},
	Action: func(c *cli.Context) error {
		cfgPath := c.String("config")
		imagesDir := c.String("images-dir")

		failures := 0

		// Validate global config.
		if _, err := config.LoadIntegerConfig(cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL integer.yaml: %v\n", err)
			failures++
		} else {
			fmt.Fprintf(os.Stdout, "OK   %s\n", cfgPath)
		}

		// Walk images/ and validate each image.yaml + referenced apko files.
		entries, err := os.ReadDir(imagesDir)
		if err != nil {
			return fmt.Errorf("reading images directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == "_base" {
				continue
			}

			imageDir := filepath.Join(imagesDir, entry.Name())
			defPath := filepath.Join(imageDir, "image.yaml")

			def, err := config.LoadImageDefinition(defPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", defPath, err)
				failures++
				continue
			}
			fmt.Fprintf(os.Stdout, "OK   %s\n", defPath)

			// Validate each version×type apko file exists.
			for _, v := range def.Versions {
				for _, typeName := range v.Types {
					apkoPath := filepath.Join(imageDir, "versions", v.Version, typeName+".apko.yaml")
					if _, err := os.Stat(apkoPath); err != nil {
						fmt.Fprintf(os.Stderr, "FAIL %s (version %s, type %q): file not found\n",
							defPath, v.Version, typeName)
						failures++
					} else {
						fmt.Fprintf(os.Stdout, "OK   %s (%s/%s)\n", apkoPath, v.Version, typeName)
					}
				}
			}
		}

		// Report overall result.
		if failures > 0 {
			return fmt.Errorf("%d error(s): %w", failures, ErrValidationFailed)
		}

		apkoFiles, err := discovery.WalkApkoFiles(imagesDir)
		if err != nil {
			return fmt.Errorf("walking apko files: %w", err)
		}
		fmt.Fprintf(os.Stdout, "\nAll configs valid (%d apko files checked)\n", len(apkoFiles))
		return nil
	},
}
