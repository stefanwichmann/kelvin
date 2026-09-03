package main

import (
	"os"
	"path/filepath"
	"testing"
)

// tempCopy copies a testdata file into a temp directory, because Read()
// migrates and rewrites the file it loads — testdata must stay untouched.
func tempCopy(t *testing.T, source string) string {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("could not read %v: %v", source, err)
	}
	target := filepath.Join(t.TempDir(), filepath.Base(source))
	if err := os.WriteFile(target, data, 0600); err != nil {
		t.Fatalf("could not write %v: %v", target, err)
	}
	return target
}

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
		c.ConfigurationFile = tempCopy(t, testFile)
		err := c.Read()
		if err != nil {
			t.Fatalf("Could not read correct configuration file : %v with error : %v", c.ConfigurationFile, err)
		}
	}
}

// TestMigrationVersion1PreservesReachability pins the version 1 to 2
// migration: existing configurations keep their current reachability
// (listen on all interfaces) instead of silently moving to loopback,
// so an unattended fleet update never locks users out (issue #128).
func TestMigrationVersion1PreservesReachability(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.json")
	v1 := `{"version":1,"bridge":{"ip":"192.168.1.10","username":"user"},"webinterface":{"enabled":true,"port":8080},"schedules":[{"name":"default","associatedDeviceIDs":[1]}]}`
	if err := os.WriteFile(file, []byte(v1), 0600); err != nil {
		t.Fatalf("could not write configuration: %v", err)
	}

	var c Configuration
	c.ConfigurationFile = file
	if err := c.Read(); err != nil {
		t.Fatalf("could not read configuration: %v", err)
	}
	if c.Version != latestConfigurationVersion {
		t.Fatalf("configuration not migrated: version %d, want %d", c.Version, latestConfigurationVersion)
	}
	if c.WebInterface.ListenAddress != "0.0.0.0" {
		t.Errorf("migrated configuration must keep listening on all interfaces, got %q", c.WebInterface.ListenAddress)
	}
	if c.Schedules[0].EnableWhenLightsAppear {
		t.Errorf("version 1 to 2 migration must not touch schedule settings")
	}
}

// TestMigrationVersion1DisabledInterfaceGetsLoopback pins the counterpart:
// a configuration that never enabled the interface must not be granted all
// interfaces by the migration (issue #128 review).
func TestMigrationVersion1DisabledInterfaceGetsLoopback(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.json")
	v1 := `{"version":1,"webinterface":{"enabled":false,"port":8080},"schedules":[{"name":"default","associatedDeviceIDs":[1]}]}`
	if err := os.WriteFile(file, []byte(v1), 0600); err != nil {
		t.Fatalf("could not write configuration: %v", err)
	}

	var c Configuration
	c.ConfigurationFile = file
	if err := c.Read(); err != nil {
		t.Fatalf("could not read configuration: %v", err)
	}
	if c.WebInterface.ListenAddress != "127.0.0.1" {
		t.Errorf("disabled interface must migrate to loopback, got %q", c.WebInterface.ListenAddress)
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
		c.ConfigurationFile = tempCopy(t, testFile)
		_ = c.Read()
		c.Hash = ""
		err := c.Write()
		if err != nil {
			t.Errorf("Could not write configuration to correct file : %v", c.ConfigurationFile)
		}
	}
}
