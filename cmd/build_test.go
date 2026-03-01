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

func TestFindApkoFile(t *testing.T) {
	def := &config.ImageDefinition{
		Name: "node",
		Versions: []config.VersionDef{
			{Version: "20", Tags: []string{"20"}, Types: []string{"default"}},
			{Version: "22", Tags: []string{"22", "latest"}, Types: []string{"default", "dev", "fips"}},
		},
	}

	t.Run("finds default type for version 22", func(t *testing.T) {
		path, err := findApkoFile(def, "22", "default", "/images/node")
		require.NoError(t, err)
		assert.Equal(t, "/images/node/versions/22/default.apko.yaml", path)
	})

	t.Run("finds dev type for version 22", func(t *testing.T) {
		path, err := findApkoFile(def, "22", "dev", "/images/node")
		require.NoError(t, err)
		assert.Equal(t, "/images/node/versions/22/dev.apko.yaml", path)
	})

	t.Run("finds default type for version 20", func(t *testing.T) {
		path, err := findApkoFile(def, "20", "default", "/images/node")
		require.NoError(t, err)
		assert.Equal(t, "/images/node/versions/20/default.apko.yaml", path)
	})

	t.Run("returns error for unknown type", func(t *testing.T) {
		_, err := findApkoFile(def, "22", "jre", "/images/node")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrVariantNotFound)
		assert.Contains(t, err.Error(), "jre")
		assert.Contains(t, err.Error(), "22")
	})

	t.Run("returns error for unknown version", func(t *testing.T) {
		_, err := findApkoFile(def, "18", "default", "/images/node")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrVariantNotFound)
		assert.Contains(t, err.Error(), "18")
		assert.Contains(t, err.Error(), "node")
	})

	t.Run("returns error for empty versions", func(t *testing.T) {
		empty := &config.ImageDefinition{Name: "empty"}
		_, err := findApkoFile(empty, "1", "default", "/images/empty")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrVariantNotFound)
	})
}

func TestBuildCommand_ApkoNotFound(t *testing.T) {
	dir := t.TempDir()
	imagesDir := filepath.Join(dir, "images")
	versionsDir := filepath.Join(imagesDir, "myapp", "versions", "1")
	require.NoError(t, os.MkdirAll(versionsDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(imagesDir, "myapp", "image.yaml"), []byte(`
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: myapp
versions:
  - version: "1"
    tags: ["latest"]
    types: [default]
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionsDir, "default.apko.yaml"), []byte("contents:\n  packages: []\n"), 0o644))

	// Ensure apko is not resolvable by stripping PATH.
	t.Setenv("PATH", "")

	app := &cli.App{Commands: []*cli.Command{BuildCommand}}
	err := app.Run([]string{
		"integer", "build",
		"--image", "myapp",
		"--version", "1",
		"--type", "default",
		"--images-dir", imagesDir,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apko not found")
}

func TestBuildCommand_MissingImage(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{BuildCommand}}
	err := app.Run([]string{
		"integer", "build",
		"--image", "nonexistent",
		"--version", "1",
		"--images-dir", "/nonexistent/images",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load image definition")
}

func TestBuildCommand_UnknownVersion(t *testing.T) {
	dir := t.TempDir()
	imagesDir := filepath.Join(dir, "images")
	versionsDir := filepath.Join(imagesDir, "myapp", "versions", "1")
	require.NoError(t, os.MkdirAll(versionsDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(imagesDir, "myapp", "image.yaml"), []byte(`
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: myapp
versions:
  - version: "1"
    tags: ["latest"]
    types: [default]
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionsDir, "default.apko.yaml"), []byte("contents:\n  packages: []\n"), 0o644))

	app := &cli.App{Commands: []*cli.Command{BuildCommand}}
	err := app.Run([]string{
		"integer", "build",
		"--image", "myapp",
		"--version", "99",
		"--images-dir", imagesDir,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVariantNotFound)
}

func TestRunApkoBuild_NotInPath(t *testing.T) {
	t.Setenv("PATH", "")
	err := runApkoBuild(context.Background(), "config.yaml", "out.tar", "amd64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apko not found")
}
