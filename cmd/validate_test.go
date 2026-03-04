package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestValidateCommand_AllValid(t *testing.T) {
	imagesDir, cfgPath := setupCmdImages(t)

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", cfgPath,
		"--images-dir", imagesDir,
	})
	assert.NoError(t, err)
}

func TestValidateCommand_InvalidImageYaml(t *testing.T) {
	imagesDir, cfgPath := setupCmdImages(t)
	writeFile(t, filepath.Join(imagesDir, "broken.yaml"), "not: valid: yaml: [")

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", cfgPath,
		"--images-dir", imagesDir,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationFailed)
}

func TestValidateCommand_InvalidIntegerConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "integer.yaml")
	writeFile(t, cfgPath, ":: bad yaml ::")

	imagesDir := filepath.Join(dir, "images")
	writeFile(t, filepath.Join(imagesDir, "node.yaml"), testNodeYAML)

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", cfgPath,
		"--images-dir", imagesDir,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationFailed)
}

func TestValidateCommand_APKINDEXCheck_Missing(t *testing.T) {
	// APKINDEX has packages, but not the nodejs packages the image needs.
	srv := makeAPKINDEXServer(t, "P:curl\nV:8.0.0\n\n")
	imagesDir, cfgPath := setupCmdImages(t)

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", cfgPath,
		"--images-dir", imagesDir,
		"--apkindex-url", srv.URL,
		"--cache-dir", t.TempDir(), // isolated cache — prevents OS temp dir hits
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationFailed)
}

func TestValidateCommand_APKINDEXCheck_Found(t *testing.T) {
	srv := makeAPKINDEXServer(t, "P:nodejs-22\nV:22.0.0\n\n")
	imagesDir, cfgPath := setupCmdImages(t)

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", cfgPath,
		"--images-dir", imagesDir,
		"--apkindex-url", srv.URL,
		"--cache-dir", t.TempDir(),
	})
	assert.NoError(t, err)
}

func TestValidateCommand_SkipsNonYAML(t *testing.T) {
	imagesDir, cfgPath := setupCmdImages(t)
	writeFile(t, filepath.Join(imagesDir, "README.md"), "# readme")
	writeFile(t, filepath.Join(imagesDir, "notes.txt"), "notes")

	app := &cli.App{Commands: []*cli.Command{ValidateCommand}}
	err := app.Run([]string{
		"integer", "validate",
		"--config", cfgPath,
		"--images-dir", imagesDir,
	})
	assert.NoError(t, err)
}
