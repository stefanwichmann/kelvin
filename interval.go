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
import "errors"
import "fmt"

// Interval represents a time range of one day with
// the given start and end configurations.
type Interval struct {
	Start TimeStamp
	End   TimeStamp
}

func (interval *Interval) calculateLightStateInInterval(timestamp time.Time) (LightState, error) {
	if timestamp.Before(interval.Start.Time) || timestamp.After(interval.End.Time) {
		return LightState{0, []float32{0.0, 0.0}, 0}, errors.New(fmt.Sprintf("Timestamp %v is not suitable for interval %v - %v\n", timestamp, interval.Start.Time, interval.End.Time))
	}

	// Calculate regular progress inside interval
	intervalDuration := interval.End.Time.Sub(interval.Start.Time)
	intervalProgress := timestamp.Sub(interval.Start.Time)
	percentProgress := intervalProgress.Minutes() / intervalDuration.Minutes()

	colorTemperatureDiff := interval.End.ColorTemperature - interval.Start.ColorTemperature
	colorTemperaturePercentageValue := float64(colorTemperatureDiff) * percentProgress
	targetColorTemperature := interval.Start.ColorTemperature + int(colorTemperaturePercentageValue)

	brightnessDiff := interval.End.Brightness - interval.Start.Brightness
	brightnessPercentageValue := float64(brightnessDiff) * percentProgress
	targetBrightness := interval.Start.Brightness + int(brightnessPercentageValue)

	x, y := colorTemperatureToXYColor(targetColorTemperature)

	return LightState{targetColorTemperature, []float32{float32(x), float32(y)}, targetBrightness}, nil
}
