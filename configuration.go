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
import "fmt"
import "crypto/sha256"
import log "github.com/Sirupsen/logrus"

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

// WebInterface respresents the webinterface of Kelvin.
type WebInterface struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

// LightSchedule represents the schedule for any given day for the associated lights.
type LightSchedule struct {
	Name                    string                  `json:"name"`
	AssociatedDeviceIDs     []int                   `json:"associatedDeviceIDs"`
	DefaultColorTemperature int                     `json:"defaultColorTemperature"`
	DefaultBrightness       int                     `json:"defaultBrightness"`
	BeforeSunrise           []TimedColorTemperature `json:"beforeSunrise"`
	AfterSunset             []TimedColorTemperature `json:"afterSunset"`
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
	ConfigurationFile string          `json:"-"`
	Hash              string          `json:"-"`
	Bridge            Bridge          `json:"bridge"`
	Location          Location        `json:"location"`
	WebInterface      WebInterface    `json:"webinterface"`
	Schedules         []LightSchedule `json:"schedules"`
}

// TimeStamp represents a parsed and validated TimedColorTemperature.
type TimeStamp struct {
	Time             time.Time
	ColorTemperature int
	Brightness       int
}

func (configuration *Configuration) initializeDefaults() {
	var bedTime TimedColorTemperature
	bedTime.Time = "22:00"
	bedTime.ColorTemperature = 2000
	bedTime.Brightness = 60

	var tvTime TimedColorTemperature
	tvTime.Time = "20:00"
	tvTime.ColorTemperature = 2300
	tvTime.Brightness = 80

	var wakeupTime TimedColorTemperature
	wakeupTime.Time = "4:00"
	wakeupTime.ColorTemperature = 2000
	wakeupTime.Brightness = 60

	var defaultSchedule LightSchedule
	defaultSchedule.Name = "default"
	defaultSchedule.AssociatedDeviceIDs = []int{}
	defaultSchedule.DefaultColorTemperature = 2750
	defaultSchedule.DefaultBrightness = 100
	defaultSchedule.AfterSunset = []TimedColorTemperature{tvTime, bedTime}
	defaultSchedule.BeforeSunrise = []TimedColorTemperature{wakeupTime}

	configuration.Schedules = []LightSchedule{defaultSchedule}

	var webinterface WebInterface
	webinterface.Enabled = false
	webinterface.Port = 8080
	configuration.WebInterface = webinterface
}

