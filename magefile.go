//go:build mage

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dio/sh"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

var FORCED = len(os.Getenv("FORCED")) > 0
var DRY_RUN = len(os.Getenv("DRY_RUN")) > 0
var GH_TOKEN = os.Getenv("GH_TOKEN")
var GPG_PASSPHRASE = os.Getenv("GPG_PASSPHRASE")
var GPG_TRUSTEE = "trustee@tetrate-istio-subscription.iam.gserviceaccount.com"
var ENV_MAP = map[string]string{
	"GH_TOKEN": GH_TOKEN,
	"TZ":       "UTC",
}

// PackIstio packs versioned Istio Helm charts.
func PackIstio(ctx context.Context) error {
	return packCharts(ctx, filepath.Join("charts", "istio"), packVersionedIstio)
}

// PackAddons packs addons Helm charts.
func PackAddons(ctx context.Context) error {
	return packCharts(ctx, filepath.Join("charts", "addons"), packChart)
}

// PackDemos packs demos Helm charts.
func PackDemos(ctx context.Context) error {
	return packCharts(ctx, filepath.Join("charts", "demos"), packChart)
}

// PackSystem packs system Helm charts.
func PackSystem(ctx context.Context) error {
	return packCharts(ctx, filepath.Join("charts", "system"), packChart)
}

// Index generates Helm charts index, and optionally merge with existing index.yaml.
// URL gives prefix for the tarballs. For example: https://github.com/tetratelabs/helm-charts/releases/download.
func Index(ctx context.Context, url string) error {
	dir := filepath.Join("dist")
	if _, err := os.Stat(dir); err != nil {
		fmt.Println("missing dist, nothing to do")
		return nil
	}
	indexYAML := filepath.Join(dir, "index.yaml")
	publishedIndexYAML := filepath.Join("gh-pages", "index.yaml")
	args := []string{
		"repo",
		"index",
		dir,
		"--url", url,
	}
	if _, err := os.Stat(publishedIndexYAML); err == nil {
		args = append(args, "--merge", publishedIndexYAML)
	}
	if err := sh.RunWithV(ctx, ENV_MAP, "helm", args...); err != nil {
		return err
	}
	b, err := os.ReadFile(indexYAML)
	if err != nil {
		return err
	}

	// TODO(dio): Remove this hack. We should call the modified "helm repo index" function.
	sanitized := strings.ReplaceAll(string(b), "download/1.", "download/istio-1.")
	_ = os.MkdirAll("gh-pages", os.ModePerm)
	return os.WriteFile(publishedIndexYAML, []byte(sanitized), os.ModePerm)
}

// packCharts packs all charts inside a directory.
func packCharts(ctx context.Context, dir string,
	packer func(context.Context, string, string, string) error) error {
	// Export keyring.
	ring, pass, err := exportSecretKey(ctx)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := packer(ctx, filepath.Join(dir, entry.Name()), ring, pass); err != nil {
			return err
		}
	}
	return nil
}

// packChart packs a chart.
func packChart(ctx context.Context, dir, ring, pass string) error {
	charts, err := filePaths(dir,
		"Chart.yaml", "*/Chart.yaml", "*/*/Chart.yaml", "*/*/*/Chart.yaml", "*/*/*/*/Chart.yaml")
	if err != nil {
		return err
	}

	for _, chart := range charts {
		metadata, err := chartutil.LoadChartfile(filepath.Join(chart))
		if err != nil {
			return err
		}

		if err := resolveDeps(ctx, filepath.Dir(chart), metadata); err != nil {
			return err
		}

		tag := metadata.Name + "-" + metadata.Version

		// If already published, skip it unless FORCED.
		err = sh.RunWithV(ctx, ENV_MAP, "gh", "release", "view", tag)
		released := err == nil
		if released && !FORCED {
			fmt.Printf("%s is already published\n", tag)
			return nil
		}

		dst := filepath.Join("dist", metadata.Name+"-"+metadata.Version)
		_ = os.MkdirAll(dst, os.ModePerm)

		if err := sh.RunWithV(ctx,
			ENV_MAP,
			"helm",
			"package",
			filepath.Dir(chart),
			"--destination", dst,
			"--sign",
			"--keyring", ring,
			"--key", GPG_TRUSTEE,
			"--passphrase-file", pass,
		); err != nil {
			return err
		}

		entries, err := os.ReadDir(dst)
		if err != nil {
			return err
		}

		files := []string{}
		for _, entry := range entries {
			files = append(files, filepath.Join(dst, entry.Name()))
		}

		if DRY_RUN {
			fmt.Println("tag", tag, "files", files)
			continue
		}

		if released {
			return sh.RunWithV(ctx, ENV_MAP, "gh", append([]string{"release", "upload", tag, "--clobber"}, files...)...)
		}
		return sh.RunWithV(ctx, ENV_MAP, "gh", append([]string{"release", "create", tag, "-n", tag, "-t", tag}, files...)...)
	}

	return nil
}

