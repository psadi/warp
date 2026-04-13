package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const repo = "psadi/warp"

func selfUpgrade() error {
	osType := runtime.GOOS
	arch := runtime.GOARCH

	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	}

	downloadURL, version, err := getLatestRelease(osType, arch)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	currentVersion := getCurrentVersion()
	if currentVersion == version {
		fmt.Println("Already on latest version:", version)
		return nil
	}

	fmt.Printf("Upgrading from %s to %s...\n", currentVersion, version)

	tmpDir, err := os.MkdirTemp("", "warp-upgrade")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tarPath := filepath.Join(tmpDir, "warp.tar.gz")
	if err := downloadFile(downloadURL, tarPath); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	binaryPath, err := extractBinary(tarPath, tmpDir, osType, arch)
	if err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get self path: %w", err)
	}

	if err := os.Rename(binaryPath, selfPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	os.Chmod(selfPath, 0755)

	fmt.Println("Upgraded successfully!")
	return nil
}

func getCurrentVersion() string {
	version := "dev"
	if data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".local/share/warp/version")); err == nil {
		version = strings.TrimSpace(string(data))
	}
	return version
}

func getLatestRelease(osType, arch string) (downloadURL, version string, err error) {
	client := &http.Client{}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = &http.Client{}
	}

	req, _ := http.NewRequest("GET", url, nil)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var result struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	version = strings.TrimPrefix(result.TagName, "v")
	assetName := fmt.Sprintf("warp-%s-%s-%s.tar.gz", version, osType, arch)

	if osType == "darwin" {
		assetName = fmt.Sprintf("warp-%s-darwin-amd64.tar.gz", version)
	}

	for _, asset := range result.Assets {
		if asset.Name == assetName {
			return asset.BrowserDownloadURL, version, nil
		}
	}

	return "", "", fmt.Errorf("no asset found for %s", assetName)
}

func downloadFile(url, dest string) error {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractBinary(tarPath, destDir, osType, arch string) (string, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var binaryName string

	if osType == "windows" {
		binaryName = fmt.Sprintf("warp-%s-%s.exe", osType, arch)
	} else {
		binaryName = fmt.Sprintf("warp-%s-%s", osType, arch)
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if filepath.Base(header.Name) == binaryName {
			out, err := os.Create(filepath.Join(destDir, binaryName))
			if err != nil {
				return "", err
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return "", err
			}
			return filepath.Join(destDir, binaryName), nil
		}
	}

	return "", fmt.Errorf("binary not found in archive")
}
