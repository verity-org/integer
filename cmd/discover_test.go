package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestDiscoverCommand(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "integer.yaml"), `
apiVersion: integer.verity.supply/v1alpha1
kind: IntegerConfig
target:
  registry: ghcr.io/test-org
defaults:
  archs: [amd64, arm64]
`)

	imagesDir := filepath.Join(dir, "images")
	versionsDir := filepath.Join(imagesDir, "alpine", "versions", "3")
	require.NoError(t, os.MkdirAll(versionsDir, 0o755))

	writeFile(t, filepath.Join(imagesDir, "alpine", "image.yaml"), `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: alpine
versions:
  - version: "3"
    tags: ["3", "latest"]
    types: [default, dev]
`)
	writeFile(t, filepath.Join(versionsDir, "default.apko.yaml"), "contents:\n  packages: []\n")
	writeFile(t, filepath.Join(versionsDir, "dev.apko.yaml"), "contents:\n  packages: []\n")

	app := &cli.App{Commands: []*cli.Command{DiscoverCommand}}

	r, w, err := os.Pipe()
	require.NoError(t, err)

	origStdout := os.Stdout
	os.Stdout = w

	runErr := app.Run([]string{
		"integer", "discover",
		"--config", filepath.Join(dir, "integer.yaml"),
		"--images-dir", imagesDir,
	})

	w.Close()
	os.Stdout = origStdout

	require.NoError(t, runErr)

	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var captured []map[string]any
	require.NoError(t, json.Unmarshal(out, &captured))

	require.Len(t, captured, 2)

	types := make([]string, 0, len(captured))
	for _, entry := range captured {
		v, ok := entry["type"].(string)
		require.True(t, ok, "type field missing or not a string")
		types = append(types, v)
	}

	assert.ElementsMatch(t, []string{"default", "dev"}, types)
	assert.Equal(t, "ghcr.io/test-org", captured[0]["registry"])
}

func TestDiscoverCommand_MissingConfig(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{DiscoverCommand}}
	err := app.Run([]string{"integer", "discover", "--config", "/nonexistent/integer.yaml"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestDiscoverCommand_MissingImagesDir(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "integer.yaml"), `
apiVersion: integer.verity.supply/v1alpha1
kind: IntegerConfig
target:
  registry: ghcr.io/test-org
`)

	app := &cli.App{Commands: []*cli.Command{DiscoverCommand}}
	err := app.Run([]string{
		"integer", "discover",
		"--config", filepath.Join(dir, "integer.yaml"),
		"--images-dir", "/nonexistent/images",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to discover images")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
