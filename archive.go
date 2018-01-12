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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	log "github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func extractBinaryFromZipArchive(archiveFile string, binaryName string, destinationFolder string) (binaryFile string, err error) {
	r, err := zip.OpenReader(archiveFile)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Iterate through the files in the archive
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			// skip directories
			continue
		} else {
			filename := filepath.Base(f.Name)
			dir := filepath.Dir(f.Name)
			if filename == filepath.Base(binaryName) {
				archiveDirs := strings.Split(dir, "/")
				if len(archiveDirs) > 1 {
					// Don't consider nested files with binaryName
					continue
				}

				log.Debugf("Found candidate %s in directory %s\n", filename, dir)
				out, err := ioutil.TempFile(destinationFolder, filepath.Base(binaryName))
				if err != nil {
					return "", err
				}
				defer out.Close()

				rc, err := f.Open()
				if err != nil {
					return "", err
				}
				defer rc.Close()

				_, err = io.Copy(out, rc)
				if err != nil {
					out.Close()
					rc.Close()
					os.Remove(out.Name())
					return "", err
				}
				log.Debugf("Extracted binary %v to file %v\n", filename, out.Name())
				return out.Name(), nil
			}
		}
	}

	return "", errors.New("Binary not found in archive")
}

func extractBinaryFromTarArchive(archiveFile string, binaryName string, destinationFolder string) (binaryFile string, err error) {
	reader, err := os.Open(archiveFile)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	gr, err := gzip.NewReader(reader)
	if err != nil {
		return "", err
	}

	tr := tar.NewReader(gr)

	for {
		f, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			log.Fatalln(err)
		}

		filename := filepath.Base(f.Name)
		dir := filepath.Dir(f.Name)
		if filename == filepath.Base(binaryName) {
			archiveDirs := strings.Split(dir, "/")
			if len(archiveDirs) > 1 {
				// Don't consider nested files with binaryName
				continue
			}

			log.Debugf("Found candidate %s in directory %s\n", filename, dir)
			out, err := ioutil.TempFile(destinationFolder, filepath.Base(binaryName))
			if err != nil {
				return "", err
			}
			defer out.Close()
			_, err = io.Copy(out, tr)
			if err != nil {
				out.Close()
				os.Remove(out.Name())
				return "", err
			}

			log.Debugf("Extracted binary %v to file %v\n", filename, out.Name())
			return out.Name(), nil
		}
	}

	return "", errors.New("Binary not found in archive")
}
