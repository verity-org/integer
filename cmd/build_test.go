package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/verity-org/integer/internal/config"
)

const buildNodeYAML = `
name: myapp
upstream:
  package: myapp
types:
  default:
    base: wolfi-base
    packages: [myapp]
    entrypoint: /usr/bin/myapp
versions:
  latest:
    latest: true
`

func setupBuildImages(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	imagesDir := filepath.Join(dir, "images")
	baseDir := filepath.Join(imagesDir, "_base")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "wolfi-base.yaml"), []byte("# base\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(imagesDir, "myapp.yaml"), []byte(buildNodeYAML), 0o644))
	return imagesDir
}

// fakeApko writes a shell script into a temp dir and puts it first in PATH.
// exitCode controls whether the fake apko succeeds (0) or fails (non-zero).
func fakeApko(t *testing.T, exitCode int) {
	t.Helper()
	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "apko")
	content := "#!/bin/sh\nexit " + string(rune('0'+exitCode)) + "\n"
	require.NoError(t, os.WriteFile(script, []byte(content), 0o755))
	existing := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+existing)
}

func TestBuildCommand_UnknownType(t *testing.T) {
	imagesDir := setupBuildImages(t)

	app := &cli.App{Commands: []*cli.Command{BuildCommand}}
	err := app.Run([]string{
		"integer", "build",
		"--image", "myapp",
		"--version", "latest",
		"--type", "jre", // does not exist
		"--images-dir", imagesDir,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVariantNotFound)
}

func TestBuildCommand_MissingImage(t *testing.T) {
	imagesDir := setupBuildImages(t)

	app := &cli.App{Commands: []*cli.Command{BuildCommand}}
	err := app.Run([]string{
		"integer", "build",
		"--image", "nonexistent",
		"--version", "latest",
		"--images-dir", imagesDir,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestRunApkoBuild_NotInPath(t *testing.T) {
	t.Setenv("PATH", "")
	err := runApkoBuild(context.Background(), "config.yaml", "out.tar", "amd64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apko not found")
}

func TestRunApkoBuild_Fails(t *testing.T) {
	fakeApko(t, 1)
	err := runApkoBuild(context.Background(), "config.yaml", "out.tar", "amd64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apko build failed")
}

func TestRunApkoBuild_Success(t *testing.T) {
	fakeApko(t, 0)
	err := runApkoBuild(context.Background(), "config.yaml", "out.tar", "amd64")
	require.NoError(t, err)
}

func TestResolveLatestVersion_Success(t *testing.T) {
	srv := makeAPKINDEXServer(t, "P:nodejs-22\nV:22.0.0\n\nP:nodejs-24\nV:24.0.0\n\n")

	def := &config.ImageDef{
		Name:     "node",
		Upstream: config.Upstream{Package: "nodejs-{{version}}"},
	}

	v, err := resolveLatestVersion(def, srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "24", v)
}

func TestResolveLatestVersion_NoVersions(t *testing.T) {
	srv := makeAPKINDEXServer(t, "P:curl\nV:8.0.0\n\n") // no nodejs packages

	def := &config.ImageDef{
		Name:     "node",
		Upstream: config.Upstream{Package: "nodejs-{{version}}"},
	}

	_, err := resolveLatestVersion(def, srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no versions found")
}

func TestResolveLatestVersion_FetchError(t *testing.T) {
	def := &config.ImageDef{
		Name:     "node",
		Upstream: config.Upstream{Package: "nodejs-{{version}}"},
	}

	_, err := resolveLatestVersion(def, "http://127.0.0.1:0/bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetching APKINDEX")
}
