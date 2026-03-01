package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/verity-org/integer/internal/config"
)

// ErrVariantNotFound is returned when the requested version/type combination is not defined.
var ErrVariantNotFound = errors.New("version/type not found")

// BuildCommand runs a local apko build for a specific image+version+type combination.
// Intended for development workflows; CI uses apko publish with multi-arch support.
var BuildCommand = &cli.Command{
	Name:  "build",
	Usage: "Build a single image variant locally using apko (single-arch)",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "image",
			Aliases:  []string{"i"},
			Usage:    "Image name (e.g., node)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "version",
			Aliases:  []string{"V"},
			Usage:    "Version stream (e.g., 22, 3.12)",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "type",
			Aliases: []string{"t"},
			Usage:   "Image type (e.g., default, dev, fips)",
			Value:   "default",
		},
		&cli.StringFlag{
			Name:  "images-dir",
			Usage: "Path to the images/ directory",
			Value: "images",
		},
		&cli.StringFlag{
			Name:  "output",
			Usage: "Output tarball path",
			Value: "image.tar",
		},
		&cli.StringFlag{
			Name:  "arch",
			Usage: "Target architecture",
			Value: "amd64",
		},
	},
	Action: func(c *cli.Context) error {
		imageName := c.String("image")
		version := c.String("version")
		typeName := c.String("type")
		imagesDir := c.String("images-dir")

		defPath := filepath.Join(imagesDir, imageName, "image.yaml")

		def, err := config.LoadImageDefinition(defPath)
		if err != nil {
			return fmt.Errorf("failed to load image definition: %w", err)
		}

		apkoFile, err := findApkoFile(def, version, typeName, filepath.Join(imagesDir, imageName))
		if err != nil {
			return err
		}

		output := c.String("output")
		arch := c.String("arch")

		fmt.Fprintf(os.Stderr, "Building %s:%s-%s (%s) → %s\n", imageName, version, typeName, arch, output)

		return runApkoBuild(c.Context, apkoFile, output, arch)
	},
}

// findApkoFile returns the absolute path to the apko YAML for a given version+type.
func findApkoFile(def *config.ImageDefinition, version, typeName, imageDir string) (string, error) {
	for _, v := range def.Versions {
		if v.Version != version {
			continue
		}

		if slices.Contains(v.Types, typeName) {
			return filepath.Join(imageDir, "versions", version, typeName+".apko.yaml"), nil
		}

		return "", fmt.Errorf("type %q not found for version %q in image %q (available: %s): %w",
			typeName, version, def.Name, strings.Join(v.Types, ", "), ErrVariantNotFound)
	}

	available := make([]string, 0, len(def.Versions))
	for _, v := range def.Versions {
		available = append(available, v.Version)
	}

	return "", fmt.Errorf("version %q not found in image %q (available: %s): %w",
		version, def.Name, strings.Join(available, ", "), ErrVariantNotFound)
}

// runApkoBuild executes apko to build an OCI tarball from the given config.
func runApkoBuild(ctx context.Context, configFile, output, arch string) error {
	apko, err := exec.LookPath("apko")
	if err != nil {
		return fmt.Errorf("apko not found in PATH — install via mise: %w", err)
	}

	cmd := exec.CommandContext(ctx, apko, "build",
		"--arch", arch,
		configFile,
		"integer:local",
		output,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apko build failed: %w", err)
	}

	return nil
}
