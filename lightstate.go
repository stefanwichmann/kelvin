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
	// if comparing light states always prefer color comparison
	if (lightstate.color[0] != 0 && lightstate.color[1] != 0) || (state.color[0] != 0 && state.color[1] != 0) {
		diffx := math.Abs(float64(lightstate.color[0] - state.color[0]))
		diffy := math.Abs(float64(lightstate.color[1] - state.color[1]))
		diffbri := math.Abs(float64(lightstate.brightness - state.brightness))

		if diffx < 0.001 && diffy < 0.001 && diffbri < 3 {
			return true
		}
		return false
	}

	if lightstate.colorTemperature != state.colorTemperature || lightstate.brightness != state.brightness {
		return false
	}
	return true
}

func (lightstate *LightState) convertValuesToHue() (int, []float32, int) {
	// color temperature
	if lightstate.colorTemperature > 6500 {
		lightstate.colorTemperature = 6500
	} else if lightstate.colorTemperature < 2000 {
		lightstate.colorTemperature = 2000
	}
	hueColor := (float64(1) / float64(lightstate.colorTemperature)) * float64(1000000)

	// brightness
	if lightstate.brightness > 100 {
		lightstate.brightness = 100
	} else if lightstate.brightness < 0 {
		lightstate.brightness = 0
	}
	hueBrightness := (float64(lightstate.brightness) / float64(100)) * float64(254)

	// map temperature to xy if not set
	x := lightstate.color[0]
	y := lightstate.color[1]
	if x == 0 || y == 0 {
		x, y = colorTemperatureToXYColor(lightstate.colorTemperature)
	}

	return int(hueColor), []float32{float32(x), float32(y)}, int(hueBrightness)
}

func lightStateFromHueValues(colorTemperature int, color []float32, brightness int) LightState {
	// color temperature
	newColorTemperature := int(float64(1000000) / float64(colorTemperature))

	if newColorTemperature > 6500 {
		newColorTemperature = 6500
	} else if newColorTemperature < 2000 {
		newColorTemperature = 2000
	}

	// color
	if len(color) != 2 || color[0] == 0 || color[1] == 0 {
		// color is not properly initialized. Since we need it
		// for state comparison we need to provide a valid state
		x, y := colorTemperatureToXYColor(newColorTemperature)
		color = []float32{float32(x), float32(y)}
	}

	// brightness
	newBrightness := (float64(brightness) / float64(254)) * float64(100)

	if newBrightness > 100 {
		newBrightness = 100
	} else if newBrightness < 0 {
		newBrightness = 0
	}

	return LightState{int(newColorTemperature), color, int(newBrightness)}
}
