package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/urfave/cli/v2"

	"github.com/verity-org/integer/internal/apkindex"
	"github.com/verity-org/integer/internal/config"
	"github.com/verity-org/integer/internal/discovery"
	"github.com/verity-org/integer/internal/render"
)

// ErrVariantNotFound is returned when the requested version/type is not available.
var ErrVariantNotFound = errors.New("version/type not found")

// ErrNoVersions is returned when no versions can be resolved for an image.
var ErrNoVersions = errors.New("no versions found")

// BuildCommand runs a local apko build for a specific image+version+type.
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
			Name:    "version",
			Aliases: []string{"V"},
			Usage:   "Version (e.g., 22, 3.12, latest)",
			Value:   "latest",
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
		&cli.StringFlag{
			Name:  "apkindex-url",
			Usage: "Wolfi APKINDEX URL",
			Value: apkindex.DefaultAPKINDEXURL,
		},
	},
	Action: func(c *cli.Context) error {
		imageName := c.String("image")
		version := c.String("version")
		typeName := c.String("type")
		imagesDir := c.String("images-dir")

		// Load image definition.
		def, err := config.LoadImage(fmt.Sprintf("%s/%s.yaml", imagesDir, imageName))
		if err != nil {
			return fmt.Errorf("loading image %q: %w", imageName, err)
		}

		// Validate the requested type exists.
		tmpl, ok := def.Types[typeName]
		if !ok {
			return fmt.Errorf("type %q not defined for image %q: %w", typeName, imageName, ErrVariantNotFound)
		}

		// Resolve "latest" version from APKINDEX if needed.
		if version == "latest" {
			version, err = resolveLatestVersion(def, c.String("apkindex-url"))
			if err != nil {
				return err
			}
		}

		// Generate the apko config to a temp file.
		tmp, err := os.CreateTemp("", "integer-build-*.apko.yaml")
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
		defer os.Remove(tmp.Name())

		basePath := imagesDir + "/_base"
		out, err := render.Config(&tmpl, version, basePath)
		if err != nil {
			return fmt.Errorf("rendering apko config: %w", err)
		}
		if _, err := tmp.Write(out); err != nil {
			return fmt.Errorf("writing apko config: %w", err)
		}
		tmp.Close()

		output := c.String("output")
		arch := c.String("arch")
		fmt.Fprintf(os.Stderr, "Building %s:%s-%s (%s) → %s\n", imageName, version, typeName, arch, output)
		return runApkoBuild(c.Context, tmp.Name(), output, arch)
	},
}

// resolveLatestVersion fetches the APKINDEX and returns the latest version for the image.
func resolveLatestVersion(def *config.ImageDef, apkindexURL string) (string, error) {
	pkgs, err := apkindex.Fetch(apkindexURL, "", 0)
	if err != nil {
		return "", fmt.Errorf("fetching APKINDEX: %w", err)
	}
	versions := discovery.ResolveVersions(def, pkgs)
	if len(versions) == 0 {
		return "", fmt.Errorf("image %q: %w", def.Name, ErrNoVersions)
	}
	return versions[len(versions)-1], nil
}

// runApkoBuild executes apko to build an OCI tarball.
func runApkoBuild(ctx context.Context, configFile, output, arch string) error {
	apko, err := exec.LookPath("apko")
	if err != nil {
		return fmt.Errorf("apko not found in PATH (install via mise): %w", err)
	}
	cmd := exec.CommandContext(ctx, apko, "build", "--arch", arch, configFile, "integer:local", output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apko build failed: %w", err)
	}
	return nil
}
