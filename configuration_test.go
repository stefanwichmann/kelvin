package main

import (
	"path/filepath"
	"testing"
)

// TestFreshConfigurationKeepsUserEdits pins the migration invariant: a
// freshly generated configuration carries the latest version, so reading
// it back never runs a migration that overrides user edits (issue #130).
func TestFreshConfigurationKeepsUserEdits(t *testing.T) {
	var c Configuration
	c.ConfigurationFile = filepath.Join(t.TempDir(), "config.json")
	c.initializeDefaults()
	c.Schedules[0].EnableWhenLightsAppear = false
	if err := c.Write(); err != nil {
		t.Fatalf("could not write configuration: %v", err)
	}

	var read Configuration
	read.ConfigurationFile = c.ConfigurationFile
	if err := read.Read(); err != nil {
		t.Fatalf("could not read configuration: %v", err)
	}
	if read.Schedules[0].EnableWhenLightsAppear {
		t.Fatalf("migration overrode enableWhenLightsAppear from false to true")
	}
	if read.Version != latestConfigurationVersion {
		t.Fatalf("fresh configuration has version %d, want %d", read.Version, latestConfigurationVersion)
	}
}

func TestReadOK(t *testing.T) {
	correctfiles := []string{
		"testdata/config-example.json",
		"testdata/config-example.yaml",
	}
	for _, testFile := range correctfiles {
		c := Configuration{}
		c.ConfigurationFile = testFile
		err := c.Read()
		if err != nil {
			t.Fatalf("Could not read correct configuration file : %v with error : %v", c.ConfigurationFile, err)
		}
	}
}

func TestReadError(t *testing.T) {
	wrongfiles := []string{
		"",          // no file passed
		"testdata/", // not a regular file
		"testdata/config-bad-wrongFormat.json",
		"testdata/config-bad-wrongFormat.yaml",
	}
	for _, testFile := range wrongfiles {
		c := Configuration{}
		c.ConfigurationFile = testFile
		err := c.Read()
		if err == nil {
			t.Errorf("reading [%v] file should return an error", c.ConfigurationFile)
		}
	}
}

func TestWriteOK(t *testing.T) {
	correctfiles := []string{
		"testdata/config-example.json",
		"testdata/config-example.yaml",
	}
	for _, testFile := range correctfiles {
		c := Configuration{}
		c.ConfigurationFile = testFile
		_ = c.Read()
		c.Hash = ""
		err := c.Write()
		if err != nil {
			t.Errorf("Could not write configuration to correct file : %v", c.ConfigurationFile)
		}
	}
}
