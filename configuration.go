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

type Bridge struct {
	IP       string `json:"ip"`
	Username string `json:"username"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type TimedColorTemperature struct {
	Time       string `json:"time"`
	Color      int    `json:"color"`
	Brightness int    `json:"brightness"`
}

type Configuration struct {
	ConfigurationFile       string                  `json:"-"`
	Bridge                  Bridge                  `json:"bridge"`
	Location                Location                `json:"location"`
	DefaultColorTemperature int                     `json:"defaultColorTemperature"`
	DefaultBrightness       int                     `json:"defaultBrightness"`
	AfterSunset             []TimedColorTemperature `json:"afterSunset"`
	BeforeSunrise           []TimedColorTemperature `json:"beforeSunrise"`
}

type TimeStamp struct {
	Time       time.Time
	Color      int
	Brightness int
}

func (self *Configuration) initializeDefaults() {
	var bridge Bridge
	bridge.IP = ""
	bridge.Username = ""

	var location Location
	location.Latitude = 0
	location.Longitude = 0

	var bedTime TimedColorTemperature
	bedTime.Time = "10:00PM"
	bedTime.Color = 2000
	bedTime.Brightness = 60

	var tvTime TimedColorTemperature
	tvTime.Time = "8:00PM"
	tvTime.Color = 5000
	tvTime.Brightness = 80

	var wakeupTime TimedColorTemperature
	wakeupTime.Time = "4:00AM"
	wakeupTime.Color = 2000
	wakeupTime.Brightness = 60

	self.ConfigurationFile = "config.json"
	self.Bridge = bridge
	self.Location = location
	self.DefaultColorTemperature = 4000
	self.DefaultBrightness = 100
	self.AfterSunset = []TimedColorTemperature{tvTime, bedTime}
	self.BeforeSunrise = []TimedColorTemperature{wakeupTime}
}

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

func (self *Configuration) Write() error {
	if self.ConfigurationFile == "" {
		return errors.New("No configuration filename configured.")
	}

	json, err := json.MarshalIndent(self, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(self.ConfigurationFile, json, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (self *Configuration) Read() error {
	if self.ConfigurationFile == "" {
		return errors.New("No configuration filename configured.")
	}

	raw, err := ioutil.ReadFile(self.ConfigurationFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(raw, self)
	if err != nil {
		return err
	}

	return nil
}

func (self *Configuration) Exists() bool {
	if self.ConfigurationFile == "" {
		return false
	}

	if _, err := os.Stat(self.ConfigurationFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func (self *TimedColorTemperature) AsTimestamp(referenceTime time.Time) (TimeStamp, error) {
	layout := "3:04PM"
	t, err := time.Parse(layout, self.Time)
	if err != nil {
		return TimeStamp{time.Now(), self.Color, self.Brightness}, err
	}
	yr, mth, day := referenceTime.Date()
	targetTime := time.Date(yr, mth, day, t.Hour(), t.Minute(), t.Second(), 0, referenceTime.Location())

	return TimeStamp{targetTime, self.Color, self.Brightness}, nil
}
