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

import "time"
import "fmt"
import log "github.com/Sirupsen/logrus"

func (configuration *Configuration) migrateToLatestVersion() {
	log.Debugf("⚙ Migrating configuration to latest version...")
	if configuration.Version == 0 {
		configuration.migrateVersion0()
	}
	log.Debugf("⚙ Migration of configuration complete")
}

func (configuration *Configuration) migrateVersion0() {
	log.Debugf("⚙ Migrating configuration version 0 to version 1...")

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
		log.Debugf("⚙ Migrating webinterface settings...")
		configuration.WebInterface.Enabled = false
		configuration.WebInterface.Port = 8080
	}

	// Migration: Automatic enable of kelvin
	for scheduleIndex := range configuration.Schedules {
		configuration.Schedules[scheduleIndex].EnableWhenLightsAppear = true
	}

	configuration.Version = 1
	log.Debugf("⚙ Migration to version 1 complete")
}

func migrateTimestampFormat(timestamp string) (string, error) {
	// Check for old format and convert
	layout := "3:04PM"
	t, err := time.Parse(layout, timestamp)
	if err == nil {
		log.Debugf("⚙ Migrating old timestamp %s to %s", timestamp, t.Format("15:04"))
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
