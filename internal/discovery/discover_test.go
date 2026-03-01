package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verity-org/integer/internal/discovery"
)

func TestDiscover(t *testing.T) {
	t.Run("discovers all version×type combos from a valid images directory", func(t *testing.T) {
		dir := t.TempDir()
		setupImageDir(t, dir, "node", `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: node
description: "Node.js runtime"
upstream:
  package: nodejs-22
versions:
  - version: "22"
    tags: ["22", "latest"]
    types: [default, dev]
`, []string{"versions/22/default.apko.yaml", "versions/22/dev.apko.yaml"})

		imgs, err := discovery.Discover(dir, "ghcr.io/verity-org")
		require.NoError(t, err)
		require.Len(t, imgs, 2)

		assert.Equal(t, "node", imgs[0].Name)
		assert.Equal(t, "22", imgs[0].Version)
		assert.Equal(t, "default", imgs[0].Type)
		assert.Equal(t, []string{"22", "latest"}, imgs[0].Tags)
		assert.Equal(t, "ghcr.io/verity-org", imgs[0].Registry)

		assert.Equal(t, "node", imgs[1].Name)
		assert.Equal(t, "22", imgs[1].Version)
		assert.Equal(t, "dev", imgs[1].Type)
		assert.Equal(t, []string{"22-dev", "latest-dev"}, imgs[1].Tags)
	})

	t.Run("multi-version image produces correct entries", func(t *testing.T) {
		dir := t.TempDir()
		setupImageDir(t, dir, "node", `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: node
versions:
  - version: "20"
    tags: ["20"]
    types: [default]
  - version: "22"
    tags: ["22", "latest"]
    types: [default, dev]
`, []string{
			"versions/20/default.apko.yaml",
			"versions/22/default.apko.yaml",
			"versions/22/dev.apko.yaml",
		})

		imgs, err := discovery.Discover(dir, "ghcr.io/verity-org")
		require.NoError(t, err)
		require.Len(t, imgs, 3)
	})

	t.Run("skips _base directory", func(t *testing.T) {
		dir := t.TempDir()
		baseDir := filepath.Join(dir, "_base")
		require.NoError(t, os.MkdirAll(baseDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(baseDir, "wolfi-base.yaml"), []byte("contents: {}"), 0o644))

		setupImageDir(t, dir, "python", `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: python
versions:
  - version: "3.12"
    tags: ["3.12", "latest"]
    types: [default]
`, []string{"versions/3.12/default.apko.yaml"})

		imgs, err := discovery.Discover(dir, "ghcr.io/verity-org")
		require.NoError(t, err)
		require.Len(t, imgs, 1)
		assert.Equal(t, "python", imgs[0].Name)
	})

	t.Run("skips directories without image.yaml", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "orphan"), 0o755))

		setupImageDir(t, dir, "nginx", `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: nginx
versions:
  - version: "1"
    tags: ["latest"]
    types: [default]
`, []string{"versions/1/default.apko.yaml"})

		imgs, err := discovery.Discover(dir, "ghcr.io/verity-org")
		require.NoError(t, err)
		require.Len(t, imgs, 1)
		assert.Equal(t, "nginx", imgs[0].Name)
	})

	t.Run("discovers multiple images", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"node", "python", "nginx"} {
			setupImageDir(t, dir, name, imageYAML(name), []string{"versions/1/default.apko.yaml"})
		}

		imgs, err := discovery.Discover(dir, "ghcr.io/verity-org")
		require.NoError(t, err)
		assert.Len(t, imgs, 3)
	})

	t.Run("returns error for missing apko file", func(t *testing.T) {
		dir := t.TempDir()
		imageDir := filepath.Join(dir, "broken")
		require.NoError(t, os.MkdirAll(imageDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(imageDir, "image.yaml"), []byte(`
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: broken
versions:
  - version: "1"
    tags: ["latest"]
    types: [default]
`), 0o644))
		// No versions/1/default.apko.yaml created

		_, err := discovery.Discover(dir, "ghcr.io/verity-org")
		require.Error(t, err)
		assert.ErrorIs(t, err, discovery.ErrVariantFileMissing)
	})

	t.Run("tags are independent copies", func(t *testing.T) {
		dir := t.TempDir()
		setupImageDir(t, dir, "redis", `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: redis
versions:
  - version: "7"
    tags: ["7", "latest"]
    types: [default]
`, []string{"versions/7/default.apko.yaml"})

		imgs, err := discovery.Discover(dir, "ghcr.io/verity-org")
		require.NoError(t, err)
		require.Len(t, imgs, 1)

		// Mutating the returned tags must not affect a second call.
		imgs[0].Tags[0] = "MUTATED"
		imgs2, err := discovery.Discover(dir, "ghcr.io/verity-org")
		require.NoError(t, err)
		assert.Equal(t, "7", imgs2[0].Tags[0])
	})
}

