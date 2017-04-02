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

import "math"

// LightState represents a light configuration.
// It can be read from or written to the physical lights.
type LightState struct {
	colorTemperature int
	color            []float32
	brightness       int
}

func (lightstate *LightState) equals(state LightState) bool {
	var sameColor = false
	var sameColorTemperature = false
	var sameBrightness = false

	// compare color values
	currentX := lightstate.color[0]
	currentY := lightstate.color[1]
	if currentX == 0 && currentY == 0 {
		// zero value implies ignore color
		sameColor = true
	} else {
		diffx := math.Abs(float64(currentX - state.color[0]))
		diffy := math.Abs(float64(currentY - state.color[1]))
		if diffx < 0.001 && diffy < 0.001 {
			sameColor = true
		}
	}

	// compare color temperature
	if lightstate.colorTemperature == 0 {
		// zero value implies ignore color temperature
		sameColorTemperature = true
	} else {
		diffTemperature := math.Abs(float64(lightstate.colorTemperature - state.colorTemperature))
		if diffTemperature < 3 {
			sameColorTemperature = true
		}
	}

	// compare brightness
	if lightstate.brightness == 0 {
		// zero value implies ignore brightness
		sameBrightness = true
	} else {
		diffBrightness := math.Abs(float64(lightstate.brightness - state.brightness))
		if diffBrightness < 3 {
			sameBrightness = true
		}
	}

	// check if equal and prefer same color over same color temperature
	if sameColor && sameBrightness {
		return true
	}
	if sameColorTemperature && sameBrightness {
		return true
	}
	return false
}

func (lightstate *LightState) convertValuesToHue() (int, []float32, int) {
	var hueColorTemperature = 0
	var hueBrightness = 0

	// color temperature
	if lightstate.colorTemperature != 0 {
		if lightstate.colorTemperature > 6500 {
			lightstate.colorTemperature = 6500
		} else if lightstate.colorTemperature < 2000 {
			lightstate.colorTemperature = 2000
		}
		hueColorTemperature = int((float64(1) / float64(lightstate.colorTemperature)) * float64(1000000))
	}

	// brightness
	if lightstate.brightness != 0 {
		if lightstate.brightness > 100 {
			lightstate.brightness = 100
		} else if lightstate.brightness < 0 {
			lightstate.brightness = 0
		}
		hueBrightness = int((float64(lightstate.brightness) / float64(100)) * float64(254))
	}

	// xy color should not need a mapping
	return hueColorTemperature, lightstate.color, hueBrightness
}

func lightStateFromHueValues(colorTemperature int, color []float32, brightness int) LightState {
	var stateColorTemperature = 0
	var stateColor = []float32{0, 0}
	var stateBrightness = 0

	// color temperature
	if colorTemperature != 0 {
		stateColorTemperature = int(float64(1000000) / float64(colorTemperature))
		if stateColorTemperature > 6500 {
			stateColorTemperature = 6500
		} else if stateColorTemperature < 2000 {
			stateColorTemperature = 2000
		}
	}

	// color
	if len(color) != 2 {
		// color is not properly initialized. Since we need it
		// for state comparison we need to provide a valid state
		x, y := colorTemperatureToXYColor(stateColorTemperature)
		stateColor = []float32{float32(x), float32(y)}
	} else {
		stateColor = color
	}

	// brightness
	if brightness != 0 {
		stateBrightness = int((float64(brightness) / float64(254)) * float64(100))
		if stateBrightness > 100 {
			stateBrightness = 100
		} else if stateBrightness < 0 {
			stateBrightness = 0
		}
	}
	return LightState{stateColorTemperature, stateColor, stateBrightness}
}