// packVersionedIstio packs a versioned Istio.
func packVersionedIstio(ctx context.Context, dir, ring, pass string) error {
	version := filepath.Base(dir)
	tag := "istio-" + version

	// If already published, skip it unless FORCED.
	err := sh.RunWithV(ctx, ENV_MAP, "gh", "release", "view", tag)
	released := err == nil
	if released && !FORCED {
		fmt.Printf("%s is already published\n", tag)
		return nil
	}

	dst := filepath.Join("dist", tag)
	_ = os.MkdirAll(dst, os.ModePerm)

	charts, err := filePaths(dir,
		"Chart.yaml", "*/Chart.yaml", "*/*/Chart.yaml", "*/*/*/Chart.yaml", "*/*/*/*/Chart.yaml")
	if err != nil {
		return err
	}
	for _, chart := range charts {
		if err := sh.RunWithV(ctx,
			ENV_MAP,
			"helm",
			"package",
			filepath.Dir(chart),
			"--destination", dst,
			"--sign",
			"--keyring", ring,
			"--key", GPG_TRUSTEE,
			"--passphrase-file", pass,
		); err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(dst)
	if err != nil {
		return err
	}

	files := []string{}
	for _, entry := range entries {
		files = append(files, filepath.Join(dst, entry.Name()))
	}

	fmt.Println("DRY_RUN", DRY_RUN, len(os.Getenv("DRY_RUN")))

	if DRY_RUN {
		fmt.Println("tag", tag, "files", files)
		return nil
	}

	if released {
		return sh.RunWithV(ctx, ENV_MAP, "gh", append([]string{"release", "upload", tag, "--clobber"}, files...)...)
	}
	return sh.RunWithV(ctx, ENV_MAP, "gh", append([]string{"release", "create", tag, "-n", tag, "-t", tag}, files...)...)
}

// filePaths retrieves all matched patterns file paths.
func filePaths(dir string, patterns ...string) ([]string, error) {
	result := []string{}
	for _, pattern := range patterns {
		files, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return result, err
		}
		result = append(result, files...)
	}
	return result, nil
}

// exportSecretKey exports keys from keyring.
func exportSecretKey(ctx context.Context) (string, string, error) {
	ring, err := os.CreateTemp(os.TempDir(), "keyring.*.gpg")
	if err != nil {
		return "", "", err
	}
	pass, err := os.CreateTemp(os.TempDir(), "pass.*.txt")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(pass.Name(), []byte(GPG_PASSPHRASE), 0600); err != nil {
		return "", "", err
	}
	return ring.Name(), pass.Name(), sh.Run(ctx,
		"gpg",
		"--batch",
		"--yes",
		"--pinentry-mode", "loopback",
		"--output", ring.Name(),
		"--passphrase", GPG_PASSPHRASE,
		"--export-secret-key", GPG_TRUSTEE)
}

// resolveDeps resolves dependencies of a chart.
func resolveDeps(ctx context.Context, chart string, metadata *chart.Metadata) error {
	if len(metadata.Dependencies) == 0 {
		return nil
	}

	for _, dep := range metadata.Dependencies {
		if err := sh.RunWithV(ctx, ENV_MAP, "helm", "repo", "add", dep.Name, dep.Repository, "--force-update"); err != nil {
			return err
		}
	}

	// Run helm dependency build for that chart. This step also does fetching updates.
	return sh.RunWithV(ctx, ENV_MAP, "helm", "dependency", "build", chart)
}