func TestDeriveTags(t *testing.T) {
	t.Run("default type returns base tags unchanged", func(t *testing.T) {
		dir := t.TempDir()
		setupImageDir(t, dir, "node", `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: node
versions:
  - version: "22"
    tags: ["22", "latest"]
    types: [default]
`, []string{"versions/22/default.apko.yaml"})

		imgs, err := discovery.Discover(dir, "ghcr.io/verity-org")
		require.NoError(t, err)
		require.Len(t, imgs, 1)
		assert.Equal(t, []string{"22", "latest"}, imgs[0].Tags)
	})

	t.Run("non-default type appends suffix to each tag", func(t *testing.T) {
		dir := t.TempDir()
		setupImageDir(t, dir, "node", `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: node
versions:
  - version: "22"
    tags: ["22", "latest"]
    types: [dev, fips]
`, []string{"versions/22/dev.apko.yaml", "versions/22/fips.apko.yaml"})

		imgs, err := discovery.Discover(dir, "ghcr.io/verity-org")
		require.NoError(t, err)
		require.Len(t, imgs, 2)
		assert.Equal(t, []string{"22-dev", "latest-dev"}, imgs[0].Tags)
		assert.Equal(t, []string{"22-fips", "latest-fips"}, imgs[1].Tags)
	})
}

func TestWalkApkoFiles(t *testing.T) {
	t.Run("returns all apko yaml files", func(t *testing.T) {
		dir := t.TempDir()
		setupImageDir(t, dir, "node", imageYAML("node"), []string{
			"versions/22/default.apko.yaml",
			"versions/22/dev.apko.yaml",
		})

		files, err := discovery.WalkApkoFiles(dir)
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})

	t.Run("excludes _base directory", func(t *testing.T) {
		dir := t.TempDir()
		baseDir := filepath.Join(dir, "_base")
		require.NoError(t, os.MkdirAll(baseDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(baseDir, "wolfi-base.yaml"), []byte("{}"), 0o644))

		setupImageDir(t, dir, "nginx", imageYAML("nginx"), []string{"versions/1/default.apko.yaml"})

		files, err := discovery.WalkApkoFiles(dir)
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("excludes image.yaml files", func(t *testing.T) {
		dir := t.TempDir()
		setupImageDir(t, dir, "python", imageYAML("python"), []string{"versions/3.12/default.apko.yaml"})

		files, err := discovery.WalkApkoFiles(dir)
		require.NoError(t, err)
		for _, f := range files {
			assert.NotEqual(t, "image.yaml", filepath.Base(f))
		}
	})
}

// setupImageDir creates an image directory with image.yaml and stub apko files.
func setupImageDir(t *testing.T, imagesDir, name, imageYAMLContent string, apkoFiles []string) {
	t.Helper()
	imageDir := filepath.Join(imagesDir, name)
	require.NoError(t, os.MkdirAll(imageDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(imageDir, "image.yaml"), []byte(imageYAMLContent), 0o644))
	for _, af := range apkoFiles {
		path := filepath.Join(imageDir, af)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte("contents: {}"), 0o644))
	}
}

func imageYAML(name string) string {
	return `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: ` + name + `
versions:
  - version: "1"
    tags: ["latest"]
    types: [default]
`
}