// InitializeConfiguration creates and returns an initialized
// configuration.
// If no configuration can be found on disk, one with default values
// will be created.
func InitializeConfiguration() (Configuration, error) {
	var configuration Configuration
	configuration.ConfigurationFile = "config.json"
	if configuration.Exists() {
		err := configuration.Read()
		if err != nil {
			return configuration, err
		}
		log.Printf("⚙ Configuration %v loaded", configuration.ConfigurationFile)
	} else {
		// write default config to disk
		configuration.initializeDefaults()
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

	if !configuration.HasChanged() {
		log.Debugf("Configuration hasn't changed. Omitting write.")
		return nil
	}
	log.Debugf("Configuration changed. Saving...")
	json, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(configuration.ConfigurationFile, json, 0644)
	if err != nil {
		return err
	}

	configuration.Hash = configuration.HashValue()
	log.Debugf("Updated configuration hash")
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

	if len(configuration.Schedules) == 0 {
		log.Warningf("⚙ Your current configuration doesn't contain any schedules! Generating default schedule...")
		err := configuration.backup()
		if err != nil {
			log.Warningf("Could not create backup: %v", err)
		} else {
			log.Printf("Configuration backup created.")
			configuration.initializeDefaults()
			log.Printf("Default schedule created.")
			configuration.Write()
		}
	}
	configuration.Hash = configuration.HashValue()
	log.Debugf("Updated configuration hash.")

	// Migrate to new timestamp format
	for scheduleIndex := range configuration.Schedules {
		for beforeTimestampIndex := range configuration.Schedules[scheduleIndex].BeforeSunrise {
			t, err := migrateTimestampFormat(configuration.Schedules[scheduleIndex].BeforeSunrise[beforeTimestampIndex].Time)
			if err != nil {
				log.Warningf(err.Error())
			} else {
				configuration.Schedules[scheduleIndex].BeforeSunrise[beforeTimestampIndex].Time = t
			}
		}
		for afterTimestampIndex := range configuration.Schedules[scheduleIndex].AfterSunset {
			t, err := migrateTimestampFormat(configuration.Schedules[scheduleIndex].AfterSunset[afterTimestampIndex].Time)
			if err != nil {
				log.Warningf(err.Error())
			} else {
				configuration.Schedules[scheduleIndex].AfterSunset[afterTimestampIndex].Time = t
			}
		}
	}

	// Migration: Disable webinterface
	if configuration.WebInterface.Port == 0 {
		log.Printf("Migrating webinterface settings...")
		configuration.WebInterface.Enabled = false
		configuration.WebInterface.Port = 8083
	}
	configuration.Write()
	return nil
}

func (configuration *Configuration) lightScheduleForDay(light int, date time.Time) (Schedule, error) {
	// initialize schedule with end of day
	var schedule Schedule
	yr, mth, dy := date.Date()
	schedule.endOfDay = time.Date(yr, mth, dy, 23, 59, 59, 59, date.Location())

	var lightSchedule LightSchedule
	found := false
	for _, candidate := range configuration.Schedules {
		if containsInt(candidate.AssociatedDeviceIDs, light) {
			lightSchedule = candidate
			found = true
			break
		}
	}

	if !found {
		return schedule, fmt.Errorf("Light %d is not associated with any schedule in configuration", light)
	}

	schedule.sunrise = TimeStamp{CalculateSunrise(date, configuration.Location.Latitude, configuration.Location.Longitude), lightSchedule.DefaultColorTemperature, lightSchedule.DefaultBrightness}
	schedule.sunset = TimeStamp{CalculateSunset(date, configuration.Location.Latitude, configuration.Location.Longitude), lightSchedule.DefaultColorTemperature, lightSchedule.DefaultBrightness}

	// Before sunrise candidates
	schedule.beforeSunrise = []TimeStamp{}
	for _, candidate := range lightSchedule.BeforeSunrise {
		timestamp, err := candidate.AsTimestamp(date)
		if err != nil {
			log.Printf(err.Error())
			continue
		}
		schedule.beforeSunrise = append(schedule.beforeSunrise, timestamp)
	}

	// After sunset candidates
	schedule.afterSunset = []TimeStamp{}
	for _, candidate := range lightSchedule.AfterSunset {
		timestamp, err := candidate.AsTimestamp(date)
		if err != nil {
			log.Printf(err.Error())
			continue
		}
		schedule.afterSunset = append(schedule.afterSunset, timestamp)
	}

	return schedule, nil
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

// HasChanged will detect changes to the configuration struct.
func (configuration *Configuration) HasChanged() bool {
	if configuration.Hash == "" {
		return true
	}
	return configuration.HashValue() != configuration.Hash
}

// HashValue will calculate a SHA256 hash of the configuration struct.
func (configuration *Configuration) HashValue() string {
	json, _ := json.Marshal(configuration)
	return fmt.Sprintf("%x", sha256.Sum256(json))
}

// AsTimestamp parses and validates a TimedColorTemperature and returns
// a corresponding TimeStamp.
func (color *TimedColorTemperature) AsTimestamp(referenceTime time.Time) (TimeStamp, error) {
	layout := "15:04"
	t, err := time.Parse(layout, color.Time)
	if err != nil {
		return TimeStamp{time.Now(), color.ColorTemperature, color.Brightness}, err
	}
	yr, mth, day := referenceTime.Date()
	targetTime := time.Date(yr, mth, day, t.Hour(), t.Minute(), t.Second(), 0, referenceTime.Location())

	return TimeStamp{targetTime, color.ColorTemperature, color.Brightness}, nil
}

func (configuration *Configuration) backup() error {
	backupFilename := configuration.ConfigurationFile + "_" + time.Now().Format("01022006")
	log.Debugf("Moving configuration to %s.", backupFilename)
	return os.Rename(configuration.ConfigurationFile, backupFilename)
}

func migrateTimestampFormat(timestamp string) (string, error) {
	// Check for old format and convert
	layout := "3:04PM"
	t, err := time.Parse(layout, timestamp)
	if err == nil {
		log.Debugf("Migrating old timestamp %s to %s", timestamp, t.Format("15:04"))
		return t.Format("15:04"), nil
	}

	// Already new format? Return unchanged
	layout = "15:04"
	t, err = time.Parse(layout, timestamp)
	if err == nil {
		return timestamp, nil
	}

	return "", fmt.Errorf("Invalid timestamp format: %s", timestamp)
}
