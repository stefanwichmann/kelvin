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
import log "github.com/sirupsen/logrus"

// Interval represents a time range of one day with
// the given start and end configurations.
type Interval struct {
	Start TimeStamp
	End   TimeStamp
}

func (interval *Interval) calculateLightStateInInterval(timestamp time.Time) LightState {
	// Timestamp before interval
	if timestamp.Before(interval.Start.Time) {
		return LightState{interval.Start.ColorTemperature, interval.Start.Brightness}
	}

	// Timestamp after interval
	if timestamp.After(interval.End.Time) {
		return LightState{interval.End.ColorTemperature, interval.End.Brightness}
	}

	// Calculate regular progress inside interval
	intervalDuration := interval.End.Time.Sub(interval.Start.Time)
	intervalProgress := timestamp.Sub(interval.Start.Time)
	percentProgress := intervalProgress.Minutes() / intervalDuration.Minutes()

	targetColorTemperature := interval.End.ColorTemperature
	if interval.Start.ColorTemperature != -1 && interval.End.ColorTemperature != -1 {
		colorTemperatureDiff := interval.End.ColorTemperature - interval.Start.ColorTemperature
		colorTemperaturePercentageValue := float64(colorTemperatureDiff) * percentProgress
		targetColorTemperature = interval.Start.ColorTemperature + int(colorTemperaturePercentageValue)
	}

	targetBrightness := interval.End.Brightness
	if interval.Start.Brightness != -1 && interval.End.Brightness != -1 {
		brightnessDiff := interval.End.Brightness - interval.Start.Brightness
		brightnessPercentageValue := float64(brightnessDiff) * percentProgress
		targetBrightness = interval.Start.Brightness + int(brightnessPercentageValue)
	}

	lightstate := LightState{targetColorTemperature, targetBrightness}
	if !lightstate.isValid() {
		log.Warningf("Validation failed in calculateLightStateInInterval")
	}
	return lightstate
}
