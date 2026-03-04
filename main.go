package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/verity-org/integer/cmd"
)

func main() {
	app := &cli.App{
		Name:  "integer",
		Usage: "Build and manage Wolfi-based OCI images from source",
		Commands: []*cli.Command{
			cmd.DiscoverCommand,
			cmd.ValidateCommand,
			cmd.BuildCommand,
			cmd.CatalogCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
