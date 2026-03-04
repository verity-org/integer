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

const testIntegerYAML = `
target:
  registry: ghcr.io/test-org
defaults:
  archs: [amd64, arm64]
`

const testNodeYAML = `
name: node
description: "Node.js"
upstream:
  package: "nodejs-{{version}}"
types:
  default:
    base: wolfi-base
    packages: ["nodejs-{{version}}"]
    entrypoint: /usr/bin/node
  dev:
    base: wolfi-dev
    packages: ["nodejs-{{version}}", "npm"]
    entrypoint: /usr/bin/node
versions:
  "22": {}
`

// writeFile is a shared helper for all cmd test files.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func setupCmdImages(t *testing.T) (imagesDir, cfgPath string) {
	t.Helper()
	dir := t.TempDir()
	cfgPath = filepath.Join(dir, "integer.yaml")
	writeFile(t, cfgPath, testIntegerYAML)

	imagesDir = filepath.Join(dir, "images")
	writeFile(t, filepath.Join(imagesDir, "_base", "wolfi-base.yaml"), "# base\n")
	writeFile(t, filepath.Join(imagesDir, "_base", "wolfi-dev.yaml"), "# base\n")
	writeFile(t, filepath.Join(imagesDir, "_base", "wolfi-fips.yaml"), "# base\n")
	writeFile(t, filepath.Join(imagesDir, "node.yaml"), testNodeYAML)
	return imagesDir, cfgPath
}

func TestDiscoverCommand(t *testing.T) {
	imagesDir, cfgPath := setupCmdImages(t)
	genDir := t.TempDir()

	app := &cli.App{Commands: []*cli.Command{DiscoverCommand}}

	r, w, err := os.Pipe()
	require.NoError(t, err)

	origStdout := os.Stdout
	os.Stdout = w

	runErr := app.Run([]string{
		"integer", "discover",
		"--config", cfgPath,
		"--images-dir", imagesDir,
		"--apkindex-url", "", // disable network fetch
		"--gen-dir", genDir,
	})

	w.Close()
	os.Stdout = origStdout

	require.NoError(t, runErr)

	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var captured []map[string]any
	require.NoError(t, json.Unmarshal(out, &captured))

	// 1 version × 2 types = 2 entries
	require.Len(t, captured, 2)

	types := make([]string, 0, len(captured))
	for _, entry := range captured {
		v, ok := entry["type"].(string)
		require.True(t, ok)
		types = append(types, v)
	}
	assert.ElementsMatch(t, []string{"default", "dev"}, types)
	assert.Equal(t, "ghcr.io/test-org", captured[0]["registry"])
}

func TestDiscoverCommand_MissingConfig(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{DiscoverCommand}}
	err := app.Run([]string{"integer", "discover", "--config", "/nonexistent/integer.yaml"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading config")
}

func TestDiscoverCommand_MissingImagesDir(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "integer.yaml")
	writeFile(t, cfgPath, testIntegerYAML)

	app := &cli.App{Commands: []*cli.Command{DiscoverCommand}}
	err := app.Run([]string{
		"integer", "discover",
		"--config", cfgPath,
		"--images-dir", "/nonexistent/images",
		"--apkindex-url", "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "discovering images")
}
