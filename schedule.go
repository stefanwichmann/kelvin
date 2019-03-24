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
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
)

// Schedule represents all relevants timestamps of one day.
// Kelvin will calculate all light states based on the intervals
// between this timestamps.
type Schedule struct {
	endOfDay               time.Time
	beforeSunrise          []TimeStamp
	sunrise                TimeStamp
	sunset                 TimeStamp
	afterSunset            []TimeStamp
	enableWhenLightsAppear bool
}

func (schedule *Schedule) currentInterval(timestamp time.Time) (Interval, error) {
	// check if timestamp respresents the current day
	if timestamp.After(schedule.endOfDay) {
		return Interval{TimeStamp{time.Now(), 0, 0}, TimeStamp{time.Now(), 0, 0}}, fmt.Errorf("No current interval as the requested timestamp (%v) lays after the end of the current schedule (%v)", timestamp, schedule.endOfDay)
	}

	// if we are between todays sunrise and sunset, return daylight interval
	if timestamp.After(schedule.sunrise.Time) && timestamp.Before(schedule.sunset.Time) {
		return Interval{schedule.sunrise, schedule.sunset}, nil
	}

	var before, after TimeStamp
	// Before sunrise
	if timestamp.Before(schedule.sunrise.Time) {
		yr, mth, dy := timestamp.Date()
		startOfDay := TimeStamp{time.Date(yr, mth, dy, 0, 0, 0, 0, timestamp.Location()), -1, -1}
		candidates := append(schedule.beforeSunrise, startOfDay, schedule.sunrise)

		before, after = findTargetTimes(timestamp, candidates)

		// fix dummy values
		if before.ColorTemperature == -1 && before.Brightness == -1 {
			before.ColorTemperature = after.ColorTemperature
			before.Brightness = after.Brightness
		}

		return Interval{before, after}, nil
	}

	// After sunset
	if timestamp.After(schedule.sunset.Time) {
		yr, mth, dy := timestamp.Date()
		endOfDay := TimeStamp{time.Date(yr, mth, dy, 23, 59, 59, 0, timestamp.Location()), -1, -1}
		candidates := append(schedule.afterSunset, endOfDay, schedule.sunset)

		before, after = findTargetTimes(timestamp, candidates)
	}

	// fix dummy values
	if after.ColorTemperature == -1 && after.Brightness == -1 {
		after.ColorTemperature = before.ColorTemperature
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
