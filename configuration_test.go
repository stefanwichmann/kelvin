package main

import (
	"testing"
)

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
