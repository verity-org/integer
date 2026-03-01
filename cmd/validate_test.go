package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestValidateCommand_AllValid(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "integer.yaml"), `
apiVersion: integer.verity.supply/v1alpha1
kind: IntegerConfig
target:
  registry: ghcr.io/test-org
`)

	imagesDir := filepath.Join(dir, "images")
	versionsDir := filepath.Join(imagesDir, "myapp", "versions", "1")
	require.NoError(t, os.MkdirAll(versionsDir, 0o755))

	writeFile(t, filepath.Join(imagesDir, "myapp", "image.yaml"), `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: myapp
versions:
  - version: "1"
    tags: ["1", "latest"]
    types: [default]
`)
	writeFile(t, filepath.Join(versionsDir, "default.apko.yaml"), "contents:\n  packages: []\n")

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", filepath.Join(dir, "integer.yaml"),
		"--images-dir", imagesDir,
	})
	assert.NoError(t, err)
}

func TestValidateCommand_MissingApkoFile(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "integer.yaml"), `
apiVersion: integer.verity.supply/v1alpha1
kind: IntegerConfig
target:
  registry: ghcr.io/test-org
`)

	imagesDir := filepath.Join(dir, "images")
	require.NoError(t, os.MkdirAll(filepath.Join(imagesDir, "myapp", "versions", "1"), 0o755))

	writeFile(t, filepath.Join(imagesDir, "myapp", "image.yaml"), `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: myapp
versions:
  - version: "1"
    tags: ["1"]
    types: [default]
`)
	// No versions/1/default.apko.yaml created

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", filepath.Join(dir, "integer.yaml"),
		"--images-dir", imagesDir,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationFailed)
}

func TestValidateCommand_InvalidImageYaml(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "integer.yaml"), `
apiVersion: integer.verity.supply/v1alpha1
kind: IntegerConfig
target:
  registry: ghcr.io/test-org
`)

	imagesDir := filepath.Join(dir, "images")
	require.NoError(t, os.MkdirAll(filepath.Join(imagesDir, "broken"), 0o755))
	writeFile(t, filepath.Join(imagesDir, "broken", "image.yaml"), ":: invalid yaml ::")

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", filepath.Join(dir, "integer.yaml"),
		"--images-dir", imagesDir,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationFailed)
}

func TestValidateCommand_InvalidIntegerConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "integer.yaml"), ":: bad yaml ::")
	imagesDir := filepath.Join(dir, "images")
	require.NoError(t, os.MkdirAll(imagesDir, 0o755))

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", filepath.Join(dir, "integer.yaml"),
		"--images-dir", imagesDir,
	})
	// Bad integer.yaml counts as a failure
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationFailed)
}

func TestValidateCommand_SkipsBaseDir(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "integer.yaml"), `
apiVersion: integer.verity.supply/v1alpha1
kind: IntegerConfig
target:
  registry: ghcr.io/test-org
`)

	imagesDir := filepath.Join(dir, "images")
	// _base should be skipped — no image.yaml there
	require.NoError(t, os.MkdirAll(filepath.Join(imagesDir, "_base"), 0o755))
	writeFile(t, filepath.Join(imagesDir, "_base", "wolfi-base.yaml"), "contents:\n  packages: []\n")

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", filepath.Join(dir, "integer.yaml"),
		"--images-dir", imagesDir,
	})
	assert.NoError(t, err)
}
