package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func runCatalogApp(t *testing.T, args []string) error {
	t.Helper()
	app := &cli.App{Commands: []*cli.Command{CatalogCommand}}
	return app.Run(append([]string{"integer"}, args...))
}

func TestCatalogCommand_Basic(t *testing.T) {
	imagesDir, cfgPath := setupCmdImages(t)
	outputPath := filepath.Join(t.TempDir(), "catalog.json")

	err := runCatalogApp(t, []string{
		"catalog",
		"--config", cfgPath,
		"--images-dir", imagesDir,
		"--apkindex-url", "",
		"--output", outputPath,
	})
	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var cat map[string]any
	require.NoError(t, json.Unmarshal(data, &cat))
	images, ok := cat["images"].([]any)
	require.True(t, ok)
	assert.Len(t, images, 1)
}

func TestCatalogCommand_WithAPKINDEX(t *testing.T) {
	srv := makeAPKINDEXServer(t, "P:nodejs-22\nV:22.0.0\n\nP:nodejs-24\nV:24.0.0\n\n")

	imagesDir, cfgPath := setupCmdImages(t)
	outputPath := filepath.Join(t.TempDir(), "catalog.json")

	err := runCatalogApp(t, []string{
		"catalog",
		"--config", cfgPath,
		"--images-dir", imagesDir,
		"--apkindex-url", srv.URL,
		"--cache-dir", t.TempDir(),
		"--output", outputPath,
	})
	require.NoError(t, err)
}

func TestCatalogCommand_APKINDEXFails_ContinuesWithVersionsMap(t *testing.T) {
	imagesDir, cfgPath := setupCmdImages(t)
	outputPath := filepath.Join(t.TempDir(), "catalog.json")

	// Use a bad URL — catalog should warn and continue with versions map.
	err := runCatalogApp(t, []string{
		"catalog",
		"--config", cfgPath,
		"--images-dir", imagesDir,
		"--apkindex-url", "http://127.0.0.1:0/bad",
		"--output", outputPath,
	})
	require.NoError(t, err)
}

func TestCatalogCommand_StdoutOutput(t *testing.T) {
	imagesDir, cfgPath := setupCmdImages(t)

	err := runCatalogApp(t, []string{
		"catalog",
		"--config", cfgPath,
		"--images-dir", imagesDir,
		"--apkindex-url", "",
		"--output", "-",
	})
	require.NoError(t, err)
}

func TestCatalogCommand_InvalidConfig(t *testing.T) {
	dir := t.TempDir()
	err := runCatalogApp(t, []string{
		"catalog",
		"--config", filepath.Join(dir, "nonexistent.yaml"),
		"--images-dir", dir,
		"--output", filepath.Join(dir, "out.json"),
	})
	require.Error(t, err)
}

func TestCatalogCommand_InvalidImagesDir(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "integer.yaml")
	writeFile(t, cfgPath, testIntegerYAML)

	err := runCatalogApp(t, []string{
		"catalog",
		"--config", cfgPath,
		"--images-dir", "/nonexistent/path",
		"--apkindex-url", "",
		"--output", filepath.Join(dir, "out.json"),
	})
	require.Error(t, err)
}
