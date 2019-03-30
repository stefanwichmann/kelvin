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

import log "github.com/Sirupsen/logrus"

// LightState represents a light configuration.
// It can be read from or written to the physical lights.
type LightState struct {
	ColorTemperature int `yaml:"colorTemperature"`
	Brightness       int `yaml:"brightness"`
}

func (lightstate *LightState) isValid() bool {
	valid := true
	// Validate Brightness
	if lightstate.Brightness != 0 && lightstate.Brightness != -1 && (lightstate.Brightness < 0 || lightstate.Brightness > 100) {
		log.Warningf("Validation: Invalid Brightness in %+v", lightstate)
		valid = false
	}

	// Validate ColorTemperature
	if lightstate.ColorTemperature != 0 && lightstate.ColorTemperature != -1 && (lightstate.ColorTemperature < 2000 || lightstate.ColorTemperature > 6500) {
		log.Warningf("Validation: Invalid ColorTemperature in %+v", lightstate)
		valid = false
	}

	return valid
}

func (lightstate *LightState) equals(l LightState) bool {
	if lightstate.ColorTemperature != l.ColorTemperature {
		return false
	}
	if lightstate.Brightness != l.Brightness {
		return false
	}
	return true
}
