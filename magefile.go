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
var GH_TOKEN = os.Getenv("GH_TOKEN")
var GPG_PASSPHRASE = os.Getenv("GPG_PASSPHRASE")
var GPG_TRUSTEE = "trustee@tetrate-istio-subscription.iam.gserviceaccount.com"
var ENV_MAP = map[string]string{
	"GH_TOKEN": GH_TOKEN,
	"TZ":       "UTC",
}

// PackIstio packs a versioned Istio Helm chart, for example: 1.16.6-tetrate-v0.
func PackIstio(ctx context.Context) error {
	// Export keyring.
	ring, pass, err := exportSecretKey(ctx)
	if err != nil {
		return err
	}

	dir := filepath.Join("charts", "istio")
	versions, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, version := range versions {
		if err := packVersionedIstio(ctx, version.Name(), ring, pass); err != nil {
			return err
		}
	}
	return nil
}

func PackAddons(ctx context.Context) error {
	// Export keyring.
	ring, pass, err := exportSecretKey(ctx)
	if err != nil {
		return err
	}

	dir := filepath.Join("charts", "addons")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := packChart(ctx, dir, entry.Name(), ring, pass); err != nil {
			return err
		}
	}
	return nil
}

func PackDemos(ctx context.Context) error {
	// Export keyring.
	ring, pass, err := exportSecretKey(ctx)
	if err != nil {
		return err
	}

	dir := filepath.Join("charts", "demos")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := packChart(ctx, dir, entry.Name(), ring, pass); err != nil {
			return err
		}
	}
	return nil
}

func PackSystem(ctx context.Context) error {
	// Export keyring.
	ring, pass, err := exportSecretKey(ctx)
	if err != nil {
		return err
	}

	dir := filepath.Join("charts", "system")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := packChart(ctx, dir, entry.Name(), ring, pass); err != nil {
			return err
		}
	}
	return nil
}

// Index generates Helm charts index, and optionally merge with existing index.yaml.
// URL gives prefix for the tarballs. For example: https://github.com/tetratelabs/legacy-charts/releases/download.
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

func packChart(ctx context.Context, dir, name, ring, pass string) error {
	charts, err := chartYAMLs(dir, "Chart.yaml", "*/Chart.yaml", "*/*/Chart.yaml", "*/*/*/Chart.yaml", "*/*/*/*/Chart.yaml")
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
		err = sh.RunWithV(ctx, ENV_MAP, "gh", "release", "view", tag, "-R", "tetratelabs/legacy-charts")
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
	}
	return nil
}

func packVersionedIstio(ctx context.Context, version, ring, pass string) error {
	tag := "istio-" + version

	// If already published, skip it unless FORCED.
	err := sh.RunWithV(ctx, ENV_MAP, "gh", "release", "view", tag, "-R", "tetratelabs/legacy-charts")
	released := err == nil
	if released && !FORCED {
		fmt.Printf("%s is already published\n", tag)
		return nil
	}

	dir := filepath.Join("charts", "istio", version)
	dst := filepath.Join("dist", version)
	_ = os.MkdirAll(dst, os.ModePerm)

	charts, err := chartYAMLs(dir, "Chart.yaml", "*/Chart.yaml", "*/*/Chart.yaml", "*/*/*/Chart.yaml", "*/*/*/*/Chart.yaml")
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

	if released {
		return sh.RunWithV(ctx, ENV_MAP, "gh", append([]string{"release", "upload", tag, "--clobber", "-R", "tetratelabs/legacy-charts"}, files...)...)
	}
	return sh.RunWithV(ctx, ENV_MAP, "gh", append([]string{"release", "create", tag, "-n", tag, "-t", tag, "-R", "tetratelabs/legacy-charts"}, files...)...)
}

func chartYAMLs(build string, patterns ...string) ([]string, error) {
	result := []string{}
	for _, pattern := range patterns {
		files, err := filepath.Glob(filepath.Join(build, pattern))
		if err != nil {
			return result, err
		}
		result = append(result, files...)
	}
	return result, nil
}

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
