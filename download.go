// MIT License
//
// # Copyright (c) 2018 Stefan Wichmann
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

// releaseInfo mirrors the fields kelvin reads from the GitHub release API.
// Typed decoding turns unexpected response shapes into errors instead of
// panics (issue #131).
type releaseInfo struct {
	TagName string         `json:"tag_name"`
	Name    string         `json:"name"`
	Assets  []releaseAsset `json:"assets"`
}

// releaseAsset represents one downloadable file of a GitHub release.
type releaseAsset struct {
	Name               string `json:"name"`
	ContentType        string `json:"content_type"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func downloadLatestReleaseInfo(url string) (releaseName string, assetURL string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("release info request failed: %s", resp.Status)
	}

	var release releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}

	name := release.TagName
	if name == "" {
		name = release.Name
	}
	if name == "" {
		return "", "", errors.New("no releases available")
	}

	for _, asset := range release.Assets {
		if assetMatchesPlattform(asset) {
			return name, asset.BrowserDownloadURL, nil
		}
	}

	return "", "", errors.New("no matching release found")
}

func assetMatchesPlattform(asset releaseAsset) bool {
	// match content type
	if !(strings.Contains(asset.ContentType, "application/gzip") || strings.Contains(asset.ContentType, "application/zip")) {
		return false
	}

	// match os and arch
	if !strings.Contains(asset.Name, runtime.GOOS) || !strings.Contains(asset.Name, runtime.GOARCH) {
		return false
	}

	// special case for arm64 vs arm, skip arm64 builds
	if runtime.GOARCH == "arm" && strings.Contains(asset.Name, "arm64") {
		return false
	}

	// match file extension
	return strings.Contains(asset.Name, "zip") || strings.Contains(asset.Name, "tar.gz")
}

func downloadReleaseArchive(url string) (archive string, err error) {
	// Create the tempfile in default location
	out, err := os.CreateTemp("", "update")
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		os.Remove(out.Name())
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		os.Remove(out.Name())
		return "", fmt.Errorf("archive download failed: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(out.Name())
		return "", err
	}

	return out.Name(), nil
}
