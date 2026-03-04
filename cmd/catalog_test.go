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

const testIntegerYAML = `apiVersion: integer.verity.supply/v1alpha1
kind: IntegerConfig
target:
  registry: ghcr.io/verity-org
defaults:
  archs:
    - amd64
    - arm64
`

const testImageYAML = `apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: node
description: "Node.js runtime"
versions:
  - version: "22"
    tags: ["22"]
    types: [default]
`

func runCatalogApp(t *testing.T, args []string) error {
	t.Helper()
	app := &cli.App{Commands: []*cli.Command{CatalogCommand}}
	return app.Run(append([]string{"integer"}, args...))
}

func TestCatalogCommand_Basic(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "integer.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(testIntegerYAML), 0o644))

	imagesDir := filepath.Join(dir, "images")
	imageYAMLPath := filepath.Join(imagesDir, "node", "image.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(imageYAMLPath), 0o755))
	require.NoError(t, os.WriteFile(imageYAMLPath, []byte(testImageYAML), 0o644))

	outputPath := filepath.Join(dir, "catalog.json")

	err := runCatalogApp(t, []string{
		"catalog",
		"--config", cfgPath,
		"--images-dir", imagesDir,
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

func TestCatalogCommand_StdoutOutput(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "integer.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(testIntegerYAML), 0o644))

	imagesDir := filepath.Join(dir, "images")
	require.NoError(t, os.MkdirAll(imagesDir, 0o755))

	// Output to stdout via "-"
	err := runCatalogApp(t, []string{
		"catalog",
		"--config", cfgPath,
		"--images-dir", imagesDir,
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
	require.NoError(t, os.WriteFile(cfgPath, []byte(testIntegerYAML), 0o644))

	err := runCatalogApp(t, []string{
		"catalog",
		"--config", cfgPath,
		"--images-dir", "/nonexistent/path",
		"--output", filepath.Join(dir, "out.json"),
	})
	require.Error(t, err)
}
