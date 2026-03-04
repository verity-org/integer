package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/verity-org/integer/internal/catalog"
	"github.com/verity-org/integer/internal/config"
)

// CatalogCommand generates catalog.json from image definitions + build reports.
var CatalogCommand = &cli.Command{
	Name:  "catalog",
	Usage: "Generate catalog.json for the verity website",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "images-dir",
			Aliases: []string{"i"},
			Usage:   "path to the images/ directory",
			Value:   "images",
		},
		&cli.StringFlag{
			Name:  "reports-dir",
			Usage: "path to checked-out reports directory (reports branch)",
		},
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "path to integer.yaml",
			Value:   "integer.yaml",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "output file path",
			Value:   "catalog.json",
		},
	},
	Action: runCatalog,
}

func runCatalog(c *cli.Context) error {
	cfg, err := config.LoadIntegerConfig(c.String("config"))
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cat, err := catalog.Generate(
		c.String("images-dir"),
		c.String("reports-dir"),
		cfg.Target.Registry,
	)
	if err != nil {
		return fmt.Errorf("generating catalog: %w", err)
	}

	out, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling catalog: %w", err)
	}

	output := c.String("output")
	if output == "-" {
		fmt.Println(string(out))
		return nil
	}

	if err := os.WriteFile(output, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", output, err)
	}

	fmt.Printf("Catalog → %s (%d images)\n", output, len(cat.Images))
	return nil
}
