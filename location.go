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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/btittelbach/astrotime"
)

// Geolocation represents a position on earth for which we can
// calculate sunrise and sunset times.
// If no location is configured Kelvin will try a geo IP lookup.
type Geolocation struct {
	Latitude  float64
	Longitude float64
}

// GeoAPIResponse respresents the result of a request to geolocationAPIURL.
type GeoAPIResponse struct {
	Location string `json:"loc"`
	City     string `json:"city"`
}

const geolocationAPIURL = "https://ipinfo.io/json"

// InitializeLocation creates and return a geolocation for the current system.
func InitializeLocation(configuration *Configuration) (Geolocation, error) {
	var location Geolocation
	if configuration.Location.Latitude == 0 || configuration.Location.Longitude == 0 {
		log.Println("🌍 Location not configured. Detecting by IP")
		err := location.updateByIP()
		if err != nil {
			return location, err
		}
		configuration.Location.Latitude = location.Latitude
		configuration.Location.Longitude = location.Longitude
	} else {
		location.Latitude = configuration.Location.Latitude
		location.Longitude = configuration.Location.Longitude
		log.Printf("🌍 Working with location %v, %v from configuration", location.Latitude, location.Longitude)
	}
	return location, nil
}

func (location *Geolocation) updateByIP() error {
	response, err := http.Get(geolocationAPIURL)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	var data GeoAPIResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	tokens := strings.Split(data.Location, ",")
	if len(data.Location) == 0 || len(tokens) != 2 {
		log.Warningf("🌍 Detection of geolocation seems to have failed... Please configure manually")
		return nil
	}

	location.Latitude, _ = strconv.ParseFloat(tokens[0], 32)
	location.Longitude, _ = strconv.ParseFloat(tokens[1], 32)
	log.Printf("🌍 Detected geolocation: %s (%.4f, %.4f)", data.City, location.Latitude, location.Longitude)

	return nil
}

// CalculateSunset calculates the sunset for the given day based on
// the configured position on earth.
func CalculateSunset(date time.Time, latitude float64, longitude float64) time.Time {
	// calculate start of day
	yr, mth, day := date.Date()
	startOfDay := time.Date(yr, mth, day, 0, 0, 0, 0, date.Location())

	return astrotime.CalcDusk(startOfDay, latitude, longitude, astrotime.GOLDEN_HOUR)
}

// CalculateSunrise calculates the sunrise for the given day based on
// the configured position on earth.
func CalculateSunrise(date time.Time, latitude float64, longitude float64) time.Time {
	// calculate start of day
	yr, mth, day := date.Date()
	startOfDay := time.Date(yr, mth, day, 0, 0, 0, 0, date.Location())

	return astrotime.CalcDawn(startOfDay, latitude, longitude, astrotime.GOLDEN_HOUR)
}
