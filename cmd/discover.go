package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/verity-org/integer/internal/config"
	"github.com/verity-org/integer/internal/discovery"
)

// DiscoverCommand walks the images/ directory and emits a JSON array of all
// image+variant combinations, suitable for use as a CI matrix.
var DiscoverCommand = &cli.Command{
	Name:  "discover",
	Usage: "List all image+variant combinations as a JSON array",
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
		cfg, err := config.LoadIntegerConfig(c.String("config"))
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		imgs, err := discovery.Discover(c.String("images-dir"), cfg.Target.Registry)
		if err != nil {
			return fmt.Errorf("failed to discover images: %w", err)
		}

		out, err := json.MarshalIndent(imgs, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}

		fmt.Fprintln(os.Stdout, string(out))
		return nil
	},
}
