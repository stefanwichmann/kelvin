// MIT License
//
// Copyright (c) 2019 Stefan Wichmann
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
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func containsString(slice []string, element string) bool {
	for _, current := range slice {
		if strings.ToLower(current) == strings.ToLower(element) {
			return true
		}
	}
	return false
}

func containsInt(slice []int, element int) bool {
	for _, current := range slice {
		if current == element {
			return true
		}
	}
	return false
}

func abs(value int) int {
	if value < 0 {
		return value * -1
	}
	return value
}

func toStringArray(data []int) []string {
	var stringArray []string
	for _, elem := range data {
		stringArray = append(stringArray, fmt.Sprintf("%d", elem))
	}
	return stringArray
}

func roundFloat(f float32, precision int) float32 {
	shift := math.Pow(10, float64(precision))
	return float32(math.Floor((float64(f)*shift)+.5) / shift)
}

func equalsFloat(a []float32, b []float32, maxDiff float32) bool {
	if len(a) != len(b) {
		return false
	}
	for index := 0; index < len(a); index++ {
		rounded := float32(math.Round(math.Abs(float64(a[index]-b[index]))/float64(maxDiff))) * maxDiff
		if rounded > maxDiff {
			return false
		}
	}
	return true
}

func equalsInt(a int, b int, maxDiff int) bool {
	if abs(a-b) > maxDiff {
		return false
	}
	return true
}

// Restart the running binary.
// All arguments, pipes and environment variables will
// be preserved.
func Restart() {
	binary := os.Args[0]
	args := []string{}
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}

	cmd := exec.Command(binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	cmd.Start()
	os.Exit(0)
}

func workingDirectory() string {
	ex, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(ex)
}

func absolutePath(filename string) string {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return filename
	}
	return abs
}

func durationUntilNextDay() time.Duration {
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 1, now.Location())
	endOfDay = endOfDay.Add(1 * time.Second)
	return time.Until(endOfDay)
}
