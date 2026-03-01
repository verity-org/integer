package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verity-org/integer/internal/config"
)

func TestLoadIntegerConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		path := writeFile(t, "integer.yaml", `
apiVersion: integer.verity.supply/v1alpha1
kind: IntegerConfig
target:
  registry: ghcr.io/verity-org
defaults:
  archs:
    - amd64
    - arm64
`)
		cfg, err := config.LoadIntegerConfig(path)
		require.NoError(t, err)
		assert.Equal(t, "integer.verity.supply/v1alpha1", cfg.APIVersion)
		assert.Equal(t, "IntegerConfig", cfg.Kind)
		assert.Equal(t, "ghcr.io/verity-org", cfg.Target.Registry)
		assert.Equal(t, []string{"amd64", "arm64"}, cfg.Defaults.Archs)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := config.LoadIntegerConfig("/nonexistent/integer.yaml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading integer config")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := writeFile(t, "bad.yaml", "{ invalid: yaml: content")
		_, err := config.LoadIntegerConfig(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing integer config")
	})
}

func TestLoadImageDefinition(t *testing.T) {
	t.Run("multi-version image", func(t *testing.T) {
		path := writeFile(t, "image.yaml", `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: node
description: "Node.js runtime"
eol-product: nodejs
versions:
  - version: "20"
    eol: "2026-04-30"
    tags: ["20"]
    types: [default, dev, fips]
  - version: "22"
    eol: "2027-04-30"
    latest: true
    tags: ["22", "latest"]
    types: [default, dev, fips]
`)
		def, err := config.LoadImageDefinition(path)
		require.NoError(t, err)
		assert.Equal(t, "node", def.Name)
		assert.Equal(t, "Node.js runtime", def.Description)
		assert.Equal(t, "nodejs", def.EOLProduct)
		require.Len(t, def.Versions, 2)
		assert.Equal(t, "20", def.Versions[0].Version)
		assert.Equal(t, "2026-04-30", def.Versions[0].EOL)
		assert.False(t, def.Versions[0].Latest)
		assert.Equal(t, []string{"default", "dev", "fips"}, def.Versions[0].Types)
		assert.Equal(t, "22", def.Versions[1].Version)
		assert.True(t, def.Versions[1].Latest)
		assert.Equal(t, []string{"22", "latest"}, def.Versions[1].Tags)
	})

	t.Run("single-version image", func(t *testing.T) {
		path := writeFile(t, "image.yaml", `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: nginx
description: "Nginx web server"
versions:
  - version: "1"
    latest: true
    tags: ["1", "latest"]
    types: [default, fips]
`)
		def, err := config.LoadImageDefinition(path)
		require.NoError(t, err)
		assert.Equal(t, "nginx", def.Name)
		require.Len(t, def.Versions, 1)
		assert.Equal(t, "1", def.Versions[0].Version)
		assert.Equal(t, []string{"default", "fips"}, def.Versions[0].Types)
	})

	t.Run("image without upstream", func(t *testing.T) {
		path := writeFile(t, "image.yaml", `
apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: custom
description: "Custom image"
versions:
  - version: "1"
    latest: true
    tags: ["latest"]
    types: [default]
`)
		def, err := config.LoadImageDefinition(path)
		require.NoError(t, err)
		assert.Equal(t, "custom", def.Name)
		assert.Empty(t, def.Upstream.Package)
		require.Len(t, def.Versions, 1)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := config.LoadImageDefinition("/nonexistent/image.yaml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading image definition")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := writeFile(t, "bad.yaml", ": bad: yaml:")
		_, err := config.LoadImageDefinition(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing image definition")
	})
}

// writeFile creates a temp file with the given content and returns its path.
func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}
