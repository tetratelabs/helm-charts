//go:build mage

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dio/sh"
)

var FORCED = len(os.Getenv("FORCED")) > 0
var GH_TOKEN = os.Getenv("GH_TOKEN")
var GPG_PASSPHRASE = os.Getenv("GPG_PASSPHRASE")
var GPG_TRUSTEE = "trustee@tetrate-istio-subscription.iam.gserviceaccount.com"

// PackIstio packs a versioned Istio Helm chart, for example: 1.16.6-tetrate-v0.
func PackIstio(ctx context.Context) error {
	dir := filepath.Join("charts", "istio")
	versions, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, version := range versions {
		if err := packVersioned(ctx, version.Name()); err != nil {
			return err
		}
	}
	return nil
}

// Index generates Helm charts index, and optionally merge with existing index.yaml.
// URL gives prefix for the tarballs. For example: https://github.com/tetratelabs/legacy-charts/releases/download.
func Index(ctx context.Context, url string) error {
	dir := filepath.Join("dist")
	indexYAML := filepath.Join("gh-pages", "index.yaml")
	args := []string{
		"repo",
		"index",
		dir,
		"--url", url,
	}
	if _, err := os.Stat(indexYAML); err == nil {
		args = append(args, "--merge", indexYAML)
	}
	return sh.Run(ctx, "helm", args...)
}

func packVersioned(ctx context.Context, version string) error {
	// Export keyring.
	ring, pass, err := exportSecretKey(ctx)
	if err != nil {
		return err
	}

	e := map[string]string{
		"GH_TOKEN": GH_TOKEN,
	}

	tag := "istio-" + version

	// If already published, skip it unless FORCED.
	err = sh.RunWithV(ctx, e, "gh", "release", "view", tag, "-R", "tetratelabs/legacy-charts")
	released := err == nil
	if released && !FORCED {
		fmt.Printf("%s is already published\n", tag)
		return nil
	}

	dir := filepath.Join("charts", "istio", tag)
	charts, err := chartYAMLs(dir, "Chart.yaml", "*/Chart.yaml", "*/*/Chart.yaml", "*/*/*/Chart.yaml", "*/*/*/*/Chart.yaml")
	if err != nil {
		return err
	}

	dst := filepath.Join("dist", version)
	for _, chart := range charts {
		_ = os.MkdirAll(dst, os.ModePerm)
		if err := sh.Run(ctx,
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
		return sh.RunWithV(ctx, e, "gh", append([]string{"release", "upload", tag, "--clobber", "-R", "tetratelabs/legacy-charts"}, files...)...)
	}
	return sh.RunWithV(ctx, e, "gh", append([]string{"release", "create", tag, "-n", tag, "-t", tag, "-R", "tetratelabs/legacy-charts"}, files...)...)
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
