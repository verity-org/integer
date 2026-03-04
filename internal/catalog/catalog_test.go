package catalog_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verity-org/integer/internal/catalog"
)

// writeFile creates a file with the given content in a temp directory derived
// from t.TempDir() and returns the full path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

const imageYAML = `apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: node
description: "Node.js runtime"
versions:
  - version: "22"
    tags: ["22"]
    types: [default, dev]
  - version: "24"
    latest: true
    tags: ["24", "latest"]
    types: [default, fips]
`

func TestGenerate_NoReports(t *testing.T) {
	imagesDir := t.TempDir()
	writeFile(t, imagesDir, "node/image.yaml", imageYAML)
	// _base should be skipped
	require.NoError(t, os.MkdirAll(filepath.Join(imagesDir, "_base"), 0o755))

	cat, err := catalog.Generate(imagesDir, "", "ghcr.io/verity-org")
	require.NoError(t, err)

	require.Len(t, cat.Images, 1)
	img := cat.Images[0]
	assert.Equal(t, "node", img.Name)
	assert.Equal(t, "Node.js runtime", img.Description)
	require.Len(t, img.Versions, 2)

	v22 := img.Versions[0]
	assert.Equal(t, "22", v22.Version)
	assert.False(t, v22.Latest)
	require.Len(t, v22.Variants, 2)

	def := v22.Variants[0]
	assert.Equal(t, "default", def.Type)
	assert.Equal(t, []string{"22"}, def.Tags)
	assert.Equal(t, "ghcr.io/verity-org/node:22", def.Ref)
	assert.Equal(t, "unknown", def.Status)
	assert.Empty(t, def.Digest)

	dev := v22.Variants[1]
	assert.Equal(t, "dev", dev.Type)
	assert.Equal(t, []string{"22-dev"}, dev.Tags)
	assert.Equal(t, "ghcr.io/verity-org/node:22-dev", dev.Ref)

	v24 := img.Versions[1]
	assert.True(t, v24.Latest)
	require.Len(t, v24.Variants, 2)
	assert.Equal(t, "default", v24.Variants[0].Type)
	assert.Equal(t, []string{"24", "latest"}, v24.Variants[0].Tags)
	assert.Equal(t, "fips", v24.Variants[1].Type)
	assert.Equal(t, []string{"24-fips", "latest-fips"}, v24.Variants[1].Tags)

	assert.Equal(t, "ghcr.io/verity-org", cat.Registry)
	assert.NotEmpty(t, cat.GeneratedAt)
}

func TestGenerate_WithReports(t *testing.T) {
	imagesDir := t.TempDir()
	reportsDir := t.TempDir()
	writeFile(t, imagesDir, "node/image.yaml", imageYAML)

	report := map[string]any{
		"digest":   "sha256:abc123",
		"status":   "success",
		"built_at": "2026-01-01T00:00:00Z",
		"tags":     []string{"22"},
	}
	reportData, err := json.Marshal(report)
	require.NoError(t, err)
	reportPath := filepath.Join(reportsDir, "node", "22", "default", "latest.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(reportPath), 0o755))
	require.NoError(t, os.WriteFile(reportPath, reportData, 0o644))

	cat, err := catalog.Generate(imagesDir, reportsDir, "ghcr.io/verity-org")
	require.NoError(t, err)

	v22 := cat.Images[0].Versions[0]
	def := v22.Variants[0]
	assert.Equal(t, "success", def.Status)
	assert.Equal(t, "sha256:abc123", def.Digest)
	assert.Equal(t, "2026-01-01T00:00:00Z", def.BuiltAt)

	// dev variant has no report → status stays "unknown"
	assert.Equal(t, "unknown", v22.Variants[1].Status)
}

func TestGenerate_SkipsNonDirectories(t *testing.T) {
	imagesDir := t.TempDir()
	writeFile(t, imagesDir, "node/image.yaml", imageYAML)
	// A plain file at the top level should be skipped without error
	writeFile(t, imagesDir, "README.md", "# images")

	cat, err := catalog.Generate(imagesDir, "", "ghcr.io/verity-org")
	require.NoError(t, err)
	assert.Len(t, cat.Images, 1)
}

func TestGenerate_SkipsDirectoriesWithoutImageYAML(t *testing.T) {
	imagesDir := t.TempDir()
	// Directory exists but no image.yaml
	require.NoError(t, os.MkdirAll(filepath.Join(imagesDir, "mystery"), 0o755))

	cat, err := catalog.Generate(imagesDir, "", "ghcr.io/verity-org")
	require.NoError(t, err)
	assert.Empty(t, cat.Images)
}

func TestGenerate_InvalidImagesDir(t *testing.T) {
	_, err := catalog.Generate("/nonexistent/path", "", "ghcr.io/verity-org")
	require.Error(t, err)
}

func TestGenerate_EmptyImagesDir(t *testing.T) {
	imagesDir := t.TempDir()
	cat, err := catalog.Generate(imagesDir, "", "ghcr.io/verity-org")
	require.NoError(t, err)
	assert.Empty(t, cat.Images)
}

func TestGenerate_CorruptReport(t *testing.T) {
	imagesDir := t.TempDir()
	reportsDir := t.TempDir()
	writeFile(t, imagesDir, "node/image.yaml", imageYAML)

	// Write a corrupt (non-JSON) report file
	reportPath := filepath.Join(reportsDir, "node", "22", "default", "latest.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(reportPath), 0o755))
	require.NoError(t, os.WriteFile(reportPath, []byte("not json"), 0o644))

	// Should not fail — corrupt report is silently skipped; status stays "unknown"
	cat, err := catalog.Generate(imagesDir, reportsDir, "ghcr.io/verity-org")
	require.NoError(t, err)
	assert.Equal(t, "unknown", cat.Images[0].Versions[0].Variants[0].Status)
}

func TestGenerate_MultipleImages(t *testing.T) {
	imagesDir := t.TempDir()
	const pythonYAML = `apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: python
description: "Python runtime"
versions:
  - version: "3.12"
    tags: ["3.12"]
    types: [default]
`
	writeFile(t, imagesDir, "node/image.yaml", imageYAML)
	writeFile(t, imagesDir, "python/image.yaml", pythonYAML)

	cat, err := catalog.Generate(imagesDir, "", "ghcr.io/verity-org")
	require.NoError(t, err)
	assert.Len(t, cat.Images, 2)
}

func TestGenerate_EOLField(t *testing.T) {
	imagesDir := t.TempDir()
	const yamlWithEOL = `apiVersion: integer.verity.supply/v1alpha1
kind: ImageDefinition
name: node
description: "Node.js"
versions:
  - version: "20"
    eol: "2026-04-30"
    tags: ["20"]
    types: [default]
`
	writeFile(t, imagesDir, "node/image.yaml", yamlWithEOL)

	cat, err := catalog.Generate(imagesDir, "", "ghcr.io/verity-org")
	require.NoError(t, err)
	assert.Equal(t, "2026-04-30", cat.Images[0].Versions[0].EOL)
}
