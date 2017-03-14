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

import "time"
import "log"
import "errors"

type Day struct {
	endOfDay      time.Time
	beforeSunrise []TimeStamp
	sunrise       TimeStamp
	sunset        TimeStamp
	afterSunset   []TimeStamp
}

func (self *Day) updateCyclic(configuration Configuration, location Geolocation, channel chan<- Interval) {
	self.updateForDay(time.Now(), configuration, location)

	for {
		currentInterval, err := self.currentInterval(time.Now())
		if err != nil {
			// should only happen when day has ended
			log.Println("NEW DAY?", err.Error())
			self.updateForDay(time.Now(), configuration, location)
		} else {
			channel <- currentInterval
			timeLeftInInterval := currentInterval.End.Time.Sub(time.Now())
			log.Printf("DAY - Time left in interval %v - %v: %v\n", currentInterval.Start.Time.Format("15:04"), currentInterval.End.Time.Format("15:04"), timeLeftInInterval)
			time.Sleep(timeLeftInInterval + (1 * time.Second))
		}
	}
}

func (self *Day) updateForDay(date time.Time, configuration Configuration, location Geolocation) {
	yr, mth, day := date.Date()
	log.Printf("Configuring intervals for %v\n", date.Format("Monday January 2 2006"))
	self.endOfDay = time.Date(yr, mth, day, 23, 59, 59, 59, date.Location())
	self.sunrise = TimeStamp{location.CalculateSunrise(date), configuration.DefaultColorTemperature, configuration.DefaultBrightness}
	self.sunset = TimeStamp{location.CalculateSunset(date), configuration.DefaultColorTemperature, configuration.DefaultBrightness}

	// Before sunrise candidates
	self.beforeSunrise = []TimeStamp{}
	for _, candidate := range configuration.BeforeSunrise {
		timestamp, err := candidate.AsTimestamp(date)
		if err != nil {
			log.Printf(err.Error())
			continue
		}
		self.beforeSunrise = append(self.beforeSunrise, timestamp)
	}

	// After sunset candidates
	self.afterSunset = []TimeStamp{}
	for _, candidate := range configuration.AfterSunset {
		timestamp, err := candidate.AsTimestamp(date)
		if err != nil {
			log.Printf(err.Error())
			continue
		}
		self.afterSunset = append(self.afterSunset, timestamp)
	}
}

func (self *Day) currentInterval(timestamp time.Time) (Interval, error) {
	// check if timestamp respresents the current day
	if timestamp.After(self.endOfDay) {
		return Interval{TimeStamp{time.Now(), 0, 0}, TimeStamp{time.Now(), 0, 0}}, errors.New("DAY - No current interval as the requested timestamp respresents a different day.")
	}

	// if we are between todays sunrise and sunset, return daylight interval
	if timestamp.After(self.sunrise.Time) && timestamp.Before(self.sunset.Time) {
		return Interval{self.sunrise, self.sunset}, nil
	}

	var before, after TimeStamp
	// Before sunrise
	if timestamp.Before(self.sunrise.Time) {
		yr, mth, day := timestamp.Date()
		startOfDay := TimeStamp{time.Date(yr, mth, day, 0, 0, 0, 0, timestamp.Location()), -1, -1}
		candidates := append(self.beforeSunrise, startOfDay, self.sunrise)

		before, after = findTargetTimes(timestamp, candidates)

		// fix dummy values
		if before.Color == -1 && before.Brightness == -1 {
			before.Color = after.Color
			before.Brightness = after.Brightness
		}

		return Interval{before, after}, nil
	}

	// After sunset
	if timestamp.After(self.sunset.Time) {
		yr, mth, day := timestamp.Date()
		endOfDay := TimeStamp{time.Date(yr, mth, day, 23, 59, 59, 0, timestamp.Location()), -1, -1}
		candidates := append(self.afterSunset, endOfDay, self.sunset)

		before, after = findTargetTimes(timestamp, candidates)
	}

	// fix dummy values
	if after.Color == -1 && after.Brightness == -1 {
		after.Color = before.Color
		after.Brightness = before.Brightness
	}

	return Interval{before, after}, nil
}

func findTargetTimes(timestamp time.Time, candidates []TimeStamp) (TimeStamp, TimeStamp) {
	beforeCandidate := TimeStamp{timestamp.AddDate(0, 0, -2), 0, 0}
	afterCandidate := TimeStamp{timestamp.AddDate(0, 0, 2), 0, 0}

	for _, candidate := range candidates {
		if candidate.Time.Before(timestamp) && candidate.Time.After(beforeCandidate.Time) {
			beforeCandidate = candidate
			continue
		}
		if candidate.Time.After(timestamp) && candidate.Time.Before(afterCandidate.Time) {
			afterCandidate = candidate
		}
	}

	if beforeCandidate.Time.Day() != timestamp.Day() || afterCandidate.Time.Day() != timestamp.Day() {
		log.Fatal("Could not find targetTime in candidates.")
	}

	return beforeCandidate, afterCandidate
}
