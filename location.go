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

import "net/http"
import "io/ioutil"
import "encoding/json"
import "log"
import "github.com/btittelbach/astrotime"
import "time"

// Geolocation represents a position on earth for which we can
// calculate sunrise and sunset times.
// If no location is configured Kelvin will try a geo IP lookup.
type Geolocation struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	RegionCode  string  `json:"region_code"`
	RegionName  string  `json:"region_name"`
	City        string  `json:"city"`
	ZipCode     string  `json:"zip_code"`
	TimeZone    string  `json:"time_zone"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	MetroCode   float64 `json:"metro_code"`
}

const geolocationURL = "https://freegeoip.net/json/"

// InitializeLocation creates and return a geolocation for the current system.
func InitializeLocation(latitude float64, longitude float64) (Geolocation, error) {
	var location Geolocation
	if latitude == 0 || longitude == 0 {
		log.Println("🌍 Location not configured. Detecting by IP")
		err := location.updateByIP()
		if err != nil {
			return location, err
		}
	} else {
		location.Latitude = latitude
		location.Longitude = longitude
		log.Printf("🌍 Working with location: %v, %v from configuration\n", location.Latitude, location.Longitude)
	}
	return location, nil
}

func (location *Geolocation) updateByIP() error {
	resp, err := http.Get(geolocationURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data Geolocation
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	log.Printf("🌍 Detected location: %s, %s (%v, %v).\n", data.City, data.CountryName, data.Latitude, data.Longitude)
	location.Latitude = data.Latitude
	location.Longitude = data.Longitude
	return nil
}

// CalculateSunset calculates the sunset for the given day based on
// the configured position on earth.
func CalculateSunset(date time.Time, latitude float64, longitude float64) time.Time {
	// calculate start of day
	yr, mth, day := date.Date()
	startOfDay := time.Date(yr, mth, day, 0, 0, 0, 0, date.Location())

	return astrotime.NextSunset(startOfDay, latitude, longitude)
}

// CalculateSunrise calculates the sunrise for the given day based on
// the configured position on earth.
func CalculateSunrise(date time.Time, latitude float64, longitude float64) time.Time {
	// calculate start of day
	yr, mth, day := date.Date()
	startOfDay := time.Date(yr, mth, day, 0, 0, 0, 0, date.Location())

	return astrotime.NextSunrise(startOfDay, latitude, longitude)
}
