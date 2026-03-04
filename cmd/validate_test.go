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
