package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// TestWriteFallsBackToInPlaceWhenRenameFails pins the single-file bind
// mount case: rename onto a file that is itself a mount point fails with
// EBUSY, and Write must land the content in place instead of failing
// (issue #135).
func TestWriteFallsBackToInPlaceWhenRenameFails(t *testing.T) {
	renameFile = func(oldpath, newpath string) error {
		return errors.New("device or resource busy")
	}
	defer func() { renameFile = os.Rename }()

	file := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(file, []byte(`{"version":1}`), 0600); err != nil {
		t.Fatalf("could not write configuration: %v", err)
	}

	var c Configuration
	c.ConfigurationFile = file
	c.initializeDefaults()
	if err := c.Write(); err != nil {
		t.Fatalf("write must fall back to an in-place write, got: %v", err)
	}

	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("could not read configuration back: %v", err)
	}
	if !strings.Contains(string(raw), "\"schedules\"") {
		t.Errorf("in-place fallback did not land the new content: %s", raw)
	}
	if _, err := os.Stat(file + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temp file left behind after fallback")
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatalf("could not stat configuration: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("configuration mode after fallback is %v, want 0600", info.Mode().Perm())
	}
}

// TestStartupToleratesUnpersistableConfiguration pins the fatal policy of
// issue #135: when the migration write-back cannot land at all, startup
// proceeds with the migrated configuration in memory. Only read failures
// abort startup.
func TestStartupToleratesUnpersistableConfiguration(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("file permissions do not bind root")
	}
	renameFile = func(oldpath, newpath string) error {
		return errors.New("device or resource busy")
	}
	defer func() { renameFile = os.Rename }()

	dir := t.TempDir()
	file := filepath.Join(dir, "config.json")
	v1 := `{"version":1,"webinterface":{"enabled":true,"port":8080},"schedules":[{"name":"default","associatedDeviceIDs":[1]}]}`
	if err := os.WriteFile(file, []byte(v1), 0400); err != nil {
		t.Fatalf("could not write configuration: %v", err)
	}
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatalf("could not make directory read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0700) })

	c, err := InitializeConfiguration(file, true)
	if err != nil {
		t.Fatalf("startup must tolerate a failed write-back, got: %v", err)
	}
	if c.Version != latestConfigurationVersion {
		t.Errorf("configuration not migrated in memory: version %d, want %d", c.Version, latestConfigurationVersion)
	}
}

// TestWriteStoresHashOfWrittenState pins the lost-update fix of issue
// #140: a mutation landing while Write persists an older state must
// leave HasChanged() true, or the next Write silently skips it and the
// change is lost on restart.
func TestWriteStoresHashOfWrittenState(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.json")
	var c Configuration
	c.ConfigurationFile = file
	c.initializeDefaults()

	renameFile = func(oldpath, newpath string) error {
		c.Schedules[0].DefaultBrightness = 42 // concurrent mutation mid-write
		return os.Rename(oldpath, newpath)
	}
	defer func() { renameFile = os.Rename }()

	if err := c.Write(); err != nil {
		t.Fatalf("could not write configuration: %v", err)
	}
	if !c.HasChanged() {
		t.Fatalf("mutation during write credited as persisted; the next write would skip it")
	}
}

// TestConcurrentWritesKeepFileParsable pins the torn-write fix (#140):
// writers share the temp path, so unserialized writes can rename a
// truncated temp file over config.json. The file must parse after every
// round of concurrent writes.
func TestConcurrentWritesKeepFileParsable(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.json")
	var a, b Configuration
	a.ConfigurationFile = file
	b.ConfigurationFile = file
	a.initializeDefaults()
	b.initializeDefaults()
	// Size skew between the two writers widens the tear window.
	b.Schedules[0].Name = strings.Repeat("b", 8192)

	for i := 0; i < 50; i++ {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); a.Hash = ""; _ = a.Write() }()
		go func() { defer wg.Done(); b.Hash = ""; _ = b.Write() }()
		wg.Wait()

		var read Configuration
		read.ConfigurationFile = file
		if err := read.Read(); err != nil {
			t.Fatalf("configuration torn by concurrent writes on round %d: %v", i, err)
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
