package cmd

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/urfave/cli/v2"

	"github.com/verity-org/integer/internal/apkindex"
	"github.com/verity-org/integer/internal/config"
)

// SyncCommand fetches the Wolfi APKINDEX and reports new or stale versions
// relative to each image's versions map. With --apply it writes updates.
var SyncCommand = &cli.Command{
	Name:  "sync",
	Usage: "Fetch APKINDEX and report new/stale versions; --apply updates image files",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "images-dir",
			Usage: "Path to the images/ directory",
			Value: "images",
		},
		&cli.StringFlag{
			Name:  "apkindex-url",
			Usage: "Wolfi APKINDEX URL",
			Value: apkindex.DefaultAPKINDEXURL,
		},
		&cli.StringFlag{
			Name:  "cache-dir",
			Usage: "Directory for caching APKINDEX data",
			Value: os.TempDir(),
		},
		&cli.BoolFlag{
			Name:  "apply",
			Usage: "Write new versions back into image YAML files",
		},
	},
	Action: runSync,
}

func runSync(c *cli.Context) error {
	imagesDir := c.String("images-dir")
	apply := c.Bool("apply")

	var pkgs []apkindex.Package
	if url := c.String("apkindex-url"); url != "" {
		var err error
		pkgs, err = apkindex.Fetch(url, c.String("cache-dir"), apkindex.DefaultCacheMaxAge)
		if err != nil {
			return fmt.Errorf("fetching APKINDEX: %w", err)
		}
	}

	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		return fmt.Errorf("reading images directory: %w", err)
	}

	totalNew, totalStale := 0, 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		n, s := processSyncEntry(entry, imagesDir, pkgs, apply)
		totalNew += n
		totalStale += s
	}

	fmt.Fprintf(os.Stdout, "\nSummary: %d new, %d stale\n", totalNew, totalStale)
	return nil
}

// processSyncEntry processes a single image YAML entry during sync and returns
// the count of new and stale versions found.
func processSyncEntry(entry os.DirEntry, imagesDir string, pkgs []apkindex.Package, apply bool) (newCount, staleCount int) {
	defPath := filepath.Join(imagesDir, entry.Name())
	def, err := config.LoadImage(defPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN %s: %v\n", entry.Name(), err)
		return 0, 0
	}

	discovered := apkindex.DiscoverVersions(pkgs, def.Upstream.Package)

	// Build sets for comparison.
	discoveredSet := make(map[string]bool, len(discovered))
	for _, v := range discovered {
		discoveredSet[v] = true
	}

	knownSet := make(map[string]bool, len(def.Versions))
	for v := range def.Versions {
		knownSet[v] = true
	}

	var newVersions, staleVersions []string
	for _, v := range discovered {
		if !knownSet[v] {
			newVersions = append(newVersions, v)
		}
	}
	for v := range def.Versions {
		if !discoveredSet[v] && v != "latest" {
			staleVersions = append(staleVersions, v)
		}
	}
	sort.Strings(newVersions)
	sort.Strings(staleVersions)

	printSyncReport(def.Name, newVersions, staleVersions)

	if apply && len(newVersions) > 0 {
		if err := applySyncUpdates(defPath, def, newVersions); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR applying updates to %s: %v\n", entry.Name(), err)
		}
	}

	return len(newVersions), len(staleVersions)
}

func printSyncReport(name string, newVersions, staleVersions []string) {
	if len(newVersions) == 0 && len(staleVersions) == 0 {
		fmt.Fprintf(os.Stdout, "%s: up to date\n", name)
		return
	}
	fmt.Fprintf(os.Stdout, "%s:\n", name)
	if len(newVersions) > 0 {
		fmt.Fprintf(os.Stdout, "  new:   %s\n", strings.Join(newVersions, ", "))
	}
	if len(staleVersions) > 0 {
		fmt.Fprintf(os.Stdout, "  stale: %s\n", strings.Join(staleVersions, ", "))
	}
}

// applySyncUpdates adds newVersions to def.Versions and writes the YAML back.
// Existing fields are preserved; new versions get an empty VersionMeta.
func applySyncUpdates(path string, def *config.ImageDef, newVersions []string) error {
	updated := *def
	versions := make(map[string]config.VersionMeta, len(def.Versions)+len(newVersions))
	maps.Copy(versions, def.Versions)
	for _, v := range newVersions {
		versions[v] = config.VersionMeta{}
	}
	updated.Versions = versions

	data, err := yaml.Marshal(&updated)
	if err != nil {
		return fmt.Errorf("marshalling: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	fmt.Fprintf(os.Stdout, "  applied: added %s\n", strings.Join(newVersions, ", "))
	return nil
}
