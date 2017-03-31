// MIT License
//
// Copyright (c) 2017 Stefan Wichmann
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

import "io/ioutil"
import "encoding/json"
import "os"
import "errors"
import "time"
import "log"

// Bridge respresents the hue bridge in your system.
type Bridge struct {
	IP       string `json:"ip"`
	Username string `json:"username"`
}

// Location represents the geolocation for which sunrise and sunset will be calculated.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// TimedColorTemperature represents a light configuration which will be
// reached at the given time.
type TimedColorTemperature struct {
	Time             string `json:"time"`
	ColorTemperature int    `json:"colorTemperature"`
	Brightness       int    `json:"brightness"`
}

// Configuration encapsulates all relevant parameters for Kelvin to operate.
type Configuration struct {
	ConfigurationFile       string                  `json:"-"`
	Bridge                  Bridge                  `json:"bridge"`
	Location                Location                `json:"location"`
	DefaultColorTemperature int                     `json:"defaultColorTemperature"`
	DefaultBrightness       int                     `json:"defaultBrightness"`
	BeforeSunrise           []TimedColorTemperature `json:"beforeSunrise"`
	AfterSunset             []TimedColorTemperature `json:"afterSunset"`
	IgnoredDeviceIDs        []int                   `json:"ignoredDeviceIDs"`
}

// TimeStamp represents a parsed and validated TimedColorTemperature.
type TimeStamp struct {
	Time             time.Time
	ColorTemperature int
	Brightness       int
}

func (configuration *Configuration) initializeDefaults() {
	var bridge Bridge
	bridge.IP = ""
	bridge.Username = ""

	var location Location
	location.Latitude = 0
	location.Longitude = 0

	var bedTime TimedColorTemperature
	bedTime.Time = "10:00PM"
	bedTime.ColorTemperature = 2000
	bedTime.Brightness = 60

	var tvTime TimedColorTemperature
	tvTime.Time = "8:00PM"
	tvTime.ColorTemperature = 2300
	tvTime.Brightness = 80

	var wakeupTime TimedColorTemperature
	wakeupTime.Time = "4:00AM"
	wakeupTime.ColorTemperature = 2000
	wakeupTime.Brightness = 60

	configuration.ConfigurationFile = "config.json"
	configuration.Bridge = bridge
	configuration.Location = location
	configuration.DefaultColorTemperature = 2750
	configuration.DefaultBrightness = 100
	configuration.AfterSunset = []TimedColorTemperature{tvTime, bedTime}
	configuration.BeforeSunrise = []TimedColorTemperature{wakeupTime}
	configuration.IgnoredDeviceIDs = []int{}
}

// InitializeConfiguration creates and returns an initialized
// configuration.
// If no configuration can be found on disk, one with default values
// will be created.
func InitializeConfiguration() (Configuration, error) {
	var configuration Configuration
	configuration.initializeDefaults()
	if configuration.Exists() {
		err := configuration.Read()
		if err != nil {
			return configuration, err
		}
		log.Printf("⚙ Configuration %v loaded", configuration.ConfigurationFile)
	} else {
		// write default config to disk
		err := configuration.Write()
		if err != nil {
			return configuration, err
		}
		log.Println("⚙ Default configuration generated")
	}
	return configuration, nil
}

// Write saves a configuration to disk.
func (configuration *Configuration) Write() error {
	if configuration.ConfigurationFile == "" {
		return errors.New("No configuration filename configured")
	}

	json, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(configuration.ConfigurationFile, json, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Read loads a configuration from disk.
func (configuration *Configuration) Read() error {
	if configuration.ConfigurationFile == "" {
		return errors.New("No configuration filename configured")
	}

	raw, err := ioutil.ReadFile(configuration.ConfigurationFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(raw, configuration)
	if err != nil {
		return err
	}

	return nil
}

// Exists return true if a configuration file is found on disk.
// False otherwise.
func (configuration *Configuration) Exists() bool {
	if configuration.ConfigurationFile == "" {
		return false
	}

	if _, err := os.Stat(configuration.ConfigurationFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// AsTimestamp parses and validates a TimedColorTemperature and returns
// a corresponding TimeStamp.
func (color *TimedColorTemperature) AsTimestamp(referenceTime time.Time) (TimeStamp, error) {
	layout := "3:04PM"
	t, err := time.Parse(layout, color.Time)
	if err != nil {
		return TimeStamp{time.Now(), color.ColorTemperature, color.Brightness}, err
	}
	yr, mth, day := referenceTime.Date()
	targetTime := time.Date(yr, mth, day, t.Hour(), t.Minute(), t.Second(), 0, referenceTime.Location())

	return TimeStamp{targetTime, color.ColorTemperature, color.Brightness}, nil
}
