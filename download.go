// MIT License
//
// Copyright (c) 2018 Stefan Wichmann
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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
)

func downloadLatestReleaseInfo(url string) (releaseName string, assetURL string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var data interface{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return "", "", err
	}

	releaseInfo := data.(map[string]interface{})
	var name string
	if releaseInfo["tag_name"] != nil {
		name = releaseInfo["tag_name"].(string)
	} else if releaseInfo["name"] != nil {
		name = releaseInfo["name"].(string)
	} else {
		return "", "", errors.New("No releases available")
	}

	releaseAssets := releaseInfo["assets"].([]interface{})
	for _, asset := range releaseAssets {
		jsonAsset := asset.(map[string]interface{})

		match, url := assetMatchesPlattform(jsonAsset)
		if match {
			return name, url, nil
		}
	}

	return "", "", errors.New("No matching release found")
}

func assetMatchesPlattform(asset map[string]interface{}) (bool, string) {
	// match content type
	contentType := asset["content_type"].(string)
	if !(strings.Contains(contentType, "application/gzip") || strings.Contains(contentType, "application/zip")) {
		return false, ""
	}

	// match os and arch
	os := runtime.GOOS
	plattform := runtime.GOARCH
	assetName := asset["name"].(string)

	if !strings.Contains(assetName, os) || !strings.Contains(assetName, plattform) {
		return false, ""
	}

	// special case for arm64 vs arm, skip arm64 builds
	if plattform == "arm" && strings.Contains(assetName, "arm64") {
		return false, ""
	} 
	
	// match file extension
	if !(strings.Contains(assetName, "zip") || strings.Contains(assetName, "tar.gz")) {
		return false, ""
	}

	return true, asset["browser_download_url"].(string)
}

func downloadReleaseArchive(url string) (archive string, err error) {
	// Create the tempfile in default location
	out, err := ioutil.TempFile("", "update")
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

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(out.Name())
		return "", err
	}

	return out.Name(), nil
}
