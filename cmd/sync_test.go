package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/verity-org/integer/internal/config"
)

func runSyncApp(t *testing.T, args []string) error {
	t.Helper()
	app := &cli.App{Commands: []*cli.Command{SyncCommand}}
	return app.Run(append([]string{"integer"}, args...))
}

// makeAPKINDEXServer returns an httptest.Server serving a minimal APKINDEX.tar.gz.
func makeAPKINDEXServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		data := []byte(content)
		_ = tw.WriteHeader(&tar.Header{Name: "APKINDEX", Mode: 0o644, Size: int64(len(data))}) //nolint:errcheck // test helper, errors not meaningful
		_, _ = tw.Write(data)                                                                  //nolint:errcheck // test helper, errors not meaningful
		tw.Close()
		gz.Close()
		_, _ = w.Write(buf.Bytes()) //nolint:errcheck // test helper, errors not meaningful
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSyncCommand_UpToDate_LatestVersion(t *testing.T) {
	dir := t.TempDir()
	imagesDir := filepath.Join(dir, "images")

	// Image with version "latest" — not reported as stale when APKINDEX is empty.
	const curlYAML = `
name: curl
upstream:
  package: curl
types:
  default:
    base: wolfi-base
    packages: [curl]
    entrypoint: /usr/bin/curl
versions:
  latest:
    latest: true
`
	writeFile(t, filepath.Join(imagesDir, "_base", "wolfi-base.yaml"), "# base\n")
	writeFile(t, filepath.Join(imagesDir, "curl.yaml"), curlYAML)

	err := runSyncApp(t, []string{
		"sync",
		"--images-dir", imagesDir,
		"--apkindex-url", "",
	})
	require.NoError(t, err)
}

func TestSyncCommand_WithAPKINDEX_NewVersion(t *testing.T) {
	srv := makeAPKINDEXServer(t, "P:nodejs-22\nV:22.0.0\n\nP:nodejs-24\nV:24.0.0\n\n")

	dir := t.TempDir()
	imagesDir := filepath.Join(dir, "images")
	writeFile(t, filepath.Join(imagesDir, "_base", "wolfi-base.yaml"), "# base\n")
	writeFile(t, filepath.Join(imagesDir, "node.yaml"), testNodeYAML) // only has "22"

	err := runSyncApp(t, []string{
		"sync",
		"--images-dir", imagesDir,
		"--apkindex-url", srv.URL,
		"--cache-dir", t.TempDir(),
	})
	require.NoError(t, err)
}

func TestSyncCommand_WithAPKINDEX_Apply(t *testing.T) {
	srv := makeAPKINDEXServer(t, "P:nodejs-22\nV:22.0.0\n\nP:nodejs-24\nV:24.0.0\n\n")

	dir := t.TempDir()
	imagesDir := filepath.Join(dir, "images")
	writeFile(t, filepath.Join(imagesDir, "_base", "wolfi-base.yaml"), "# base\n")
	writeFile(t, filepath.Join(imagesDir, "node.yaml"), testNodeYAML) // only has "22"

	err := runSyncApp(t, []string{
		"sync",
		"--images-dir", imagesDir,
		"--apkindex-url", srv.URL,
		"--cache-dir", t.TempDir(),
		"--apply",
	})
	require.NoError(t, err)

	// node.yaml should now contain "24".
	updated, err := config.LoadImage(filepath.Join(imagesDir, "node.yaml"))
	require.NoError(t, err)
	assert.Contains(t, updated.Versions, "22")
	assert.Contains(t, updated.Versions, "24")
}

func TestSyncCommand_MissingImagesDir(t *testing.T) {
	err := runSyncApp(t, []string{
		"sync",
		"--images-dir", "/nonexistent/images",
		"--apkindex-url", "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading images directory")
}

func TestSyncCommand_APKINDEXFetchFails(t *testing.T) {
	err := runSyncApp(t, []string{
		"sync",
		"--images-dir", "/nonexistent/images",
		"--apkindex-url", "http://127.0.0.1:0/bad",
	})
	require.Error(t, err)
}

func TestSyncCommand_ApplyAddsNewVersions(t *testing.T) {
	dir := t.TempDir()
	imagesDir := filepath.Join(dir, "images")

	const imageYAML = `
name: node
upstream:
  package: "nodejs-{{version}}"
types:
  default:
    base: wolfi-base
    packages: ["nodejs-{{version}}"]
    entrypoint: /usr/bin/node
versions:
  "22": {}
`
	writeFile(t, filepath.Join(imagesDir, "_base", "wolfi-base.yaml"), "# base\n")
	writeFile(t, filepath.Join(imagesDir, "node.yaml"), imageYAML)

	imagePath := filepath.Join(imagesDir, "node.yaml")
	def, err := config.LoadImage(imagePath)
	require.NoError(t, err)

	err = applySyncUpdates(imagePath, def, []string{"24", "26"})
	require.NoError(t, err)

	updated, err := config.LoadImage(imagePath)
	require.NoError(t, err)
	assert.Contains(t, updated.Versions, "22")
	assert.Contains(t, updated.Versions, "24")
	assert.Contains(t, updated.Versions, "26")
}

func TestSyncCommand_SkipsNonYAML(t *testing.T) {
	imagesDir, _ := setupCmdImages(t)
	writeFile(t, filepath.Join(imagesDir, "README.md"), "# readme")

	err := runSyncApp(t, []string{
		"sync",
		"--images-dir", imagesDir,
		"--apkindex-url", "",
	})
	require.NoError(t, err)
}
