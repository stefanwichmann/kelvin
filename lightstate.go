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

import (
	"math"
)

type LightState struct {
	colorTemperature int
	color            []float32
	brightness       int
}

func (self *LightState) equals(state LightState) bool {
	// if comparing light states always prefer color comparison
	if (self.color[0] != 0 && self.color[1] != 0) || (state.color[0] != 0 && state.color[1] != 0) {
		diffx := math.Abs(float64(self.color[0] - state.color[0]))
		diffy := math.Abs(float64(self.color[1] - state.color[1]))
		diffbri := math.Abs(float64(self.brightness - state.brightness))

		if diffx < 0.001 && diffy < 0.001 && diffbri < 3 {
			return true
		} else {
			return false
		}
	} else {
		if self.colorTemperature != state.colorTemperature || self.brightness != state.brightness {
			return false
		} else {
			return true
		}
	}
}

func (self *LightState) convertValuesToHue() (int, []float32, int) {
	// color temperature
	if self.colorTemperature > 6500 {
		self.colorTemperature = 6500
	} else if self.colorTemperature < 2000 {
		self.colorTemperature = 2000
	}
	hueColor := (float64(1) / float64(self.colorTemperature)) * float64(1000000)

	// brightness
	if self.brightness > 100 {
		self.brightness = 100
	} else if self.brightness < 0 {
		self.brightness = 0
	}
	hueBrightness := (float64(self.brightness) / float64(100)) * float64(254)

	// map temperature to xy if not set
	x := self.color[0]
	y := self.color[1]
	if x == 0 || y == 0 {
		x, y = colorTemperatureToXYColor(self.colorTemperature)
	}

	return int(hueColor), []float32{float32(x), float32(y)}, int(hueBrightness)
}

func lightStateFromHueValues(colorTemperature int, color []float32, brightness int) LightState {
	// color temperature
	newColorTemperature := (float64(1000000) / float64(colorTemperature))

	if newColorTemperature > 6500 {
		newColorTemperature = 6500
	} else if newColorTemperature < 2000 {
		newColorTemperature = 2000
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
