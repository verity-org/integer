package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/verity-org/integer/internal/config"
)

// ErrValidationFailed is returned when one or more image configs fail validation.
var ErrValidationFailed = errors.New("validation failed")

// ValidateCommand schema-validates all YAML configs in images/.
var ValidateCommand = &cli.Command{
	Name:  "validate",
	Usage: "Schema-validate all image configs in images/",
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
		if _, err := config.LoadConfig(cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", cfgPath, err)
			failures++
		} else {
			fmt.Fprintf(os.Stdout, "OK   %s\n", cfgPath)
		}

		// Walk images/ and validate each *.yaml file.
		entries, err := os.ReadDir(imagesDir)
		if err != nil {
			return fmt.Errorf("reading images directory: %w", err)
		}

		checked := 0
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
				continue
			}

			defPath := filepath.Join(imagesDir, entry.Name())
			def, err := config.LoadImage(defPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", defPath, err)
				failures++
				continue
			}
			if err := config.Validate(def); err != nil {
				fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", defPath, err)
				failures++
				continue
			}
			fmt.Fprintf(os.Stdout, "OK   %s (%d types, %d declared versions)\n",
				defPath, len(def.Types), len(def.Versions))
			checked++
		}

		if failures > 0 {
			return fmt.Errorf("%d error(s): %w", failures, ErrValidationFailed)
		}

		fmt.Fprintf(os.Stdout, "\nAll configs valid (%d images checked)\n", checked)
		return nil
	},
}
